package providers

import (
	"encoding/json"
	"fmt"
	"gower/internal/utils"
	"gower/pkg/models"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// BingBaseURL es la URL base para la API de Bing Image of the Day.
var BingBaseURL = "https://www.bing.com/HPImageArchive.aspx"

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
	u, err := url.Parse(BingBaseURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("format", "js")                   // Solicitar formato JSON
	q.Set("idx", strconv.Itoa(opts.Page-1)) // idx=0 es hoy, idx=1 es ayer, etc.
	q.Set("n", strconv.Itoa(limit))         // Solicitar up to limit imágenes
	q.Set("mkt", p.Market)

	u.RawQuery = q.Encode()

	utils.Log.Debug("Bing fetching: %s", u.String())

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bing api returned status: %d", resp.StatusCode)
	}

	var result struct {
		Images []struct {
			Hsh       string `json:"hsh"` // Hash único, bueno para ID
			URL       string `json:"url"` // URL relativa de la imagen
			URLBase   string `json:"urlbase"`
			Copyright string `json:"copyright"`
			Drk       int    `json:"drk"` // 1 si es oscuro, 0 si es claro (no siempre presente o fiable)
		} `json:"images"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var wallpapers []models.Wallpaper
	for _, item := range result.Images {
		fullURL := "https://www.bing.com" + item.URL

		// Intentar extraer la resolución de la URL
		dimension := ""
		if strings.Contains(item.URL, "_") && strings.Contains(item.URL, ".jpg") {
			parts := strings.Split(item.URL, "_")
			if len(parts) > 1 {
				resPart := parts[len(parts)-1] // Última parte después del último '_'
				if strings.Contains(resPart, "x") {
					resPart = strings.Split(resPart, ".jpg")[0]                             // Eliminar .jpg
					if _, err := strconv.Atoi(strings.Split(resPart, "x")[0]); err == nil { // Verificar que es un número
						dimension = resPart
					}
				}
			}
		}

		wp := models.Wallpaper{
			ID:        "bing_" + item.Hsh, // Usamos el hash como ID único
			URL:       fullURL,
			Thumbnail: fullURL, // Usamos la misma URL para el thumbnail
			Source:    "bing",
			Dimension: dimension,
			// El tema se determinará en el análisis de color
		}
		wallpapers = append(wallpapers, wp)
	}

	return wallpapers, nil
}

// GetCategories no es aplicable para Bing Image of the Day.
func (p *BingProvider) GetCategories() []string {
	return []string{}
}
