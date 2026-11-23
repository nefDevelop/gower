package providers

import (
	"context"
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

// wallhavenSorting traduce los valores genéricos de Sort a los que acepta la
// API de Wallhaven. Un valor desconocido se omite en lugar de enviarse tal cual.
func wallhavenSorting(sort string) string {
	switch sort {
	case "date_added", "relevance", "random", "favorites", "toplist", "views", "hot":
		return sort
	case "newest", "latest":
		return "date_added"
	case "top", "popular":
		return "toplist"
	default:
		return ""
	}
}

func (p *WallhavenProvider) Search(ctx context.Context, query string, opts SearchOptions) ([]models.Wallpaper, error) {
	u, err := url.Parse(WallhavenBaseURL)
	if err != nil {
		return nil, err
	}

	limit := normalizeLimit(opts.Limit, 24, 48)

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
	if sorting := wallhavenSorting(opts.Sort); sorting != "" {
		q.Set("sorting", sorting)
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
	// Wallhaven solo admite 24, 32 o 48 resultados por página.
	switch {
	case limit <= 24:
		q.Set("per_page", "24")
	case limit <= 32:
		q.Set("per_page", "32")
	default:
		q.Set("per_page", "48")
	}

	u.RawQuery = q.Encode()

	p.log().Debug("Wallhaven fetching: %s", utils.RedactURL(u.String()))

	if err := waitRateLimiter(p.RateLimiter, ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := utils.DoRequest(ctx, utils.DefaultHTTPClient, req, 3) //nolint:bodyclose // cerrado con utils.CloseResponse
	if err != nil {
		return nil, err
	}
	defer utils.CloseResponse(resp)

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
		Meta struct {
			Error string `json:"error"`
		} `json:"meta"`
	}

	if err := utils.DecodeJSON(resp.Body, &result); err != nil {
		return nil, err
	}
	if result.Meta.Error != "" {
		return nil, fmt.Errorf("wallhaven api error: %s", result.Meta.Error)
	}

	wallpapers := make([]models.Wallpaper, 0, len(result.Data))
	for _, item := range result.Data {
		if len(wallpapers) >= limit {
			break
		}
		if opts.ExcludeIDs != nil && opts.ExcludeIDs["wh_"+item.ID] {
			continue
		}
		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        "wh_" + item.ID,
			URL:       item.Path,
			Thumbnail: item.Thumbs.Large,
			Source:    "wallhaven",
			Dimension: item.Resolution,
			Ratio:     item.Ratio,
		})
	}

	return wallpapers, nil
}
