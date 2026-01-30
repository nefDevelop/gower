package providers

import (
	"encoding/json"
	"fmt"
	"gower/pkg/models"
	"net/http"
	"net/url"
	"strings"
)

// NasaProvider implements the Provider interface for NASA APIs.
type NasaProvider struct {
	APIKey string
}

// NewNasaProvider creates a new NasaProvider.
func NewNasaProvider(apiKey string) *NasaProvider {
	return &NasaProvider{APIKey: apiKey}
}

func (p *NasaProvider) GetName() string {
	return "nasa"
}

func (p *NasaProvider) Search(query string, opts SearchOptions) ([]models.Wallpaper, error) {
	if query == "" {
		return p.fetchAPOD(opts.Limit, opts.ExcludeIDs)
	}
	return p.searchImageLibrary(query, opts.Limit, opts.ExcludeIDs)
}

func (p *NasaProvider) fetchAPOD(limit int, excludeIDs map[string]bool) ([]models.Wallpaper, error) {
	if limit <= 0 {
		limit = 10
	}
	// APOD 'count' parameter returns random images
	// Pedimos más items para filtrar videos, pero respetando el límite de 100 de la API
	count := limit * 2
	if count > 100 {
		count = 100
	}
	apiKey := p.APIKey
	if apiKey == "" {
		apiKey = "DEMO_KEY"
	}
	apiURL := fmt.Sprintf("https://api.nasa.gov/planetary/apod?api_key=%s&count=%d", apiKey, count)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nasa apod api returned status: %d", resp.StatusCode)
	}

	var results []struct {
		Title     string `json:"title"`
		Url       string `json:"url"`
		HdUrl     string `json:"hdurl"`
		MediaType string `json:"media_type"`
		Date      string `json:"date"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	var wallpapers []models.Wallpaper
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

		imgURL := item.HdUrl
		if imgURL == "" {
			imgURL = item.Url
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        id,
			URL:       imgURL,
			Thumbnail: item.Url, // APOD no da thumbnail separado, usamos la URL estándar
			Source:    "nasa",
			Title:     item.Title,
			Category:  "space",
		})
	}
	return wallpapers, nil
}

func (p *NasaProvider) searchImageLibrary(query string, limit int, excludeIDs map[string]bool) ([]models.Wallpaper, error) {
	baseURL := "https://images-api.nasa.gov/search"
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("q", query)
	q.Set("media_type", "image")
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var wallpapers []models.Wallpaper
	count := 0
	if limit <= 0 {
		limit = 25
	}

	for _, item := range result.Collection.Items {
		if count >= limit {
			break
		}
		if len(item.Data) == 0 || len(item.Links) == 0 {
			continue
		}

		data := item.Data[0]
		id := "ns_" + data.NasaID
		if excludeIDs != nil && excludeIDs[id] {
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

		// Heurística mejorada para alta resolución
		// Usamos ~large o ~medium que son más seguros que ~orig (que puede ser tif)
		imgURL := strings.Replace(thumb, "~thumb", "~large", 1)
		if imgURL == thumb {
			imgURL = strings.Replace(thumb, "~small", "~large", 1)
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        id,
			URL:       imgURL,
			Thumbnail: thumb,
			Source:    "nasa",
			Title:     data.Title,
			Category:  "space",
		})
		count++
	}
	return wallpapers, nil
}
