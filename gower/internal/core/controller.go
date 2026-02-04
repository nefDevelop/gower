package core

import (
	"fmt"
	"gower/internal/providers"
	"gower/internal/utils"
	"gower/pkg/models" // Import models package
	"io"
	"math/rand"
	"net/http"
	"net/url"
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
	jsonManager := utils.NewSecureJSONManager()
	homeDir, _ := os.UserHomeDir()

	for _, providerConfig := range config.GenericProviders {
		if providerConfig.Enabled {
			if homeDir != "" {
				parserPath := filepath.Join(homeDir, ".gower", "data", "parser", providerConfig.Name+".json")
				var mapping models.ResponseMapping
				if err := jsonManager.ReadJSON(parserPath, &mapping); err == nil {
					providerConfig.ResponseMapping = mapping
				}
			}

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

	// Load dynamic palette for color filtering
	var palette []string
	if color != "" {
		// For feed, we use the feed palette
		palette, _, _ = c.LoadColorPalettes()
	}

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
			// 1. Find which bucket the user selected (snap input to nearest palette color)
			targetBucket := c.ColorManager.FindNearestColorInPalette(color, palette)

			// 2. Find which bucket the wallpaper belongs to
			wpBucket := c.ColorManager.FindNearestColorInPalette(wp.Color, palette)

			if wpBucket != targetBucket {
				matchesColor = false
			}
		}

		if blacklistMap[wp.ID] {
			continue
		}

		if matchesSearch && matchesTheme && matchesColor {
			filteredFeed = append(filteredFeed, wp)
		}
	}

	// Algoritmo de Feed: 50% nuevos, orden aleatorio estable por 1 hora
	var unseen []models.Wallpaper
	var seen []models.Wallpaper

	for _, wp := range filteredFeed {
		if !wp.Seen {
			unseen = append(unseen, wp)
		} else {
			seen = append(seen, wp)
		}
	}

	seed := time.Now().Truncate(time.Hour).UnixNano()
	r := rand.New(rand.NewSource(seed))

	r.Shuffle(len(unseen), func(i, j int) { unseen[i], unseen[j] = unseen[j], unseen[i] })
	r.Shuffle(len(seen), func(i, j int) { seen[i], seen[j] = seen[j], seen[i] })

	var mixedFeed []models.Wallpaper
	uIdx, sIdx := 0, 0
	for uIdx < len(unseen) || sIdx < len(seen) {
		if uIdx < len(unseen) {
			mixedFeed = append(mixedFeed, unseen[uIdx])
			uIdx++
		}
		if sIdx < len(seen) {
			mixedFeed = append(mixedFeed, seen[sIdx])
			sIdx++
		}
	}

	start := (page - 1) * limit
	end := start + limit

	if start >= len(mixedFeed) {
		return []models.Wallpaper{}, nil
	}
	if end > len(mixedFeed) {
		end = len(mixedFeed)
	}

	result := mixedFeed[start:end]

	// Marcar los ítems mostrados como vistos (seen = true)
	changed := false
	idsToMark := make(map[string]bool)

	for i := range result {
		if !result[i].Seen {
			// No marcamos como visto en el objeto de retorno para que la UI pueda mostrar "[NEW]"
			idsToMark[result[i].ID] = true
			changed = true
		}
	}

	if changed {
		// Actualizar el feed original para guardar en disco
		for i := range feed {
			if idsToMark[feed[i].ID] {
				feed[i].Seen = true
			}
		}
		// Ignoramos error de guardado para no interrumpir la visualización
		_ = c.saveFeed(feed)
	}

	return result, nil
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
func (c *Controller) AnalyzeFeed(all bool, force bool) error {
	feed, err := c.loadFeed()
	if err != nil {
		return err
	}
	utils.Log.Info("Analyzing feed: %d items found", len(feed))

	// Index local wallpapers if enabled
	if c.Config.Paths.IndexWallpapers && c.Config.Paths.Wallpapers != "" {
		if err := c.indexLocalWallpapers(&feed); err != nil {
			utils.Log.Error("Error indexing local wallpapers: %v", err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	thumbDir := filepath.Join(homeDir, ".gower", "cache", "thumbs")

	type job struct {
		Controller *Controller
		Index      int
		Wp         models.Wallpaper
		Delete     bool
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
				ctrl := j.Controller

				newWp, changed, deleteItem := ctrl.processWallpaperItem(wp, force, all, thumbDir)

				if deleteItem {
					results <- job{Index: j.Index, Delete: true}
				} else if changed {
					results <- job{Index: j.Index, Wp: newWp}
				}
			}
		}()
	}

	for i, wp := range feed {
		jobs <- job{Controller: c, Index: i, Wp: wp}
	}
	close(jobs)
	wg.Wait()
	close(results)

	updatedCount := 0
	for res := range results {
		if res.Delete {
			feed[res.Index].ID = "" // Mark for deletion
		} else {
			feed[res.Index] = res.Wp
		}
		updatedCount++
	}

	// Filter out deleted items
	if updatedCount > 0 {
		newFeed := make([]models.Wallpaper, 0, len(feed))
		for _, wp := range feed {
			if wp.ID != "" {
				newFeed = append(newFeed, wp)
			}
		}
		feed = newFeed
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

// indexLocalWallpapers scans the configured wallpapers directory and updates the feed.
func (c *Controller) indexLocalWallpapers(feed *[]models.Wallpaper) error {
	localDir := c.Config.Paths.Wallpapers
	files, err := os.ReadDir(localDir)
	if err != nil {
		return err
	}

	// Map existing local items in feed
	localInFeed := make(map[string]int)
	for i, wp := range *feed {
		if wp.Source == "local" {
			localInFeed[wp.ID] = i
		}
	}

	// Track found files to detect deletions
	foundFiles := make(map[string]bool)
	addedCount := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			continue
		}

		// Generate ID: local_<filename_sanitized>
		safeName := strings.ReplaceAll(file.Name(), " ", "_")
		id := "local_" + safeName
		foundFiles[id] = true

		if _, exists := localInFeed[id]; !exists {
			// Add new local wallpaper
			fullPath := filepath.Join(localDir, file.Name())
			newWp := models.Wallpaper{
				ID:        id,
				Source:    "local",
				URL:       fullPath,
				Thumbnail: fullPath, // Use original as source for thumb generation
				Seen:      false,
				Added:     time.Now().Unix(),
			}
			*feed = append(*feed, newWp)
			addedCount++
		}
	}

	// Remove local items from feed that no longer exist on disk
	// We mark them with empty ID to be cleaned up by AnalyzeFeed's main loop
	removedCount := 0
	for id, idx := range localInFeed {
		if !foundFiles[id] {
			(*feed)[idx].ID = "" // Mark for deletion
			removedCount++
		}
	}

	utils.Log.Info("Local indexing: %d added, %d removed", addedCount, removedCount)
	return nil
}

// AnalyzeFavorites analyzes the favorites items, regenerates thumbnails/colors if needed.
func (c *Controller) AnalyzeFavorites(all bool, force bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	favPath := filepath.Join(homeDir, ".gower", "data", "favorites.json")
	thumbDir := filepath.Join(homeDir, ".gower", "cache", "thumbs")

	// Define struct locally to match JSON
	type Favorite struct {
		models.Wallpaper
		Notes string `json:"notes,omitempty"`
	}

	var favorites []Favorite
	if err := c.feedManager.ReadJSON(favPath, &favorites); err != nil {
		return err
	}

	utils.Log.Info("Analyzing favorites: %d items found", len(favorites))

	type job struct {
		Index  int
		Fav    Favorite
		Delete bool
	}

	jobs := make(chan job, len(favorites))
	results := make(chan job, len(favorites))

	workers := 5
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				fav := j.Fav
				// Process the embedded Wallpaper
				newWp, changed, deleteItem := c.processWallpaperItem(fav.Wallpaper, force, all, thumbDir)

				if deleteItem {
					results <- job{Index: j.Index, Delete: true}
				} else if changed {
					fav.Wallpaper = newWp
					results <- job{Index: j.Index, Fav: fav}
				}
			}
		}()
	}

	for i, fav := range favorites {
		jobs <- job{Index: i, Fav: fav}
	}
	close(jobs)
	wg.Wait()
	close(results)

	updatedCount := 0
	for res := range results {
		if res.Delete {
			// Mark for deletion (empty ID)
			favorites[res.Index].ID = ""
		} else {
			favorites[res.Index] = res.Fav
		}
		updatedCount++
	}

	if updatedCount > 0 {
		// Filter deleted
		newFavorites := make([]Favorite, 0, len(favorites))
		for _, fav := range favorites {
			if fav.ID != "" {
				newFavorites = append(newFavorites, fav)
			}
		}

		if err := c.feedManager.WriteJSON(favPath, newFavorites); err != nil {
			return err
		}
	}

	// Rebuild index
	return c.RebuildColorIndex()
}

// processWallpaperItem handles the analysis logic for a single wallpaper item
func (c *Controller) processWallpaperItem(wp models.Wallpaper, force, all bool, thumbDir string) (models.Wallpaper, bool, bool) {
	changed := false
	thumbPath := filepath.Join(thumbDir, wp.ID+".jpg")

	// 1. Validar dimensiones por metadatos (si existen) para limpiar items de baja resolución
	if !c.isValidDimension(wp.Dimension) {
		utils.Log.Info("Removing invalid item %s (dimension %s)", wp.ID, wp.Dimension)
		os.Remove(thumbPath)
		return wp, false, true
	}

	info, errStat := os.Stat(thumbPath)
	thumbExists := errStat == nil && info.Size() > 0

	// If thumbnail is missing, or if we are forcing a full regeneration (force+all), generate it
	if !thumbExists || force {
		utils.Log.Info("Generating thumbnail for %s", wp.ID)
		src := wp.Thumbnail
		checkResolution := false
		if src == "" || src == wp.URL {
			if localPath, found := c.findWallpaperCacheFile(wp); found {
				src = localPath // Use local file if available, it's faster
			}
			checkResolution = true
		}
		w, h, err := c.ColorManager.GenerateThumbnail(src, thumbPath)
		if err == nil {
			// Check validity immediately after generation
			if !c.isValidImage(w, h, checkResolution) {
				utils.Log.Info("Removing invalid item %s (ratio %dx%d)", wp.ID, w, h)
				os.Remove(thumbPath)
				return wp, false, true
			}

			utils.Log.Info("Successfully generated thumbnail for %s", wp.ID)
			wp.Extension = ".jpg"
			changed = true
			// If we generated it, we can set ratio if missing
			if wp.Ratio == "" && w > 0 && h > 0 {
				wp.Ratio = calculateRatio(w, h)
				changed = true
			}
			// And we can analyze color
			if color, err := c.ColorManager.AnalyzeColor(thumbPath); err == nil {
				wp.Color = color
				if c.ColorManager.IsDark(color) {
					wp.Theme = "dark"
				} else {
					wp.Theme = "light"
				}
				changed = true
			}
		} else {
			utils.Log.Error("Failed to generate thumbnail for %s: %v", wp.ID, err)
			return wp, false, true
		}
	} else {
		// Thumbnail exists, verify it matches current criteria (prune invalid items)
		w, h, err := c.ColorManager.GetImageDimensions(thumbPath)
		if err == nil {
			checkResolution := false
			if wp.Thumbnail == "" || wp.Thumbnail == wp.URL {
				checkResolution = true
			}

			if !c.isValidImage(w, h, checkResolution) {
				utils.Log.Info("Removing invalid item %s (ratio %dx%d)", wp.ID, w, h)
				os.Remove(thumbPath)
				return wp, false, true
			}

			if wp.Extension == "" {
				wp.Extension = ".jpg"
				changed = true
			}
		}

		// Re-analyze color if requested or missing
		if all || wp.Color == "" {
			if color, err := c.ColorManager.AnalyzeColor(thumbPath); err == nil {
				if wp.Color != color {
					wp.Color = color
					if c.ColorManager.IsDark(color) {
						wp.Theme = "dark"
					} else {
						wp.Theme = "light"
					}
					changed = true
				}
			} else {
				utils.Log.Error("Failed to analyze color for %s: %v", wp.ID, err)
			}
		}
	}

	// 4. Check and fix main wallpaper filename if it exists
	expectedPath, err := c.GetWallpaperLocalPath(wp)
	if err == nil {
		actualPath, found := c.findWallpaperCacheFile(wp)
		if found && actualPath != expectedPath {
			// Check if expected path already exists to avoid overwrite error on rename
			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				utils.Log.Info("Renaming cached wallpaper from %s to %s", filepath.Base(actualPath), filepath.Base(expectedPath))
				if err := os.Rename(actualPath, expectedPath); err != nil {
					utils.Log.Error("Failed to rename %s: %v", filepath.Base(actualPath), err)
				}
			} else if actualPath != expectedPath {
				// Expected path exists, and it's not the same file. Remove the old one with the bad name.
				utils.Log.Info("Warning: Found duplicate for %s. Removing old file with incorrect name: %s", wp.ID, filepath.Base(actualPath))
				os.Remove(actualPath)
			}
		}
	}

	return wp, changed, false
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

	if wallpaper.Added == 0 {
		wallpaper.Added = time.Now().Unix()
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
			if wp.Added == 0 {
				wp.Added = time.Now().Unix()
			}
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

// RemoveFromBlacklist removes an ID from the blacklist.
func (c *Controller) RemoveFromBlacklist(id string) error {
	blacklist, err := c.loadBlacklist()
	if err != nil {
		return err
	}

	newBlacklist := make([]string, 0, len(blacklist))
	found := false
	for _, existing := range blacklist {
		if existing == id {
			found = true
			continue
		}
		newBlacklist = append(newBlacklist, existing)
	}

	if !found {
		return fmt.Errorf("ID %s not found in blacklist", id)
	}

	path, err := c.getBlacklistPath()
	if err != nil {
		return err
	}
	if err := c.feedManager.WriteJSON(path, newBlacklist); err != nil {
		return err
	}
	utils.Log.Info("Removed wallpaper %s from blacklist", id)
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
	// Parse URL to safely get the extension from the path, ignoring query parameters.
	parsedURL, err := url.Parse(wp.URL)
	var ext string
	if err == nil {
		ext = filepath.Ext(parsedURL.Path)
	} else {
		// Fallback for invalid URLs, though less likely
		ext = filepath.Ext(wp.URL)
	}

	if ext == "" {
		ext = ".jpg" // Default extension
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
	thumbPath := filepath.Join(thumbDir, wp.ID+".jpg")
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

// findWallpaperCacheFile finds the local cache file for a wallpaper, even if it has a bad name.
// It returns the full path to the file and a boolean indicating if it was found.
func (c *Controller) findWallpaperCacheFile(wp models.Wallpaper) (string, bool) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	wallpaperCacheDir := filepath.Join(homeDir, ".gower", "cache", "wallpapers")
	safeID := strings.ReplaceAll(wp.ID, "/", "_")

	// Glob for files starting with the safe ID
	matches, err := filepath.Glob(filepath.Join(wallpaperCacheDir, safeID+"*"))
	if err != nil || len(matches) == 0 {
		// As a fallback, check if the URL is a local file path itself
		if _, err := os.Stat(wp.URL); err == nil {
			return wp.URL, true
		}
		return "", false
	}

	// Return the first match. This assumes one wallpaper ID doesn't have multiple cached files.
	return matches[0], true
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
				// Filtrar por metadatos antes de intentar descargar nada
				if !c.isValidDimension(wp.Dimension) {
					continue
				}

				thumbPath := filepath.Join(thumbDir, wp.ID+".jpg")

				// Usar thumbnail URL si existe para ahorrar ancho de banda
				src := wp.Thumbnail
				checkResolution := false
				if src == "" || src == wp.URL {
					src = wp.URL
					checkResolution = true
				}

				// Generar thumbnail y analizar color
				width, height, err := c.ColorManager.GenerateThumbnail(src, thumbPath)
				if err == nil {
					// Validar Ratio antes de continuar
					if !c.isValidImage(width, height, checkResolution) {
						utils.Log.Info("Rejected item %s: dimensions %dx%d do not match criteria. Removing thumbnail.", wp.ID, width, height)
						os.Remove(thumbPath) // Limpiar thumbnail generado
						continue
					}

					wp.Extension = ".jpg"

					// Calcular Ratio si falta
					if wp.Ratio == "" && width > 0 && height > 0 {
						wp.Ratio = calculateRatio(width, height)
					}

					if color, err := c.ColorManager.AnalyzeColor(thumbPath); err == nil {
						wp.Color = color
						if c.ColorManager.IsDark(color) {
							wp.Theme = "dark"
						} else {
							wp.Theme = "light"
						}
					}

					// Marcar como no visto
					wp.Seen = false

					results <- wp
				} else {
					utils.Log.Info("Failed to download/generate thumbnail for %s: %v", wp.ID, err)
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
					wp.Added = feed[i].Added // Preserve added time
					feed[i] = wp
					break
				}
			}
			repairedCount++
		} else {
			wp.Added = time.Now().Unix()
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

// RebuildColorIndex rebuilds the colors.json file generating a dynamic palette.
func (c *Controller) RebuildColorIndex() error {
	feed, err := c.loadFeed()
	if err != nil {
		return err
	}
	return c.rebuildColorsIndex(feed)
}

func (c *Controller) rebuildColorsIndex(feed []models.Wallpaper) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(homeDir, ".gower", "data", "colors.json")

	// Collect Feed colors
	var feedColors []string
	for _, wp := range feed {
		if wp.Color != "" {
			feedColors = append(feedColors, wp.Color)
		}
	}

	// Collect Favorites colors
	favPath := filepath.Join(homeDir, ".gower", "data", "favorites.json")
	var favorites []struct {
		models.Wallpaper
		Notes string `json:"notes,omitempty"`
	}
	var favColors []string
	if err := c.feedManager.ReadJSON(favPath, &favorites); err == nil {
		for _, fav := range favorites {
			if fav.Color != "" {
				favColors = append(favColors, fav.Color)
			}
		}
	}

	// Generate separate palettes
	feedPalette := c.ColorManager.GenerateDynamicPalette(feedColors, 16)
	favPalette := c.ColorManager.GenerateDynamicPalette(favColors, 16)

	output := struct {
		FeedPalette      []string `json:"feed_palette"`
		FavoritesPalette []string `json:"favorites_palette"`
	}{
		FeedPalette:      feedPalette,
		FavoritesPalette: favPalette,
	}

	return c.feedManager.WriteJSON(path, output)
}

// LoadColorPalettes loads the generated palettes from colors.json
func (c *Controller) LoadColorPalettes() ([]string, []string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	path := filepath.Join(homeDir, ".gower", "data", "colors.json")

	var data struct {
		FeedPalette      []string `json:"feed_palette"`
		FavoritesPalette []string `json:"favorites_palette"`
	}

	if err := c.feedManager.ReadJSON(path, &data); err != nil {
		return nil, nil, err
	}
	return data.FeedPalette, data.FavoritesPalette, nil
}

func (c *Controller) isValidImage(width, height int, checkResolution bool) bool {
	if c.Config == nil {
		return true
	}

	if checkResolution && c.Config.Search.MinWidth > 0 && width < c.Config.Search.MinWidth {
		return false
	}
	if checkResolution && c.Config.Search.MinHeight > 0 && height < c.Config.Search.MinHeight {
		return false
	}

	if c.Config.Search.AspectRatio == "" {
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

func (c *Controller) isValidDimension(dimension string) bool {
	if c.Config == nil || dimension == "" {
		return true // Si no hay datos, asumimos válido (ej. NASA a veces)
	}
	parts := strings.Split(dimension, "x")
	if len(parts) != 2 {
		return true
	}
	w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return true
	}

	if c.Config.Search.MinWidth > 0 && w < c.Config.Search.MinWidth {
		return false
	}
	if c.Config.Search.MinHeight > 0 && h < c.Config.Search.MinHeight {
		return false
	}
	return true
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
