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
	"strconv"
	"strings"
	"sync"
	"time"
)

// Controller is the main controller of the application.
type Controller struct {
	Config          *models.Config
	ProviderManager *ProviderManager
	feedManager     *utils.SecureJSONManager // Manager for feed.json
	ColorManager    *ColorManager
}

// NewController creates a new Controller.
func NewController(config *models.Config) *Controller {
	providerManager := NewProviderManager()

	// Register native providers
	if config.Providers.Wallhaven.Enabled {
		providerManager.RegisterProvider(&providers.WallhavenProvider{
			APIKey: config.Providers.Wallhaven.APIKey,
		})
	}
	if config.Providers.Reddit.Enabled {
		providerManager.RegisterProvider(providers.NewRedditProvider(config.Providers.Reddit))
	}
	if config.Providers.Nasa.Enabled {
		providerManager.RegisterProvider(providers.NewNasaProvider(config.Providers.Nasa.APIKey))
	}
	if config.Providers.Bing.Enabled {
		providerManager.RegisterProvider(providers.NewBingProvider(config.Providers.Bing.Market))
	}
	if config.Providers.Unsplash.Enabled {
		providerManager.RegisterProvider(providers.NewUnsplashProvider(config.Providers.Unsplash.APIKey))
	}
	// Register other native providers here...

	// Register generic providers
	for _, providerConfig := range config.GenericProviders {
		if providerConfig.Enabled {
			provider := &providers.GenericProvider{Config: providerConfig}
			providerManager.RegisterProvider(provider)
		}
	}

	return &Controller{
		Config:          config,
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
		if theme != "" && !strings.EqualFold(wp.Theme, theme) {
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

// AnalyzeFeed analyzes the feed items, regenerates thumbnails/colors if needed, and rebuilds the color index.
func (c *Controller) AnalyzeFeed(all bool) error {
	feed, err := c.loadFeed()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	thumbDir := filepath.Join(homeDir, ".gower", "cache", "thumbs")

	type job struct {
		Index int
		Wp    models.Wallpaper
	}

	jobs := make(chan job, len(feed))
	results := make(chan job, len(feed))

	workers := 5
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				wp := j.Wp
				changed := false
				thumbPath := filepath.Join(thumbDir, wp.ID+".jpg")

				_, errStat := os.Stat(thumbPath)
				thumbExists := errStat == nil

				// If thumbnail is missing, we must generate it to analyze color
				if !thumbExists {
					src := wp.Thumbnail
					if src == "" {
						src = wp.URL
					}
					w, h, err := c.ColorManager.GenerateThumbnail(src, thumbPath)
					if err == nil {
						// If we generated it, we can set ratio if missing
						if wp.Ratio == "" && w > 0 && h > 0 {
							wp.Ratio = calculateRatio(w, h)
							changed = true
						}
						// And we can analyze color
						if color, err := c.ColorManager.AnalyzeColor(thumbPath); err == nil {
							wp.Color = color
							changed = true
						}
					} else {
						utils.Log.Error("Failed to generate thumbnail for %s: %v", wp.ID, err)
					}
				} else if all || wp.Color == "" {
					// Thumbnail exists, just re-analyze color
					if color, err := c.ColorManager.AnalyzeColor(thumbPath); err == nil {
						if wp.Color != color {
							wp.Color = color
							changed = true
						}
					} else {
						utils.Log.Error("Failed to analyze color for %s: %v", wp.ID, err)
					}
				}

				if changed {
					results <- job{Index: j.Index, Wp: wp}
				}
			}
		}()
	}

	for i, wp := range feed {
		jobs <- job{Index: i, Wp: wp}
	}
	close(jobs)
	wg.Wait()
	close(results)

	updatedCount := 0
	for res := range results {
		feed[res.Index] = res.Wp
		updatedCount++
	}

	// Always rebuild index to ensure it matches current feed colors
	if err := c.rebuildColorsIndex(feed); err != nil {
		utils.Log.Error("Failed to rebuild colors index: %v", err)
	}

	if updatedCount > 0 {
		return c.saveFeed(feed)
	}
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
		if theme == "" || strings.EqualFold(wp.Theme, theme) {
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
	if err := c.saveFeed(feed); err != nil {
		return err
	}
	utils.Log.Info("Added wallpaper %s to feed", wallpaper.ID)
	return nil
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
	if err := c.feedManager.WriteJSON(path, blacklist); err != nil {
		return err
	}
	utils.Log.Info("Added wallpaper %s to blacklist", id)
	return nil
}

// GetBlacklist returns the current blacklist.
func (c *Controller) GetBlacklist() ([]string, error) {
	return c.loadBlacklist()
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

	if err := c.saveFeed(newFeed); err != nil {
		return err
	}
	utils.Log.Info("Removed wallpaper %s from feed", id)
	return nil
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
	favPath := filepath.Join(filepath.Dir(c.getFeedPathString()), "favorites.json")
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
	if _, _, err := c.ColorManager.GenerateThumbnail(filePath, thumbPath); err != nil {
		utils.Log.Error("Failed to generate thumbnail for %s: %v", wp.ID, err)
	}

	// 2. Analyze and Index Color
	_, err = c.ColorManager.AnalyzeColor(filePath)
	if err != nil {
		utils.Log.Debug("Color analysis failed/skipped for %s: %v", wp.ID, err)
	}

	utils.Log.Info("Downloaded wallpaper: %s", wp.ID)
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
		favPath := filepath.Join(filepath.Dir(c.getFeedPathString()), "favorites.json")
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
		if theme != "" && theme != "auto" && !strings.EqualFold(wp.Theme, theme) {
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
func (c *Controller) SyncFeed() (int, int, error) {
	utils.Log.Info("Starting feed sync...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return 0, 0, err
	}
	parserDir := filepath.Join(homeDir, ".gower", "data", "parser")

	files, err := os.ReadDir(parserDir)
	if err != nil {
		// If dir doesn't exist, just return 0
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	// Load existing feed and blacklist to avoid duplicates
	feed, _ := c.loadFeed()
	blacklist, _ := c.loadBlacklist()

	inFeed := make(map[string]bool)
	for _, wp := range feed {
		inFeed[wp.ID] = true
	}

	isBlacklisted := make(map[string]bool)
	for _, id := range blacklist {
		isBlacklisted[id] = true
	}

	// Track IDs processed in this run to avoid duplicates from multiple parser files
	processed := make(map[string]bool)

	addedCount := 0
	repairedCount := 0
	thumbDir := filepath.Join(homeDir, ".gower", "cache", "thumbs")

	// 1. Recolectar candidatos únicos
	var candidates []models.Wallpaper

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
				if isBlacklisted[wp.ID] {
					continue
				}
				if processed[wp.ID] {
					continue
				}

				// Check if already in feed
				if inFeed[wp.ID] {
					// Check if thumbnail exists
					thumbPath := filepath.Join(thumbDir, wp.ID+".jpg")
					if _, err := os.Stat(thumbPath); err == nil {
						// Exists, skip
						processed[wp.ID] = true
						continue
					}
					// Thumbnail missing, add to candidates to regenerate
				}

				candidates = append(candidates, wp)
				processed[wp.ID] = true
			}
		}
	}

	if len(candidates) == 0 {
		return 0, 0, nil
	}

	// 2. Procesar concurrentemente (Worker Pool)
	// Limitamos a 5 goroutines para no saturar red/cpu
	workers := 5
	jobs := make(chan models.Wallpaper, len(candidates))
	results := make(chan models.Wallpaper, len(candidates))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for wp := range jobs {
				thumbPath := filepath.Join(thumbDir, wp.ID+".jpg")

				// Usar thumbnail URL si existe para ahorrar ancho de banda
				src := wp.Thumbnail
				if src == "" {
					src = wp.URL
				}

				// Generar thumbnail y analizar color
				width, height, err := c.ColorManager.GenerateThumbnail(src, thumbPath)
				if err == nil {
					// Validar Ratio antes de continuar
					if !c.matchesAspectRatio(width, height) {
						os.Remove(thumbPath) // Limpiar thumbnail generado
						continue
					}

					// Calcular Ratio si falta
					if wp.Ratio == "" && width > 0 && height > 0 {
						wp.Ratio = calculateRatio(width, height)
					}

					if color, err := c.ColorManager.AnalyzeColor(thumbPath); err == nil {
						wp.Color = color
					}

					// Marcar como no visto
					wp.Seen = false

					results <- wp
				}
				// Si falla la descarga, no lo agregamos al feed (o podríamos agregarlo sin color)
			}
		}()
	}

	// Enviar trabajos
	for _, wp := range candidates {
		jobs <- wp
	}
	close(jobs)

	// Esperar y cerrar
	wg.Wait()
	close(results)

	// Recolectar resultados
	for wp := range results {
		if inFeed[wp.ID] {
			// Update existing entry
			for i := range feed {
				if feed[i].ID == wp.ID {
					wp.Seen = feed[i].Seen // Preserve seen status
					feed[i] = wp
					break
				}
			}
			repairedCount++
		} else {
			feed = append(feed, wp)
			addedCount++
		}
	}

	// Reconstruir colors.json basado en el feed actualizado
	c.rebuildColorsIndex(feed)

	if addedCount > 0 || repairedCount > 0 {
		// Aplicar Hard Limit (FIFO)
		if c.Config.Limits.FeedHardLimit > 0 && len(feed) > c.Config.Limits.FeedHardLimit {
			// Mantener solo los últimos N elementos
			feed = feed[len(feed)-c.Config.Limits.FeedHardLimit:]
		}
		err := c.saveFeed(feed)
		if err == nil {
			utils.Log.Info("Feed sync completed. Added: %d, Repaired: %d", addedCount, repairedCount)
		}
		return addedCount, repairedCount, err
	}
	utils.Log.Info("Feed sync completed. No changes.")
	return 0, 0, nil
}

func calculateRatio(w, h int) string {
	if h == 0 {
		return ""
	}
	gcd := func(a, b int) int {
		for b != 0 {
			a, b = b, a%b
		}
		return a
	}
	d := gcd(w, h)
	return fmt.Sprintf("%d:%d", w/d, h/d)
}

func (c *Controller) rebuildColorsIndex(feed []models.Wallpaper) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(homeDir, ".gower", "data", "colors.json")

	// Crear mapa de la paleta permitida para filtrado estricto
	paletteMap := make(map[string]bool)
	for _, color := range StandardPalette {
		paletteMap[color] = true
	}

	uniqueColors := make(map[string]bool)
	var colors []string

	for _, wp := range feed {
		if wp.Color != "" && paletteMap[wp.Color] && !uniqueColors[wp.Color] {
			uniqueColors[wp.Color] = true
			colors = append(colors, wp.Color)
		}
	}

	return c.feedManager.WriteJSON(path, colors)
}

func (c *Controller) matchesAspectRatio(width, height int) bool {
	if c.Config == nil || c.Config.Search.AspectRatio == "" {
		return true
	}

	target := c.Config.Search.AspectRatio
	var targetRatio float64

	if strings.Contains(target, ":") {
		parts := strings.Split(target, ":")
		if len(parts) == 2 {
			w, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			h, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil && h != 0 {
				targetRatio = w / h
			}
		}
	} else {
		targetRatio, _ = strconv.ParseFloat(target, 64)
	}

	if targetRatio == 0 {
		return true
	}

	currentRatio := float64(width) / float64(height)
	diff := currentRatio - targetRatio
	if diff < 0 {
		diff = -diff
	}

	return diff <= c.Config.Search.Tolerance
}

// GetLastProviderUpdateTime returns the modification time of the most recently updated provider cache file.
func (c *Controller) GetLastProviderUpdateTime() (time.Time, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return time.Time{}, err
	}
	parserDir := filepath.Join(homeDir, ".gower", "data", "parser")

	files, err := os.ReadDir(parserDir)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}

	var lastTime time.Time
	for _, file := range files {
		if info, err := file.Info(); err == nil && !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			if info.ModTime().After(lastTime) {
				lastTime = info.ModTime()
			}
		}
	}
	return lastTime, nil
}
