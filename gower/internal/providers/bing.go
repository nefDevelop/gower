package providers

import (
	"encoding/json"
	"fmt"
	"gower/internal/utils"
	"gower/pkg/models"
	"net/http"
)

type BingProvider struct {
	Market string
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

func (p *BingProvider) Search(query string, opts SearchOptions) ([]models.Wallpaper, error) {
	// Bing devuelve las imágenes del día (hasta 8 días atrás)
	limit := opts.Limit
	if limit <= 0 {
		limit = 8
	}
	if limit > 8 {
		limit = 8
	}

	url := fmt.Sprintf("https://www.bing.com/HPImageArchive.aspx?format=js&idx=0&n=%d&mkt=%s", limit, p.Market)

	utils.Log.Debug("Bing fetching: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bing api returned status: %d", resp.StatusCode)
	}

	var result struct {
		Images []struct {
			Startdate string `json:"startdate"`
			Url       string `json:"url"`
			Title     string `json:"title"`
			Copyright string `json:"copyright"`
		} `json:"images"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var wallpapers []models.Wallpaper
	for _, img := range result.Images {
		fullURL := "https://www.bing.com" + img.Url

		id := "bg_" + img.Startdate
		if opts.ExcludeIDs != nil && opts.ExcludeIDs[id] {
			continue
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        id,
			URL:       fullURL,
			Thumbnail: fullURL,
			Source:    "bing",
			Title:     img.Title,
			Category:  "daily",
		})
	}

	return wallpapers, nil
}
