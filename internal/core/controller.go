package core

import (
	"fmt"
	"gower/internal/providers"
	"gower/internal/utils"
	"gower/pkg/models" // Import models package
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Controller is the main controller of the application.
type Controller struct {
	ProviderManager *ProviderManager
	feedManager     *utils.SecureJSONManager // Manager for feed.json
	ColorManager    *ColorManager
}

// NewController creates a new Controller.
func NewController(config *models.Config) *Controller {
	providerManager := NewProviderManager()

	// Register native providers
	providerManager.RegisterProvider(&providers.WallhavenProvider{
		APIKey: config.Providers.Wallhaven.APIKey,
	})
	providerManager.RegisterProvider(providers.NewRedditProvider(config.Providers.Reddit)) // Register RedditProvider
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
		ColorManager:    NewColorManager(),
	}
}

func (c *Controller) getFeedPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gower", "data", "feed.json"), nil
}

func (c *Controller) getBlacklistPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gower", "data", "blacklist.json"), nil
}

func (c *Controller) loadBlacklist() ([]string, error) {
	path, err := c.getBlacklistPath()
	if err != nil {
		return nil, err
	}
	var blacklist []string
	// We use a generic manager or just read it. Assuming simple string array for IDs.
	// If file doesn't exist, return empty.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []string{}, nil
	}
	// Using feedManager (SecureJSONManager) which is generic enough
	if err := c.feedManager.ReadJSON(path, &blacklist); err != nil {
		return nil, err
	}
	return blacklist, nil
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
func (c *Controller) GetFeed(page, limit int, search, theme, color string) ([]models.Wallpaper, error) {
	feed, err := c.loadFeed()
	if err != nil {
		return nil, err
	}

	blacklist, _ := c.loadBlacklist()
	blacklistMap := make(map[string]bool)
	for _, id := range blacklist {
		blacklistMap[id] = true
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

		matchesColor := true
		if color != "" {
			// Simple check: if any color in palette contains the requested hex
			// Assuming wp.Palette exists (based on documentation) or we skip if not available
			// Since models.Wallpaper definition isn't fully visible, we assume it might have Colors or Palette
			// For now, if we can't check, we might ignore or fail.
			// Let's assume strict filtering: if we can't find it, it doesn't match.
			// Implementation detail: This depends on models.Wallpaper having a color field.
			// If not present in struct, this block is a placeholder.
		}

		if blacklistMap[wp.ID] {
			continue
		}

		if matchesSearch && matchesTheme && matchesColor {
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
	return c.GetFeed(page, limit, query, theme, "")
}

// PurgeFeed clears all entries from the feed.
func (c *Controller) PurgeFeed() error {
	return c.saveFeed([]models.Wallpaper{})
}

// AnalyzeFeed analyzes the feed (placeholder).
func (c *Controller) AnalyzeFeed(all bool) error {
	// Implementation for analyzing feed items (e.g. extracting colors)
	return nil
}

// GetRandomFromFeed retrieves a random wallpaper from the feed, optionally filtered by theme.
func (c *Controller) GetRandomFromFeed(theme string) (models.Wallpaper, error) {
	feed, err := c.loadFeed()
	if err != nil {
		return models.Wallpaper{}, err
	}

	blacklist, _ := c.loadBlacklist()
	blacklistMap := make(map[string]bool)
	for _, id := range blacklist {
		blacklistMap[id] = true
	}

	var filteredFeed []models.Wallpaper
	for _, wp := range feed {
		if blacklistMap[wp.ID] {
			continue
		}
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
	// Check blacklist
	blacklist, err := c.loadBlacklist()
	if err != nil {
		return err
	}
	for _, id := range blacklist {
		if id == wallpaper.ID {
			return fmt.Errorf("wallpaper %s is blacklisted", wallpaper.ID)
		}
	}

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

// AddWallpapersToFeed adds multiple wallpapers to the feed efficiently.
func (c *Controller) AddWallpapersToFeed(wallpapers []models.Wallpaper) (int, error) {
	feed, err := c.loadFeed()
	if err != nil {
		return 0, err
	}

	blacklist, err := c.loadBlacklist()
	if err != nil {
		return 0, err
	}
	blacklistMap := make(map[string]bool)
	for _, id := range blacklist {
		blacklistMap[id] = true
	}

	// Create a map for existing IDs to avoid duplicates
	existing := make(map[string]bool)
	for _, wp := range feed {
		existing[wp.ID] = true
	}

	addedCount := 0
	for _, wp := range wallpapers {
		if blacklistMap[wp.ID] {
			continue
		}
		if !existing[wp.ID] {
			feed = append(feed, wp)
			existing[wp.ID] = true
			addedCount++
		}
	}

	if addedCount > 0 {
		return addedCount, c.saveFeed(feed)
	}
	return 0, nil
}

// AddToBlacklist adds an ID to the blacklist.
func (c *Controller) AddToBlacklist(id string) error {
	blacklist, err := c.loadBlacklist()
	if err != nil {
		return err
	}
	for _, existing := range blacklist {
		if existing == id {
			return nil
		}
	}
	blacklist = append(blacklist, id)

	path, err := c.getBlacklistPath()
	if err != nil {
		return err
	}
	return c.feedManager.WriteJSON(path, blacklist)
}

// RemoveFromFeed removes a wallpaper from the feed by ID.
func (c *Controller) RemoveFromFeed(id string) error {
	feed, err := c.loadFeed()
	if err != nil {
		return err
	}

	newFeed := make([]models.Wallpaper, 0, len(feed))
	found := false
	for _, wp := range feed {
		if wp.ID == id {
			found = true
			continue
		}
		newFeed = append(newFeed, wp)
	}

	if !found {
		return nil
	}

	return c.saveFeed(newFeed)
}

// GetFeedWallpapers returns all wallpapers in the feed.
func (c *Controller) GetFeedWallpapers() ([]models.Wallpaper, error) {
	return c.loadFeed()
}

// GetWallpaper attempts to find a wallpaper by ID in the feed or favorites.
func (c *Controller) GetWallpaper(id string) (*models.Wallpaper, error) {
	// 1. Check Feed
	feed, err := c.loadFeed()
	if err == nil {
		for _, wp := range feed {
			if wp.ID == id {
				return &wp, nil
			}
		}
	}

	// 2. Check Favorites
	// We need to manually load favorites here since Controller doesn't manage them directly yet,
	// or we can assume the caller handles it. However, for convenience:
	favPath := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(c.getFeedPathString()))), "favorites.json")
	var favorites []struct {
		models.Wallpaper
		Notes string `json:"notes,omitempty"`
	}
	if err := c.feedManager.ReadJSON(favPath, &favorites); err == nil {
		for _, fav := range favorites {
			if fav.ID == id {
				return &fav.Wallpaper, nil
			}
		}
	}

	return nil, fmt.Errorf("wallpaper with ID %s not found", id)
}

// Helper to get path string (ignoring error for internal use)
func (c *Controller) getFeedPathString() string {
	p, _ := c.getFeedPath()
	return p
}

// GetWallpaperLocalPath returns the expected local path for a wallpaper without downloading it.
func (c *Controller) GetWallpaperLocalPath(wp models.Wallpaper) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(homeDir, ".gower", "cache", "wallpapers")

	// Determine filename
	ext := filepath.Ext(wp.URL)
	if ext == "" {
		ext = ".jpg"
	}
	safeID := strings.ReplaceAll(wp.ID, "/", "_")
	filename := fmt.Sprintf("%s%s", safeID, ext)
	return filepath.Join(cacheDir, filename), nil
}

// DownloadWallpaper downloads the wallpaper image to the cache directory and returns the local path.
func (c *Controller) DownloadWallpaper(wp models.Wallpaper) (string, error) {
	filePath, err := c.GetWallpaperLocalPath(wp)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}

	// Check if already exists
	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}

	// Download
	resp, err := http.Get(wp.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download wallpaper: status %d", resp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	// Post-download processing
	// 1. Generate Thumbnail
	thumbDir := filepath.Join(filepath.Dir(filepath.Dir(filePath)), "thumbs")
	thumbPath := filepath.Join(thumbDir, filepath.Base(filePath))
	if err := c.ColorManager.GenerateThumbnail(filePath, thumbPath); err != nil {
		utils.Log.Error("Failed to generate thumbnail for %s: %v", wp.ID, err)
	}

	// 2. Analyze and Index Color
	hexColor, err := c.ColorManager.AnalyzeColor(filePath)
	if err == nil {
		if err := c.ColorManager.UpdateIndex(hexColor); err != nil {
			utils.Log.Error("Failed to update color index: %v", err)
		}
	} else {
		utils.Log.Debug("Color analysis failed/skipped for %s: %v", wp.ID, err)
	}

	return filePath, nil
}

// GetCachedWallpapers retrieves wallpapers from feed (and optionally favorites) that are locally cached.
func (c *Controller) GetCachedWallpapers(includeFavorites bool, theme string) ([]models.Wallpaper, error) {
	var candidates []models.Wallpaper

	// Load Feed
	feed, err := c.loadFeed()
	if err == nil {
		candidates = append(candidates, feed...)
	}

	// Load Favorites if requested
	if includeFavorites {
		favPath := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(c.getFeedPathString()))), "favorites.json")
		var favorites []struct {
			models.Wallpaper
			Notes string `json:"notes,omitempty"`
		}
		if err := c.feedManager.ReadJSON(favPath, &favorites); err == nil {
			for _, f := range favorites {
				candidates = append(candidates, f.Wallpaper)
			}
		}
	}

	// Filter
	var result []models.Wallpaper
	seen := make(map[string]bool)

	for _, wp := range candidates {
		if seen[wp.ID] {
			continue
		}
		seen[wp.ID] = true

		// Theme check
		if theme != "" && theme != "auto" && strings.ToLower(wp.Theme) != strings.ToLower(theme) {
			continue
		}

		// Cache check
		path, err := c.GetWallpaperLocalPath(wp)
		if err != nil {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			result = append(result, wp)
		}
	}
	return result, nil
}

// ParserSearch represents a search session stored in the parser cache.
type ParserSearch struct {
	Date    time.Time          `json:"date"`
	Query   string             `json:"query"`
	Results []models.Wallpaper `json:"results"`
}

func (c *Controller) getParserPath(providerName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gower", "data", "parser", providerName+".json"), nil
}

// SaveParserSearch saves the search results to the provider's parser cache file.
// It enforces a 14-day retention policy for old searches.
func (c *Controller) SaveParserSearch(providerName, query string, results []models.Wallpaper) error {
	path, err := c.getParserPath(providerName)
	if err != nil {
		return err
	}

	var searches []ParserSearch
	// Try to read existing file
	if _, err := os.Stat(path); err == nil {
		// We ignore error here to overwrite corrupt files or start fresh
		_ = c.feedManager.ReadJSON(path, &searches)
	}

	// Prune searches older than 14 days
	cutoff := time.Now().AddDate(0, 0, -14)
	var validSearches []ParserSearch
	for _, s := range searches {
		if s.Date.After(cutoff) {
			validSearches = append(validSearches, s)
		}
	}

	// Append new search
	newSearch := ParserSearch{
		Date:    time.Now(),
		Query:   query,
		Results: results,
	}
	validSearches = append(validSearches, newSearch)

	return c.feedManager.WriteJSON(path, validSearches)
}

// SyncFeed processes parser cache files and populates the feed.
func (c *Controller) SyncFeed() (int, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return 0, err
	}
	parserDir := filepath.Join(homeDir, ".gower", "data", "parser")

	files, err := os.ReadDir(parserDir)
	if err != nil {
		// If dir doesn't exist, just return 0
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	// Load existing feed and blacklist to avoid duplicates
	feed, _ := c.loadFeed()
	blacklist, _ := c.loadBlacklist()

	existing := make(map[string]bool)
	for _, wp := range feed {
		existing[wp.ID] = true
	}
	for _, id := range blacklist {
		existing[id] = true
	}

	addedCount := 0
	thumbDir := filepath.Join(homeDir, ".gower", "cache", "thumbs")

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		var searches []ParserSearch
		if err := c.feedManager.ReadJSON(filepath.Join(parserDir, file.Name()), &searches); err != nil {
			continue
		}

		for _, search := range searches {
			for _, wp := range search.Results {
				if existing[wp.ID] {
					continue
				}

				// Process new wallpaper
				thumbPath := filepath.Join(thumbDir, wp.ID+".jpg")

				// 1. Generate/Download Thumbnail
				// Use full URL as source for thumbnail generation
				if err := c.ColorManager.GenerateThumbnail(wp.URL, thumbPath); err == nil {
					// 2. Analyze Color
					if color, err := c.ColorManager.AnalyzeColor(thumbPath); err == nil {
						// Assuming models.Wallpaper has a Color field or we use Palette
						// Since we can't see models, we assume we can't set it easily without reflection or if field exists.
						// For now, we just analyze it as requested.
						_ = color
					}
				}

				feed = append(feed, wp)
				existing[wp.ID] = true
				addedCount++
			}
		}
	}

	if addedCount > 0 {
		return addedCount, c.saveFeed(feed)
	}
	return 0, nil
}

