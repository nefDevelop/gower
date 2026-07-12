package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gower/internal/utils"
	"gower/pkg/models"

	"github.com/tidwall/gjson"
)

// GenericProvider implements a provider based on configuration.
type GenericProvider struct {
	Config      models.GenericProviderConfig
	RateLimiter *utils.RateLimiter
	Log         *utils.Logger
}

func (p *GenericProvider) log() *utils.Logger {
	if p.Log != nil {
		return p.Log
	}
	return utils.Log
}

func (p *GenericProvider) GetName() string {
	return p.Config.Name
}

func (p *GenericProvider) Search(ctx context.Context, query string, opts SearchOptions) ([]models.Wallpaper, error) {
	url := p.Config.APIURL
	url = strings.ReplaceAll(url, "{query}", query)
	url = strings.ReplaceAll(url, "{apikey}", p.Config.APIKey)

	p.log().Debug("Generic provider %s fetching: %s", p.Config.Name, url)

	if p.RateLimiter != nil {
		p.RateLimiter.Wait()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := utils.DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generic api returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	jsonStr := string(body)

	results := gjson.Get(jsonStr, p.Config.ResponseMapping.ResultsPath)
	if !results.Exists() || !results.IsArray() {
		return []models.Wallpaper{}, nil
	}

	var wallpapers []models.Wallpaper
	for _, item := range results.Array() {
		id := item.Get(p.Config.ResponseMapping.IDPath).String()
		imgURL := item.Get(p.Config.ResponseMapping.URLPath).String()

		thumb := imgURL
		if p.Config.ResponseMapping.ThumbnailPath != "" {
			t := item.Get(p.Config.ResponseMapping.ThumbnailPath).String()
			if t != "" {
				thumb = t
			}
		}

		fullID := fmt.Sprintf("%s-%s", p.Config.Name, id)

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        fullID,
			URL:       imgURL,
			Thumbnail: thumb,
			Source:    p.Config.Name,
		})
	}

	return wallpapers, nil
}
