package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	rawURL := p.Config.APIURL
	rawURL = strings.ReplaceAll(rawURL, "{query}", url.QueryEscape(query))
	rawURL = strings.ReplaceAll(rawURL, "{apikey}", url.QueryEscape(p.Config.APIKey))

	p.log().Debug("Generic provider %s fetching: %s", p.Config.Name, utils.RedactURL(rawURL))

	if err := waitRateLimiter(p.RateLimiter, ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := utils.DoRequest(ctx, utils.DefaultHTTPClient, req, 3) //nolint:bodyclose // cerrado con utils.CloseResponse
	if err != nil {
		return nil, err
	}
	defer utils.CloseResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("generic api returned status: %d", resp.StatusCode)
	}

	body, err := utils.ReadBodyLimit(resp.Body, utils.MaxJSONResponseBytes)
	if err != nil {
		return nil, err
	}

	results := gjson.GetBytes(body, p.Config.ResponseMapping.ResultsPath)
	if !results.Exists() || !results.IsArray() {
		return []models.Wallpaper{}, nil
	}

	limit := opts.Limit
	wallpapers := make([]models.Wallpaper, 0, len(results.Array()))
	for i, item := range results.Array() {
		if limit > 0 && len(wallpapers) >= limit {
			break
		}

		id := item.Get(p.Config.ResponseMapping.IDPath).String()
		imgURL := item.Get(p.Config.ResponseMapping.URLPath).String()
		if imgURL == "" {
			continue
		}
		if strings.HasPrefix(imgURL, "//") {
			imgURL = "https:" + imgURL
		}
		if !strings.HasPrefix(imgURL, "http://") && !strings.HasPrefix(imgURL, "https://") {
			p.log().Debug("Generic provider %s: skipping item with non-http URL %q", p.Config.Name, imgURL)
			continue
		}
		if id == "" {
			id = fmt.Sprintf("%d", i)
		}

		thumb := imgURL
		if p.Config.ResponseMapping.ThumbnailPath != "" {
			if t := item.Get(p.Config.ResponseMapping.ThumbnailPath).String(); t != "" {
				thumb = t
			}
		}

		dimension := ""
		if p.Config.ResponseMapping.DimensionPath != "" {
			dimension = item.Get(p.Config.ResponseMapping.DimensionPath).String()
		}

		fullID := fmt.Sprintf("%s-%s", p.Config.Name, id)
		if opts.ExcludeIDs != nil && opts.ExcludeIDs[fullID] {
			continue
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        fullID,
			URL:       imgURL,
			Thumbnail: thumb,
			Source:    p.Config.Name,
			Dimension: dimension,
		})
	}

	return wallpapers, nil
}
