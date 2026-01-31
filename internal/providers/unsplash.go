// internal/providers/unsplash.go
package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"gower/internal/utils"
	"gower/pkg/models"
)

const unsplashAPIBaseURL = "https://api.unsplash.com"

// UnsplashProvider implements the Provider interface for Unsplash.
type UnsplashProvider struct {
	APIKey string
}

// NewUnsplashProvider creates a new provider for Unsplash.
func NewUnsplashProvider(apiKey string) *UnsplashProvider {
	return &UnsplashProvider{APIKey: apiKey}
}

// GetName returns the name of the provider.
func (p *UnsplashProvider) GetName() string {
	return "unsplash"
}

// Search fetches wallpapers from Unsplash.
// For Unsplash, 'query' can be used to search for specific topics.
func (p *UnsplashProvider) Search(query string, opts SearchOptions) ([]models.Wallpaper, error) {
	if p.APIKey == "" {
		return nil, fmt.Errorf("Unsplash API key is not configured")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 30 { // Unsplash's max limit for random is 30
		limit = 30
	}

	reqURL := fmt.Sprintf("%s/photos/random?count=%d", unsplashAPIBaseURL, limit)
	if query != "" {
		reqURL = fmt.Sprintf("%s/search/photos?query=%s&per_page=%d", unsplashAPIBaseURL, query, limit)
	}

	utils.Log.Debug("Unsplash fetching: %s", reqURL)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Client-ID "+p.APIKey)
	req.Header.Set("Accept-Version", "v1")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unsplash API returned status: %d", resp.StatusCode)
	}

	// The response structure is different for random vs search
	if query != "" {
		return p.parseSearchResult(resp)
	}
	return p.parseRandomResult(resp)
}

// Unsplash API response for a single photo
type unsplashPhoto struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Color       string `json:"color"`
	Urls        struct {
		Full    string `json:"full"`
		Regular string `json:"regular"`
		Small   string `json:"small"`
		Thumb   string `json:"thumb"`
	} `json:"urls"`
	User struct {
		Name string `json:"name"`
	} `json:"user"`
}

// parseRandomResult parses the response from the /photos/random endpoint.
func (p *UnsplashProvider) parseRandomResult(resp *http.Response) ([]models.Wallpaper, error) {
	var results []unsplashPhoto
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return p.mapResultsToWallpapers(results), nil
}

// parseSearchResult parses the response from the /search/photos endpoint.
func (p *UnsplashProvider) parseSearchResult(resp *http.Response) ([]models.Wallpaper, error) {
	var searchResult struct {
		Results []unsplashPhoto `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, err
	}
	return p.mapResultsToWallpapers(searchResult.Results), nil
}

// mapResultsToWallpapers converts Unsplash API results to the common Wallpaper model.
func (p *UnsplashProvider) mapResultsToWallpapers(results []unsplashPhoto) []models.Wallpaper {
	var wallpapers []models.Wallpaper
	for _, photo := range results {
		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        "us_" + photo.ID,
			URL:       photo.Urls.Full,
			Thumbnail: photo.Urls.Thumb,
			Source:    p.GetName(),
			Title:     p.formatTitle(photo),
			Author:    photo.User.Name,
			Width:     strconv.Itoa(photo.Width),
			Height:    strconv.Itoa(photo.Height),
			Color:     photo.Color,
		})
	}
	return wallpapers
}

func (p *UnsplashProvider) formatTitle(photo unsplashPhoto) string {
	if photo.Description != "" {
		return photo.Description
	}
	return "Photo by " + photo.User.Name
}
