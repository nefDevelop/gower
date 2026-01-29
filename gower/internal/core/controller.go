package core

import (
	"fmt"
	"gower/internal/providers"
	"gower/internal/utils"
	"gower/pkg/models"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Controller is the main controller of the application.
type Controller struct {
	ProviderManager *ProviderManager
	feedManager     *utils.SecureJSONManager // Manager for feed.json
}

// NewController creates a new Controller.
func NewController(config *models.Config) *Controller {
	providerManager := NewProviderManager()

	// Register native providers
	providerManager.RegisterProvider(&providers.WallhavenProvider{})
	// Register other native providers here...

	// Register generic providers
	for _, providerConfig := range config.GenericProviders {
		if providerConfig.Enabled {
			provider := &providers.GenericProvider{Config: providerConfig}
			providerManager.RegisterProvider(provider)
		}
	}

	return &Controller{
		ProviderManager: providerManager,
		feedManager:     utils.NewSecureJSONManager(),
	}
}

func (c *Controller) getFeedPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gower", "data", "feed.json"), nil
}

func (c *Controller) loadFeed() ([]models.Wallpaper, error) {
	path, err := c.getFeedPath()
	if err != nil {
		return nil, err
	}

	var feed []models.Wallpaper
	// If file doesn't exist, return empty list without error
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []models.Wallpaper{}, nil
	}

	if err := c.feedManager.ReadJSON(path, &feed); err != nil {
		return nil, err
	}
	return feed, nil
}

func (c *Controller) saveFeed(feed []models.Wallpaper) error {
	path, err := c.getFeedPath()
	if err != nil {
		return err
	}
	return c.feedManager.WriteJSON(path, feed)
}

// GetFeed retrieves wallpapers from the feed with pagination and optional search/theme filters.
func (c *Controller) GetFeed(page, limit int, search, theme string) ([]models.Wallpaper, error) {
	feed, err := c.loadFeed()
	if err != nil {
		return nil, err
	}

	var filteredFeed []models.Wallpaper
	for _, wp := range feed {
		matchesSearch := true
		if search != "" && !strings.Contains(strings.ToLower(wp.ID), strings.ToLower(search)) &&
			!strings.Contains(strings.ToLower(wp.Source), strings.ToLower(search)) {
			matchesSearch = false
		}

		matchesTheme := true
		if theme != "" && strings.ToLower(wp.Theme) != strings.ToLower(theme) {
			matchesTheme = false
		}

		if matchesSearch && matchesTheme {
			filteredFeed = append(filteredFeed, wp)
		}
	}

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= len(filteredFeed) {
		return []models.Wallpaper{}, nil
	}
	if end > len(filteredFeed) {
		end = len(filteredFeed)
	}

	return filteredFeed[start:end], nil
}

// SearchFeed searches the feed for wallpapers matching a query.
func (c *Controller) SearchFeed(query string, page, limit int, theme string) ([]models.Wallpaper, error) {
	// Reuse GetFeed with the search parameter
	return c.GetFeed(page, limit, query, theme)
}

// PurgeFeed clears all entries from the feed.
func (c *Controller) PurgeFeed() error {
	return c.saveFeed([]models.Wallpaper{})
}

// GetRandomFromFeed retrieves a random wallpaper from the feed, optionally filtered by theme.
func (c *Controller) GetRandomFromFeed(theme string) (models.Wallpaper, error) {
	feed, err := c.loadFeed()
	if err != nil {
		return models.Wallpaper{}, err
	}

	var filteredFeed []models.Wallpaper
	for _, wp := range feed {
		if theme == "" || strings.ToLower(wp.Theme) == strings.ToLower(theme) {
			filteredFeed = append(filteredFeed, wp)
		}
	}

	if len(filteredFeed) == 0 {
		return models.Wallpaper{}, fmt.Errorf("no wallpapers found in feed (with given theme)")
	}

	// TODO: Use a proper random number generator
	randomIndex := time.Now().Nanosecond() % len(filteredFeed)
	return filteredFeed[randomIndex], nil
}

// GetFeedStats calculates and returns statistics about the feed.
func (c *Controller) GetFeedStats() (models.FeedStats, error) {
	feed, err := c.loadFeed()
	if err != nil {
		return models.FeedStats{}, err
	}

	stats := models.FeedStats{
		Total: len(feed),
	}

	for _, wp := range feed {
		if strings.ToLower(wp.Theme) == "dark" {
			stats.DarkCount++
		} else if strings.ToLower(wp.Theme) == "light" {
			stats.LightCount++
		}
		// For FavoritesCount and LastAdded, we'd need more info in Wallpaper model
		// or a separate favorites manager. For now, leave as 0 or default.
	}

	return stats, nil
}

// AddWallpaperToFeed adds a wallpaper to the feed.
func (c *Controller) AddWallpaperToFeed(wallpaper models.Wallpaper) error {
	feed, err := c.loadFeed()
	if err != nil {
		return err
	}

	// Check if already exists to avoid duplicates
	for _, existingWp := range feed {
		if existingWp.ID == wallpaper.ID {
			return nil // Already in feed, do nothing
		}
	}

	feed = append(feed, wallpaper)
	return c.saveFeed(feed)
}

// GetFeedWallpapers returns all wallpapers in the feed.
func (c *Controller) GetFeedWallpapers() ([]models.Wallpaper, error) {
	return c.loadFeed()
}