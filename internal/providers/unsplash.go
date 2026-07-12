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

var UnsplashBaseURL = "https://api.unsplash.com"

type UnsplashProvider struct {
	APIKey      string
	RateLimiter *utils.RateLimiter
	Log         *utils.Logger
}

func (p *UnsplashProvider) log() *utils.Logger {
	if p.Log != nil {
		return p.Log
	}
	return utils.Log
}

func (p *UnsplashProvider) GetName() string {
	return "unsplash"
}

func (p *UnsplashProvider) Search(ctx context.Context, query string, opts SearchOptions) ([]models.Wallpaper, error) {
	if query == "" {
		return p.fetchRandom(ctx, opts.Limit, opts.ExcludeIDs)
	}
	return p.searchPhotos(ctx, query, opts)
}

func (p *UnsplashProvider) fetchRandom(ctx context.Context, limit int, excludeIDs map[string]bool) ([]models.Wallpaper, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 30 {
		limit = 30
	}

	u, _ := url.Parse(UnsplashBaseURL + "/photos/random")
	q := u.Query()
	q.Set("count", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	p.log().Debug("Unsplash random fetching: %s", u.String())

	if p.RateLimiter != nil {
		p.RateLimiter.Wait()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Client-ID "+p.APIKey)

	resp, err := utils.DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unsplash api returned status: %d", resp.StatusCode)
	}

	var photos []unsplashPhoto
	if err := json.NewDecoder(resp.Body).Decode(&photos); err != nil {
		return nil, err
	}

	return p.mapPhotos(photos, limit, excludeIDs), nil
}

func (p *UnsplashProvider) searchPhotos(ctx context.Context, query string, opts SearchOptions) ([]models.Wallpaper, error) {
	perPage := opts.Limit
	if perPage <= 0 {
		perPage = 30
	}
	if perPage > 30 {
		perPage = 30
	}
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	u, _ := url.Parse(UnsplashBaseURL + "/search/photos")
	q := u.Query()
	q.Set("query", query)
	q.Set("per_page", strconv.Itoa(perPage))
	q.Set("page", strconv.Itoa(page))
	u.RawQuery = q.Encode()

	p.log().Debug("Unsplash search fetching: %s", u.String())

	if p.RateLimiter != nil {
		p.RateLimiter.Wait()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Client-ID "+p.APIKey)

	resp, err := utils.DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unsplash api returned status: %d", resp.StatusCode)
	}

	var result struct {
		Total   int             `json:"total"`
		Results []unsplashPhoto `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return p.mapPhotos(result.Results, opts.Limit, opts.ExcludeIDs), nil
}

type unsplashPhoto struct {
	ID          string `json:"id"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Color       string `json:"color"`
	Description string `json:"description"`
	AltDesc     string `json:"alt_description"`
	Urls        struct {
		Raw     string `json:"raw"`
		Full    string `json:"full"`
		Regular string `json:"regular"`
		Small   string `json:"small"`
		Thumb   string `json:"thumb"`
	} `json:"urls"`
	User struct {
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"user"`
	Links struct {
		HTML string `json:"html"`
	} `json:"links"`
}

func (p *UnsplashProvider) mapPhotos(photos []unsplashPhoto, limit int, excludeIDs map[string]bool) []models.Wallpaper {
	var wallpapers []models.Wallpaper
	for _, photo := range photos {
		if len(wallpapers) >= limit {
			break
		}
		id := "us_" + photo.ID
		if excludeIDs != nil && excludeIDs[id] {
			continue
		}

		title := photo.Description
		if title == "" {
			title = photo.AltDesc
		}

		dim := ""
		if photo.Width > 0 && photo.Height > 0 {
			dim = fmt.Sprintf("%dx%d", photo.Width, photo.Height)
		}

		// Use raw URL with q=100 for max quality, or regular as fallback
		imgURL := photo.Urls.Regular
		if photo.Urls.Raw != "" {
			imgURL = photo.Urls.Raw + "&q=100&w=2400"
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        id,
			URL:       imgURL,
			Thumbnail: photo.Urls.Small,
			Source:    "unsplash",
			Title:     title,
			Color:     photo.Color,
			Dimension: dim,
			Permalink: photo.Links.HTML,
		})
	}
	return wallpapers
}
