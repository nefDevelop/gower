package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"gower/internal/utils"
	"gower/pkg/models"
)

// BingBaseURL es la URL base para la API de Bing Image of the Day.
var BingBaseURL = "https://www.bing.com/HPImageArchive.aspx"

var bingResolutionRE = regexp.MustCompile(`_(\d{3,5}x\d{3,5})\.(jpg|jpeg|png)$`)

type BingProvider struct {
	Market      string
	RateLimiter *utils.RateLimiter
	Log         *utils.Logger
}

func (p *BingProvider) log() *utils.Logger {
	if p.Log != nil {
		return p.Log
	}
	return utils.Log
}

func NewBingProvider(market string) *BingProvider {
	if market == "" {
		market = "en-US"
	}
	return &BingProvider{Market: market}
}

func (p *BingProvider) GetName() string {
	return "bing"
}

func (p *BingProvider) Search(ctx context.Context, query string, opts SearchOptions) ([]models.Wallpaper, error) {
	// Bing devuelve las imágenes del día (hasta 8 días atrás).
	limit := normalizeLimit(opts.Limit, 8, 8)

	idx := opts.Page - 1
	if idx < 0 {
		idx = 0
	}

	u, err := url.Parse(BingBaseURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("format", "js")
	q.Set("idx", strconv.Itoa(idx))
	q.Set("n", strconv.Itoa(limit))
	q.Set("mkt", p.Market)
	u.RawQuery = q.Encode()

	p.log().Debug("Bing fetching: %s", u.String())

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
		return nil, fmt.Errorf("bing api returned status: %d", resp.StatusCode)
	}

	var result struct {
		Images []struct {
			Hsh       string `json:"hsh"`
			URL       string `json:"url"`
			URLBase   string `json:"urlbase"`
			Copyright string `json:"copyright"`
		} `json:"images"`
	}

	if err := utils.DecodeJSON(resp.Body, &result); err != nil {
		return nil, err
	}

	wallpapers := make([]models.Wallpaper, 0, len(result.Images))
	for i, item := range result.Images {
		fullURL := item.URL
		if !strings.HasPrefix(fullURL, "http://") && !strings.HasPrefix(fullURL, "https://") {
			fullURL = "https://www.bing.com" + fullURL
		}
		if fullURL == "" {
			continue
		}

		dimension := ""
		if match := bingResolutionRE.FindStringSubmatch(fullURL); len(match) == 3 {
			dimension = match[1]
		}

		id := item.Hsh
		if id == "" {
			id = fmt.Sprintf("%s_%d", item.Copyright, i)
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        "bing_" + utils.SafeFilename(id),
			URL:       fullURL,
			Thumbnail: fullURL,
			Source:    "bing",
			Dimension: dimension,
		})
	}

	return wallpapers, nil
}

// GetCategories no es aplicable para Bing Image of the Day.
func (p *BingProvider) GetCategories() []string {
	return []string{}
}
