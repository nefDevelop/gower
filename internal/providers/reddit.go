package providers

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"gower/internal/utils"
	"gower/pkg/models"
)

// RedditProvider implements the Provider interface for Reddit.
type RedditProvider struct {
	Config      models.RedditConfig
	RateLimiter *utils.RateLimiter
	Log         *utils.Logger
}

func (p *RedditProvider) log() *utils.Logger {
	if p.Log != nil {
		return p.Log
	}
	return utils.Log
}

// NewRedditProvider creates a new RedditProvider.
func NewRedditProvider(config models.RedditConfig) *RedditProvider {
	return &RedditProvider{Config: config}
}

func (p *RedditProvider) GetName() string {
	return "reddit"
}

func (p *RedditProvider) Search(ctx context.Context, query string, opts SearchOptions) ([]models.Wallpaper, error) {
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

	groups := make(map[string][]string)
	for _, item := range strings.Split(subConfig, "+") {
		parts := strings.Split(item, ":")
		sub := parts[0]
		sortMode := defaultSort
		if len(parts) > 1 && parts[1] != "" {
			sortMode = parts[1]
		}
		groups[sortMode] = append(groups[sortMode], sub)
	}

	// El orden del map no es determinista: ordenar los grupos evita que el
	// proveedor por defecto cambie entre ejecuciones.
	sortModes := make([]string, 0, len(groups))
	for mode := range groups {
		sortModes = append(sortModes, mode)
	}
	sort.Strings(sortModes)

	// Reddit no permite offset directo: se pide suficiente historial y se
	// recorta la página solicitada.
	page := opts.Page
	if page <= 0 {
		page = 1
	}
	apiLimit := limit * page
	if apiLimit > 100 {
		apiLimit = 100
	}
	if apiLimit < limit {
		apiLimit = limit
	}

	var allWallpapers []models.Wallpaper
	var firstErr error
	successes := 0

	for _, sortMode := range sortModes {
		subreddit := strings.Join(groups[sortMode], "+")

		var wps []models.Wallpaper
		var err error

		if query == "" && sortMode == "mix" {
			wps, err = p.searchMixed(ctx, subreddit, apiLimit, opts)
		} else {
			u, errURL := redditListingURL(subreddit, sortMode, query)
			if errURL != nil {
				err = errURL
			} else {
				q := u.Query()
				q.Set("limit", fmt.Sprintf("%d", apiLimit))
				if sortMode == "top" || sortMode == "controversial" {
					q.Set("t", "all")
				}
				u.RawQuery = q.Encode()
				wps, err = p.fetchFromReddit(ctx, u.String(), apiLimit, opts)
			}
		}

		if err != nil {
			p.log().Error("Error fetching from reddit group %s (sort: %s): %v", subreddit, sortMode, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		successes++
		allWallpapers = append(allWallpapers, wps...)
	}

	if successes == 0 && firstErr != nil {
		return nil, firstErr
	}

	// Deduplicar manteniendo el orden de llegada.
	seen := make(map[string]bool, len(allWallpapers))
	unique := allWallpapers[:0]
	for _, wp := range allWallpapers {
		if seen[wp.ID] {
			continue
		}
		seen[wp.ID] = true
		unique = append(unique, wp)
	}
	allWallpapers = unique

	if len(sortModes) > 1 && len(allWallpapers) > 1 {
		rand.Shuffle(len(allWallpapers), func(i, j int) {
			allWallpapers[i], allWallpapers[j] = allWallpapers[j], allWallpapers[i]
		})
	}

	return paginate(allWallpapers, page, limit), nil
}

// redditListingURL construye la URL de un listing escapando la query.
func redditListingURL(subreddit, sortMode, query string) (*url.URL, error) {
	path := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json", subreddit, sortMode)
	if query != "" {
		path = fmt.Sprintf("https://www.reddit.com/r/%s/search.json", subreddit)
	}
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if query != "" {
		q.Set("q", query)
		q.Set("restrict_sr", "1")
		q.Set("sort", sortMode)
	}
	u.RawQuery = q.Encode()
	return u, nil
}

func paginate(list []models.Wallpaper, page, limit int) []models.Wallpaper {
	if limit <= 0 {
		return list
	}
	start := (page - 1) * limit
	if start >= len(list) {
		return []models.Wallpaper{}
	}
	end := start + limit
	if end > len(list) {
		end = len(list)
	}
	return list[start:end]
}

func (p *RedditProvider) searchMixed(ctx context.Context, subreddit string, limit int, opts SearchOptions) ([]models.Wallpaper, error) {
	apiLimit := normalizeLimit(limit, 40, 100)

	urls := []string{
		fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%d", subreddit, apiLimit),
		fmt.Sprintf("https://www.reddit.com/r/%s/new.json?limit=%d", subreddit, apiLimit),
		fmt.Sprintf("https://www.reddit.com/r/%s/top.json?limit=%d&t=month", subreddit, apiLimit),
	}

	var combined []models.Wallpaper
	seen := make(map[string]bool)
	var firstErr error
	successes := 0

	for _, u := range urls {
		list, err := p.fetchFromReddit(ctx, u, apiLimit, opts)
		if err != nil {
			p.log().Error("Reddit mixed listing failed (%s): %v", u, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		successes++
		for _, wp := range list {
			if seen[wp.ID] {
				continue
			}
			seen[wp.ID] = true
			combined = append(combined, wp)
		}
	}

	if successes == 0 && firstErr != nil {
		return nil, firstErr
	}

	rand.Shuffle(len(combined), func(i, j int) { combined[i], combined[j] = combined[j], combined[i] })
	return combined, nil
}

func (p *RedditProvider) fetchFromReddit(ctx context.Context, rawURL string, limit int, opts SearchOptions) ([]models.Wallpaper, error) {
	p.log().Debug("Reddit fetching: %s", rawURL)
	if err := waitRateLimiter(p.RateLimiter, ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gower/1.0 (+https://github.com/nefDevelop/gower)")

	resp, err := utils.DoRequest(ctx, utils.DefaultHTTPClient, req, 3) //nolint:bodyclose // cerrado con utils.CloseResponse
	if err != nil {
		return nil, err
	}
	defer utils.CloseResponse(resp)

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

	if err := utils.DecodeJSON(resp.Body, &result); err != nil {
		return nil, err
	}

	wallpapers := make([]models.Wallpaper, 0, len(result.Data.Children))
	for _, child := range result.Data.Children {
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

		isImage := data.PostHint == "image"
		if !isImage {
			lowerURL := strings.ToLower(data.URL)
			for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif"} {
				if strings.HasSuffix(lowerURL, ext) {
					isImage = true
					break
				}
			}
		}
		if !isImage {
			continue
		}

		imageURL := data.URL
		if len(data.Preview.Images) > 0 {
			imageURL = data.Preview.Images[0].Source.URL
		}
		imageURL = strings.ReplaceAll(imageURL, "&amp;", "&")

		thumb := ""
		if len(data.Preview.Images) > 0 {
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

		dimension := ""
		width := 0
		height := 0
		if len(data.Preview.Images) > 0 {
			src := data.Preview.Images[0].Source
			dimension = fmt.Sprintf("%dx%d", src.Width, src.Height)
			width = src.Width
			height = src.Height
		}

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
