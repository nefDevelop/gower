package providers

import (
	"encoding/json"
	"fmt"
	"gower/internal/utils"
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
	subConfig := p.Config.Subreddit
	if subConfig == "" {
		subConfig = "wallpapers"
	}
	defaultSort := p.Config.Sort
	if defaultSort == "" {
		defaultSort = "mix"
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = p.Config.Limit
	}
	if limit <= 0 {
		limit = 25
	}

	// Agrupar subreddits por tipo de ordenamiento
	groups := make(map[string][]string)
	for _, item := range strings.Split(subConfig, "+") {
		parts := strings.Split(item, ":")
		sub := parts[0]
		sort := defaultSort
		if len(parts) > 1 && parts[1] != "" {
			sort = parts[1]
		}
		groups[sort] = append(groups[sort], sub)
	}

	var allWallpapers []models.Wallpaper

	for sort, subs := range groups {
		subreddit := strings.Join(subs, "+")

		// Calcular límite API para este grupo
		apiLimit := 50
		if limit > apiLimit {
			apiLimit = limit
		}
		if apiLimit > 100 {
			apiLimit = 100
		}

		var wps []models.Wallpaper
		var err error

		if query == "" && sort == "mix" {
			wps, err = p.searchMixed(subreddit, limit, opts)
		} else {
			var url string
			timeParam := ""
			if sort == "top" || sort == "controversial" {
				timeParam = "&t=all"
			}

			if query != "" {
				url = fmt.Sprintf("https://www.reddit.com/r/%s/search.json?q=%s&restrict_sr=1&sort=%s&limit=%d%s", subreddit, query, sort, apiLimit, timeParam)
			} else {
				url = fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d%s", subreddit, sort, apiLimit, timeParam)
			}
			wps, err = p.fetchFromReddit(url, limit, opts)
		}

		if err == nil {
			allWallpapers = append(allWallpapers, wps...)
		} else {
			utils.Log.Error("Error fetching from reddit group %s (sort: %s): %v", subreddit, sort, err)
		}
	}

	// Mezclar resultados si hay múltiples grupos
	if len(allWallpapers) > limit {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(allWallpapers), func(i, j int) {
			allWallpapers[i], allWallpapers[j] = allWallpapers[j], allWallpapers[i]
		})
		return allWallpapers[:limit], nil
	}

	return allWallpapers, nil
}

func (p *RedditProvider) searchMixed(subreddit string, limit int, opts SearchOptions) ([]models.Wallpaper, error) {
	// Dividimos el esfuerzo, pero pedimos suficiente de cada uno
	apiLimit := 40

	urlHot := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%d", subreddit, apiLimit)
	urlNew := fmt.Sprintf("https://www.reddit.com/r/%s/new.json?limit=%d", subreddit, apiLimit)
	urlTop := fmt.Sprintf("https://www.reddit.com/r/%s/top.json?limit=%d&t=month", subreddit, apiLimit) // Top del mes para variar

	// Hacemos las peticiones (secuenciales por simplicidad, podrían ser paralelas)
	resHot, _ := p.fetchFromReddit(urlHot, limit, opts)
	resNew, _ := p.fetchFromReddit(urlNew, limit, opts)
	resTop, _ := p.fetchFromReddit(urlTop, limit, opts)

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

func (p *RedditProvider) fetchFromReddit(url string, limit int, opts SearchOptions) ([]models.Wallpaper, error) {
	utils.Log.Debug("Reddit fetching: %s", url)
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
		if opts.ExcludeIDs != nil && opts.ExcludeIDs[id] {
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
		width := 0
		height := 0
		if len(data.Preview.Images) > 0 {
			src := data.Preview.Images[0].Source
			dimension = fmt.Sprintf("%dx%d", src.Width, src.Height)
			width = src.Width
			height = src.Height
		}

		// Filtrar por resolución si está disponible
		if width > 0 && height > 0 {
			if opts.MinWidth > 0 && width < opts.MinWidth {
				continue
			}
			if opts.MinHeight > 0 && height < opts.MinHeight {
				continue
			}
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
