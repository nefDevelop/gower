package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"gower/internal/utils"
	"gower/pkg/models"
)

// NasaProvider implements the Provider interface for NASA APIs.
type NasaProvider struct {
	APIKey      string
	RateLimiter *utils.RateLimiter
	Log         *utils.Logger
}

func (p *NasaProvider) log() *utils.Logger {
	if p.Log != nil {
		return p.Log
	}
	return utils.Log
}

// NewNasaProvider creates a new NasaProvider.
func NewNasaProvider(apiKey string) *NasaProvider {
	return &NasaProvider{APIKey: apiKey}
}

func (p *NasaProvider) GetName() string {
	return "nasa"
}

func (p *NasaProvider) Search(ctx context.Context, query string, opts SearchOptions) ([]models.Wallpaper, error) {
	if query == "" {
		return p.fetchAPOD(ctx, opts.Limit, opts.ExcludeIDs)
	}
	return p.searchImageLibrary(ctx, query, opts)
}

func (p *NasaProvider) fetchAPOD(ctx context.Context, limit int, excludeIDs map[string]bool) ([]models.Wallpaper, error) {
	limit = normalizeLimit(limit, 10, 100)

	// APOD 'count' devuelve imágenes aleatorias: se piden más para poder
	// descartar vídeos sin quedarse corto.
	count := limit * 2
	if count > 100 {
		count = 100
	}

	apiKey := p.APIKey
	if apiKey == "" {
		apiKey = "DEMO_KEY"
	}

	u, err := url.Parse("https://api.nasa.gov/planetary/apod")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("api_key", apiKey)
	q.Set("count", strconv.Itoa(count))
	u.RawQuery = q.Encode()

	p.log().Debug("NASA APOD fetching: %s", utils.RedactURL(u.String()))

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
		return nil, fmt.Errorf("nasa apod api returned status: %d", resp.StatusCode)
	}

	var results []struct {
		Title     string `json:"title"`
		URL       string `json:"url"`
		HdURL     string `json:"hdurl"`
		MediaType string `json:"media_type"`
		Date      string `json:"date"`
	}

	if err := utils.DecodeJSON(resp.Body, &results); err != nil {
		return nil, err
	}

	wallpapers := make([]models.Wallpaper, 0, limit)
	for _, item := range results {
		if len(wallpapers) >= limit {
			break
		}
		if item.MediaType != "image" {
			continue
		}

		id := "ns_" + item.Date
		if excludeIDs != nil && excludeIDs[id] {
			continue
		}

		imgURL := item.HdURL
		if imgURL == "" {
			imgURL = item.URL
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        id,
			URL:       imgURL,
			Thumbnail: item.URL,
			Source:    "nasa",
			Title:     item.Title,
		})
	}
	return wallpapers, nil
}

func (p *NasaProvider) searchImageLibrary(ctx context.Context, query string, opts SearchOptions) ([]models.Wallpaper, error) {
	limit := normalizeLimit(opts.Limit, 25, 100)
	page := opts.Page
	if page <= 0 {
		page = 1
	}

	u, err := url.Parse("https://images-api.nasa.gov/search")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("q", query)
	q.Set("media_type", "image")
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	p.log().Debug("NASA Image Library fetching: %s", u.String())

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
		return nil, fmt.Errorf("nasa image api returned status: %d", resp.StatusCode)
	}

	var result struct {
		Collection struct {
			Items []struct {
				Data []struct {
					NasaID string `json:"nasa_id"`
					Title  string `json:"title"`
				} `json:"data"`
				Links []struct {
					Href string `json:"href"`
					Rel  string `json:"rel"`
				} `json:"links"`
			} `json:"items"`
		} `json:"collection"`
	}

	if err := utils.DecodeJSON(resp.Body, &result); err != nil {
		return nil, err
	}

	wallpapers := make([]models.Wallpaper, 0, limit)
	for _, item := range result.Collection.Items {
		if len(wallpapers) >= limit {
			break
		}
		if len(item.Data) == 0 || len(item.Links) == 0 {
			continue
		}

		data := item.Data[0]
		id := "ns_" + data.NasaID
		if opts.ExcludeIDs != nil && opts.ExcludeIDs[id] {
			continue
		}

		thumb := ""
		for _, link := range item.Links {
			if link.Rel == "preview" {
				thumb = link.Href
				break
			}
		}
		if thumb == "" && len(item.Links) > 0 {
			thumb = item.Links[0].Href
		}

		// ~thumb y ~small son versiones reducidas; ~orig puede ser TIFF.
		imgURL := strings.Replace(thumb, "~thumb", "~large", 1)
		if imgURL == thumb {
			imgURL = strings.Replace(thumb, "~small", "~large", 1)
		}
		if imgURL == thumb && !strings.HasPrefix(thumb, "http") {
			continue
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        id,
			URL:       imgURL,
			Thumbnail: thumb,
			Source:    "nasa",
			Title:     data.Title,
		})
	}
	return wallpapers, nil
}
