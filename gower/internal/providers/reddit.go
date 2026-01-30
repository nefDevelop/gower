package providers

import (
	"encoding/json"
	"fmt"
	"gower/pkg/models"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// RedditProvider implements the Provider interface for Reddit.
type RedditProvider struct {
	Config models.RedditConfig
}

// NewRedditProvider creates a new RedditProvider.
func NewRedditProvider(config models.RedditConfig) *RedditProvider {
	return &RedditProvider{Config: config}
}

func (p *RedditProvider) GetName() string {
	return "reddit"
}

func (p *RedditProvider) Search(query string, opts SearchOptions) ([]models.Wallpaper, error) {
	subreddit := p.Config.Subreddit
	if subreddit == "" {
		subreddit = "wallpapers"
	}
	sort := p.Config.Sort
	if sort == "" {
		sort = "mix"
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = p.Config.Limit
	}
	if limit <= 0 {
		limit = 25
	}

	// Estrategia MIX: Combinar Hot, New y Top
	if query == "" && sort == "mix" {
		return p.searchMixed(subreddit, limit, opts.ExcludeIDs)
	}

	// Pedimos más items a la API (hasta 100) para compensar el filtrado posterior
	apiLimit := 50
	if limit > apiLimit {
		apiLimit = limit
	}
	if apiLimit > 100 {
		apiLimit = 100
	}

	var url string
	timeParam := ""
	if sort == "top" || sort == "controversial" {
		timeParam = "&t=all" // Si es top, traemos los mejores de todos los tiempos para asegurar resultados
	}

	if query != "" {
		url = fmt.Sprintf("https://www.reddit.com/r/%s/search.json?q=%s&restrict_sr=1&sort=%s&limit=%d%s", subreddit, query, sort, apiLimit, timeParam)
	} else {
		url = fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d%s", subreddit, sort, apiLimit, timeParam)
	}

	return p.fetchFromReddit(url, limit, opts.ExcludeIDs)
}

func (p *RedditProvider) searchMixed(subreddit string, limit int, excludeIDs map[string]bool) ([]models.Wallpaper, error) {
	// Dividimos el esfuerzo, pero pedimos suficiente de cada uno
	apiLimit := 40

	urlHot := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%d", subreddit, apiLimit)
	urlNew := fmt.Sprintf("https://www.reddit.com/r/%s/new.json?limit=%d", subreddit, apiLimit)
	urlTop := fmt.Sprintf("https://www.reddit.com/r/%s/top.json?limit=%d&t=month", subreddit, apiLimit) // Top del mes para variar

	// Hacemos las peticiones (secuenciales por simplicidad, podrían ser paralelas)
	resHot, _ := p.fetchFromReddit(urlHot, limit, excludeIDs)
	resNew, _ := p.fetchFromReddit(urlNew, limit, excludeIDs)
	resTop, _ := p.fetchFromReddit(urlTop, limit, excludeIDs)

	// Combinar y desduplicar
	uniqueMap := make(map[string]models.Wallpaper)
	var combined []models.Wallpaper

	// Función helper para agregar
	add := func(list []models.Wallpaper) {
		for _, wp := range list {
			if _, exists := uniqueMap[wp.ID]; !exists {
				uniqueMap[wp.ID] = wp
				combined = append(combined, wp)
			}
		}
	}

	add(resHot)
	add(resNew)
	add(resTop)

	// Barajar resultados
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(combined), func(i, j int) { combined[i], combined[j] = combined[j], combined[i] })

	if len(combined) > limit {
		return combined[:limit], nil
	}
	return combined, nil
}

func (p *RedditProvider) fetchFromReddit(url string, limit int, excludeIDs map[string]bool) ([]models.Wallpaper, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Gower/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit api returned status: %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Children []struct {
				Data struct {
					ID        string `json:"id"`
					Title     string `json:"title"`
					URL       string `json:"url"`
					PostHint  string `json:"post_hint"`
					Permalink string `json:"permalink"`
					IsVideo   bool   `json:"is_video"`
					Preview   struct {
						Images []struct {
							Source struct {
								URL    string `json:"url"`
								Width  int    `json:"width"`
								Height int    `json:"height"`
							} `json:"source"`
							Resolutions []struct {
								URL   string `json:"url"`
								Width int    `json:"width"`
							} `json:"resolutions"`
						} `json:"images"`
					} `json:"preview"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var wallpapers []models.Wallpaper
	for _, child := range result.Data.Children {
		// Respetar el límite solicitado por el usuario
		if len(wallpapers) >= limit {
			break
		}

		data := child.Data
		if data.IsVideo {
			continue
		}

		id := "rd_" + data.ID
		if excludeIDs != nil && excludeIDs[id] {
			continue
		}

		// Detección mejorada de imágenes
		isImage := data.PostHint == "image"
		if !isImage {
			// Fallback: comprobar extensión si post_hint no es explícito
			lowerURL := strings.ToLower(data.URL)
			if strings.HasSuffix(lowerURL, ".jpg") || strings.HasSuffix(lowerURL, ".jpeg") || strings.HasSuffix(lowerURL, ".png") {
				isImage = true
			}
		}

		if !isImage {
			continue
		}

		// Obtener la mejor URL posible
		imageURL := data.URL
		if len(data.Preview.Images) > 0 {
			imageURL = data.Preview.Images[0].Source.URL
		}
		// Decodificar entidades HTML (&amp; -> &)
		imageURL = strings.ReplaceAll(imageURL, "&amp;", "&")

		// Determinar thumbnail
		thumb := ""
		if len(data.Preview.Images) > 0 {
			// Intentar buscar una resolución cercana a 320px de ancho
			for _, res := range data.Preview.Images[0].Resolutions {
				if res.Width >= 320 {
					thumb = strings.ReplaceAll(res.URL, "&amp;", "&")
					break
				}
			}
			if thumb == "" && len(data.Preview.Images[0].Resolutions) > 0 {
				thumb = strings.ReplaceAll(data.Preview.Images[0].Resolutions[0].URL, "&amp;", "&")
			}
		}

		// Determinar dimensión
		dimension := ""
		if len(data.Preview.Images) > 0 {
			src := data.Preview.Images[0].Source
			dimension = fmt.Sprintf("%dx%d", src.Width, src.Height)
		}

		wallpapers = append(wallpapers, models.Wallpaper{
			ID:        id,
			URL:       imageURL,
			Thumbnail: thumb,
			Source:    "reddit",
			Title:     data.Title,
			Permalink: "https://reddit.com" + data.Permalink,
			Dimension: dimension,
		})
	}

	return wallpapers, nil
}
