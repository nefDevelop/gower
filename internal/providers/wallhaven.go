package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"gower/internal/utils"
	"gower/pkg/models"
)

var WallhavenBaseURL = "https://wallhaven.cc/api/v1/search"

// WallhavenProvider implements the Provider interface for Wallhaven.cc.
type WallhavenProvider struct {
	APIKey      string
	RateLimiter *utils.RateLimiter
	Log         *utils.Logger
}

func (p *WallhavenProvider) log() *utils.Logger {
	if p.Log != nil {
		return p.Log
	}
	return utils.Log
}

func (p *WallhavenProvider) GetName() string {
	return "wallhaven"
}

func (p *WallhavenProvider) Search(ctx context.Context, query string, opts SearchOptions) ([]models.Wallpaper, error) {
	u, err := url.Parse(WallhavenBaseURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	if query != "" {
		q.Set("q", query)
	}
	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.Category != "" {
		q.Set("categories", opts.Category)
	}
	if opts.Sort != "" {
		q.Set("sorting", opts.Sort)
	}
	if opts.MinWidth > 0 || opts.MinHeight > 0 {
		w := opts.MinWidth
		h := opts.MinHeight
		if w <= 0 {
			w = 1
		}
		if h <= 0 {
			h = 1
		}
		q.Set("atleast", fmt.Sprintf("%dx%d", w, h))
	}
	if opts.AspectRatio != "" {
		q.Set("ratios", opts.AspectRatio)
	}
	if opts.Color != "" {
		q.Set("colors", opts.Color)
	}
	if p.APIKey != "" {
		q.Set("apikey", p.APIKey)
	}

	u.RawQuery = q.Encode()

	p.log().Debug("Wallhaven fetching: %s", u.String())

	if p.RateLimiter != nil {
		p.RateLimiter.Wait()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := utils.DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wallhaven api returned status: %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID         string   `json:"id"`
			Path       string   `json:"path"`
			Resolution string   `json:"resolution"`
			Ratio      string   `json:"ratio"`
			Colors     []string `json:"colors"`
			Thumbs     struct {
				Large    string `json:"large"`
				Original string `json:"original"`
				Small    string `json:"small"`
			} `json:"thumbs"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var wallpapers []models.Wallpaper
	for _, item := range result.Data {
		wp := models.Wallpaper{
			ID:        "wh_" + item.ID,
			URL:       item.Path,
			Thumbnail: item.Thumbs.Large, // Usamos 'large' para el thumbnail
			Source:    "wallhaven",
			Dimension: item.Resolution,
			Ratio:     item.Ratio,
			// Theme se deja vacío para que omitempty lo oculte hasta que se analice
		}
		wallpapers = append(wallpapers, wp)
	}

	return wallpapers, nil
}
