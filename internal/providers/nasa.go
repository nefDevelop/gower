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
		return p.fetchAPOD(opts.Limit)
	}
	return p.searchImageLibrary(query, opts.Limit)
}

func (p *NasaProvider) fetchAPOD(limit int) ([]models.Wallpaper, error) {
	if limit <= 0 {
		limit = 10
	}
	// APOD 'count' parameter returns random images
	apiURL := fmt.Sprintf("https://api.nasa.gov/planetary/apod?api_key=%s&count=%d", p.APIKey, limit)

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
		if item.MediaType != "image" {
			continue
		}

		imgURL := item.HdUrl
		if imgURL == "" {
			imgURL = item.Url
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        "ns_" + item.Date,
			URL:       imgURL,
			Thumbnail: item.Url, // APOD no da thumbnail separado, usamos la URL estándar
			Source:    "nasa",
			Title:     item.Title,
			Category:  "space",
		})
	}
	return wallpapers, nil
}

func (p *NasaProvider) searchImageLibrary(query string, limit int) ([]models.Wallpaper, error) {
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
		// La API suele devolver ...~thumb.jpg. Intentamos adivinar la original.
		imgURL := strings.Replace(thumb, "~thumb", "~orig", 1)
		if imgURL == thumb {
			// Si no hubo reemplazo, intentamos con ~medium como fallback seguro si existe ese patrón
			imgURL = strings.Replace(thumb, "~small", "~medium", 1)
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        "ns_" + data.NasaID,
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
