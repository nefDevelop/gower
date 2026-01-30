package providers

import (
	"fmt"
	"gower/pkg/models"
	"io"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
)

// GenericProvider implements a provider based on configuration.
type GenericProvider struct {
	Config models.GenericProviderConfig
}

func (p *GenericProvider) GetName() string {
	return p.Config.Name
}

func (p *GenericProvider) Search(query string, opts SearchOptions) ([]models.Wallpaper, error) {
	url := p.Config.APIURL
	url = strings.ReplaceAll(url, "{query}", query)
	url = strings.ReplaceAll(url, "{apikey}", p.Config.APIKey)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
