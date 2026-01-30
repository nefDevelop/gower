package providers

import (
	"encoding/json"
	"fmt"
	"gower/internal/utils"
	"gower/pkg/models"
	"net/http"
	"net/url"
	"strconv"
)

// WallhavenProvider implements the Provider interface for Wallhaven.cc.
type WallhavenProvider struct {
	APIKey string
}

func (p *WallhavenProvider) GetName() string {
	return "wallhaven"
}

func (p *WallhavenProvider) Search(query string, opts SearchOptions) ([]models.Wallpaper, error) {
	baseURL := "https://wallhaven.cc/api/v1/search"
	u, err := url.Parse(baseURL)
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

	utils.Log.Debug("Wallhaven fetching: %s", u.String())

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wallhaven api returned status: %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			ID         string   `json:"id"`
			Path       string   `json:"path"`
			Resolution string   `json:"resolution"`
			Ratio      string   `json:"ratio"`
			Category   string   `json:"category"`
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
			Category:  item.Category,
			// Theme se deja vacío para que omitempty lo oculte hasta que se analice
		}
		wallpapers = append(wallpapers, wp)
	}

	return wallpapers, nil
}
