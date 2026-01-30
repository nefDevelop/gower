package providers

import (
	"encoding/json"
	"fmt"
	"gower/pkg/models"
	"net/http"
	"strings"
	"time"
)

// RedditProvider is a wallpaper provider for Reddit.
type RedditProvider struct {
	Config models.RedditConfig
	Client *http.Client
}

// NewRedditProvider creates a new instance of RedditProvider.
func NewRedditProvider(config models.RedditConfig) *RedditProvider {
	return &RedditProvider{
		Config: config,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetName returns the name of the provider.
func (rp *RedditProvider) GetName() string {
	return "Reddit"
}

// Search searches for wallpapers on Reddit.
func (rp *RedditProvider) Search(query string, options SearchOptions) ([]models.Wallpaper, error) {
	if !rp.Config.Enabled {
		return nil, fmt.Errorf("Reddit provider is not enabled")
	}

	subreddit := rp.Config.Subreddit
	if subreddit == "" {
		subreddit = "wallpapers" // Default to r/wallpapers if not specified
	}

	// Allow overriding subreddits with the query, if it's a "subreddit:foo+bar" format
	if strings.HasPrefix(query, "subreddit:") {
		subreddit = strings.TrimPrefix(query, "subreddit:")
		query = "" // Clear query if it was a subreddit override
	}

	// Construct URL
	// Example: https://www.reddit.com/r/wallpapers/search.json?q=nature&restrict_sr=on&sort=hot&limit=100
	url := fmt.Sprintf("https://www.reddit.com/r/%s/search.json?restrict_sr=on", subreddit)
	
	// Add query if not empty
	if query != "" {
		url += fmt.Sprintf("&q=%s", query)
	}

	// Add sort option
	sort := rp.Config.Sort
	if options.Sort != "" {
		sort = options.Sort
	}
	url += fmt.Sprintf("&sort=%s", sort)

	// Add limit option
	limit := rp.Config.Limit
	if options.Limit > 0 {
		limit = options.Limit
	}
	url += fmt.Sprintf("&limit=%d", limit)


	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gower-wallpaper-app/0.1") // Reddit requires a User-Agent

	resp, err := rp.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Reddit API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var redditResponse struct {
		Data struct {
			Children []struct {
				Data struct {
					ID        string `json:"id"`
					URL       string `json:"url_overridden_by_dest"` // Direct image URL
					Permalink string `json:"permalink"`
					Title     string `json:"title"`
					// Could add more fields like "thumbnail", "preview" for rich metadata
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&redditResponse); err != nil {
		return nil, fmt.Errorf("failed to decode Reddit API response: %w", err)
	}

	var wallpapers []models.Wallpaper
	for _, child := range redditResponse.Data.Children {
		data := child.Data
		// Filter out non-image URLs (e.g., self-posts, video links)
		if strings.HasSuffix(data.URL, ".jpg") || strings.HasSuffix(data.URL, ".png") || strings.HasSuffix(data.URL, ".jpeg") {
			wallpapers = append(wallpapers, models.Wallpaper{
				ID:        fmt.Sprintf("rd_%s", data.ID), // Prefix with rd_ to avoid ID collisions
				URL:       data.URL,
				Source:    "Reddit",
				Permalink: fmt.Sprintf("https://www.reddit.com%s", data.Permalink),
				Title:     data.Title,
			})
		}
	}

	return wallpapers, nil
}
