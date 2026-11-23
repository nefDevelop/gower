package core

import (
	crand "crypto/rand"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gower/internal/providers"
	"gower/internal/utils"
	"gower/pkg/models"
)

// Controller is the main controller of the application.
type Controller struct {
	Config          *models.Config
	ProviderManager *ProviderManager
	feedManager     *utils.SecureJSONManager
	ColorManager    *ColorManager
	DownloadService *DownloadService
	FeedService     *FeedService
	AnalysisService *AnalysisService
	Log             *utils.Logger
}

func (c *Controller) log() *utils.Logger {
	if c.Log != nil {
		return c.Log
	}
	return utils.Log
}

func (c *Controller) getWorkers() int {
	if c.Config != nil && c.Config.Limits.AnalysisWorkers > 0 {
		return c.Config.Limits.AnalysisWorkers
	}
	return 5
}

type FeedCache struct {
	Hour int64    `json:"hour"`
	IDs  []string `json:"ids"`
}

// FavoriteWallpaper represents a wallpaper in the favorites list with optional notes.
type FavoriteWallpaper struct {
	models.Wallpaper
	Notes string `json:"notes,omitempty"`
}

// GetAppDir returns the application directory, preferring XDG but falling back to legacy .gower
func GetAppDir() (string, error) {
	return utils.AppDir()
}

// NewController creates a new Controller.
var NewController = func(config *models.Config) *Controller {
	providerManager := NewProviderManager()

	// Register native providers
	if config.Providers.Wallhaven.Enabled {
		rl := config.Providers.Wallhaven.RateLimit
		providerManager.RegisterProvider(&providers.WallhavenProvider{
			APIKey:      config.Providers.Wallhaven.APIKey,
			RateLimiter: utils.NewRateLimiter(rl.Requests, rl.PerSeconds),
		})
	}
	if config.Providers.Reddit.Enabled {
		providerManager.RegisterProvider(&providers.RedditProvider{
			Config:      config.Providers.Reddit,
			RateLimiter: utils.NewRateLimiter(30, 60),
		})
	}
	if config.Providers.Nasa.Enabled {
		providerManager.RegisterProvider(&providers.NasaProvider{
			APIKey:      config.Providers.Nasa.APIKey,
			RateLimiter: utils.NewRateLimiter(30, 60),
		})
	}
	if config.Providers.Bing.Enabled {
		providerManager.RegisterProvider(&providers.BingProvider{
			Market:      config.Providers.Bing.Market,
			RateLimiter: utils.NewRateLimiter(10, 60),
		})
	}
	if config.Providers.Unsplash.Enabled {
		providerManager.RegisterProvider(&providers.UnsplashProvider{
			APIKey:      config.Providers.Unsplash.APIKey,
			RateLimiter: utils.NewRateLimiter(30, 60),
		})
	}

	// Register generic providers. La comprobación de disponibilidad se hacía
	// aquí con un HEAD secuencial y con los placeholders {query}/{apikey} sin
	// sustituir: bloqueaba el arranque y podía omitir proveedores válidos. La
	// primera búsqueda real ya reporta el error si la API no responde.
	jsonManager := utils.NewSecureJSONManager()
	appDir, _ := GetAppDir()

	for _, providerConfig := range config.GenericProviders {
		if !providerConfig.Enabled {
			continue
		}
		if appDir != "" {
			parserPath := filepath.Join(appDir, "data", "parser", utils.SafeFilename(providerConfig.Name)+".json")
			var mapping models.ResponseMapping
			if err := jsonManager.ReadJSON(parserPath, &mapping); err == nil {
				providerConfig.ResponseMapping = mapping
			}
		}

		providerManager.RegisterProvider(&providers.GenericProvider{
			Config:      providerConfig,
			RateLimiter: utils.NewRateLimiter(30, 60),
		})
	}

	feedManager := utils.NewSecureJSONManager()
	colorManager := NewColorManager()

	return &Controller{
		Config:          config,
		ProviderManager: providerManager,
		feedManager:     feedManager,
		ColorManager:    colorManager,
		DownloadService: NewDownloadService(config, colorManager, feedManager),
		FeedService:     NewFeedService(config, feedManager),
		AnalysisService: NewAnalysisService(config, colorManager, feedManager),
	}
}

func (c *Controller) getFeedPath() (string, error) {
	if c.FeedService == nil {
		return "", fmt.Errorf("feed service not initialized")
	}
	return c.FeedService.getFeedPath()
}

func (c *Controller) getFeedCachePath() (string, error) {
	if c.FeedService == nil {
		return "", fmt.Errorf("feed service not initialized")
	}
	return c.FeedService.getFeedCachePath()
}

func (c *Controller) getBlacklistPath() (string, error) {
	if c.FeedService == nil {
		return "", fmt.Errorf("feed service not initialized")
	}
	return c.FeedService.getBlacklistPath()
}

func (c *Controller) withFeedLock(fn func() error) error {
	path, err := c.getFeedPath()
	if err != nil {
		return err
	}
	return utils.WithFileLock(path, fn)
}

func (c *Controller) withBlacklistLock(fn func() error) error {
	path, err := c.getBlacklistPath()
	if err != nil {
		return err
	}
	return utils.WithFileLock(path, fn)
}

func (c *Controller) withFavoritesLock(fn func() error) error {
	if c.FeedService == nil {
		return fmt.Errorf("feed service not initialized")
	}
	path := c.FeedService.getFavoritesPath()
	if path == "" {
		return fmt.Errorf("cannot resolve favorites path")
	}
	return utils.WithFileLock(path, fn)
}

func (c *Controller) loadBlacklist() ([]string, error) {
	if c.FeedService == nil {
		return nil, fmt.Errorf("feed service not initialized")
	}
	return c.FeedService.loadBlacklist()
}

func (c *Controller) loadFeed() ([]models.Wallpaper, error) {
	if c.FeedService == nil {
		return nil, fmt.Errorf("feed service not initialized")
	}
	return c.FeedService.loadFeed()
}

func (c *Controller) saveFeed(feed []models.Wallpaper) error {
	if c.FeedService == nil {
		return fmt.Errorf("feed service not initialized")
	}
	return c.FeedService.saveFeed(feed)
}

// GetFeed retrieves wallpapers from the feed with pagination and optional search/theme filters.
func (c *Controller) GetFeed(page, limit int, search, theme, color, sortMode string, refresh bool) ([]models.Wallpaper, error) {
	if page < 1 {
		return nil, fmt.Errorf("page must be >= 1")
	}
	if limit < 1 {
		return nil, fmt.Errorf("limit must be >= 1")
	}

	var result []models.Wallpaper
	err := c.withFeedLock(func() error {
		var err error
		result, err = c.getFeed(page, limit, search, theme, color, sortMode, refresh)
		return err
	})
	return result, err
}

// getFeed contiene la implementación de GetFeed. Debe llamarse con el lock del
// feed tomado.
func (c *Controller) getFeed(page, limit int, search, theme, color, sortMode string, refresh bool) ([]models.Wallpaper, error) {
	feed, err := c.loadFeed()
	if err != nil {
		return nil, err
	}

	blacklist, err := c.loadBlacklist()
	if err != nil {
		return nil, err
	}
	blacklistMap := make(map[string]bool)
	for _, id := range blacklist {
		blacklistMap[id] = true
	}

	var filteredFeed []models.Wallpaper

	// Load dynamic palette for color filtering.
	var palette []string
	targetBucket := ""
	if color != "" {
		var errPalette error
		palette, _, errPalette = c.LoadColorPalettes()
		if errPalette != nil {
			return nil, fmt.Errorf("color filter unavailable: %w", errPalette)
		}
		if len(palette) == 0 {
			return nil, fmt.Errorf("color filter unavailable: no color palette (run 'gower feed analyze' first)")
		}
		// El bucket objetivo se calcula una sola vez, no por wallpaper.
		targetBucket = c.ColorManager.FindNearestColorInPalette(color, palette)
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
			wpBucket := c.ColorManager.FindNearestColorInPalette(wp.Color, palette)
			if wpBucket != targetBucket {
				matchesColor = false
			}
		}

		if matchesSearch && matchesTheme && matchesColor {
			filteredFeed = append(filteredFeed, wp)
		}
	}

	var mixedFeed []models.Wallpaper

	switch sortMode {
	case "newest":
		sort.Slice(filteredFeed, func(i, j int) bool {
			return filteredFeed[i].Added > filteredFeed[j].Added
		})
		mixedFeed = filteredFeed
	case "oldest":
		sort.Slice(filteredFeed, func(i, j int) bool {
			return filteredFeed[i].Added < filteredFeed[j].Added
		})
		mixedFeed = filteredFeed
	case "source":
		sort.Slice(filteredFeed, func(i, j int) bool {
			if filteredFeed[i].Source == filteredFeed[j].Source {
				return filteredFeed[i].Added > filteredFeed[j].Added
			}
			return filteredFeed[i].Source < filteredFeed[j].Source
		})
		mixedFeed = filteredFeed
	case "unseen":
		sort.Slice(filteredFeed, func(i, j int) bool {
			if filteredFeed[i].Seen != filteredFeed[j].Seen {
				return !filteredFeed[i].Seen // Unseen (false) comes before Seen (true)
			}
			return filteredFeed[i].Added > filteredFeed[j].Added
		})
		mixedFeed = filteredFeed
	case "random":
		r := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec // G404: Cryptographically secure random is not required for wallpaper shuffling.
		r.Shuffle(len(filteredFeed), func(i, j int) {
			filteredFeed[i], filteredFeed[j] = filteredFeed[j], filteredFeed[i]
		})
		mixedFeed = filteredFeed
	default: // "smart" or empty
		// Algoritmo de Feed: 50% nuevos, orden aleatorio estable por 1 hora (o forzado con refresh)

		cachePath, _ := c.getFeedCachePath()
		currentHour := time.Now().Truncate(time.Hour).Unix()
		var cachedIDs []string
		useCache := false
		shouldCache := search == "" && theme == "" && color == ""

		if !refresh && cachePath != "" && shouldCache {
			var cache FeedCache
			if err := c.feedManager.ReadJSON(cachePath, &cache); err == nil {
				if cache.Hour == currentHour {
					cachedIDs = cache.IDs
					useCache = true
				}
			}
		}

		if useCache {
			wpMap := make(map[string]models.Wallpaper)
			for _, wp := range filteredFeed {
				wpMap[wp.ID] = wp
			}

			for _, id := range cachedIDs {
				if wp, exists := wpMap[id]; exists {
					mixedFeed = append(mixedFeed, wp)
					delete(wpMap, id)
				}
			}

			var remaining []models.Wallpaper
			for _, wp := range filteredFeed {
				if _, exists := wpMap[wp.ID]; exists {
					remaining = append(remaining, wp)
				}
			}
			mixedFeed = append(mixedFeed, remaining...)
		} else {
			var unseen []models.Wallpaper
			var seen []models.Wallpaper

			for _, wp := range filteredFeed {
				if !wp.Seen {
					unseen = append(unseen, wp)
				} else {
					seen = append(seen, wp)
				}
			}

			var seed int64 //nolint:gosec // G404: Cryptographically secure random is not required for feed sorting.
			if shouldCache {
				seed = time.Now().UnixNano()
			} else {
				seed = time.Now().Truncate(time.Hour).UnixNano()
				if refresh {
					seed = time.Now().UnixNano()
				}
			} //nolint:gosec // G404: Cryptographically secure random is not required for feed sorting.
			r := rand.New(rand.NewSource(seed))

			r.Shuffle(len(unseen), func(i, j int) { unseen[i], unseen[j] = unseen[j], unseen[i] })
			r.Shuffle(len(seen), func(i, j int) { seen[i], seen[j] = seen[j], seen[i] })

			// Interleave
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

			if cachePath != "" && shouldCache {
				var ids []string
				for _, wp := range mixedFeed {
					ids = append(ids, wp.ID)
				}
				cache := FeedCache{
					Hour: currentHour,
					IDs:  ids,
				}
				if err := c.feedManager.WriteJSON(cachePath, cache); err != nil {
					c.log().Error("Failed to write feed cache: %v", err)
				}
			}
		}
	}

	// Filter blacklist AFTER shuffle to maintain stable order for non-blacklisted items
	var finalFeed []models.Wallpaper
	for _, wp := range mixedFeed {
		if !blacklistMap[wp.ID] {
			finalFeed = append(finalFeed, wp)
		}
	}

	start := (page - 1) * limit
	end := start + limit

	if start >= len(finalFeed) {
		return []models.Wallpaper{}, nil
	}
	if end > len(finalFeed) {
		end = len(finalFeed)
	}

	result := finalFeed[start:end]

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
		// no interrumpir la visualización si falla el guardado
		if err := c.saveFeed(feed); err != nil {
			c.log().Error("Failed to save feed seen state: %v", err)
		}
	}

	return result, nil
}

// SearchFeed searches the feed for wallpapers matching a query.
func (c *Controller) SearchFeed(query string, page, limit int, theme string) ([]models.Wallpaper, error) {
	// Reuse GetFeed with the search parameter
	return c.GetFeed(page, limit, query, theme, "", "smart", false)
}

// PurgeFeed clears all entries from the feed.
func (c *Controller) PurgeFeed() error {
	return c.withFeedLock(func() error {
		if err := c.saveFeed([]models.Wallpaper{}); err != nil {
			return err
		}
		// Los IDs cacheados quedarían obsoletos tras vaciar el feed.
		if cachePath, err := c.getFeedCachePath(); err == nil {
			if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
				c.log().Error("Failed to remove feed cache: %v", err)
			}
		}
		return c.rebuildColorsIndex([]models.Wallpaper{})
	})
}

// AnalyzeFeed analyzes the feed items, regenerates thumbnails/colors if needed, and rebuilds the color index.
func (c *Controller) AnalyzeFeed(all bool, force bool, progress func(string)) error {
	if c == nil {
		return fmt.Errorf("controller is nil")
	}
	return c.withFeedLock(func() error {
		return c.analyzeFeed(all, force, progress)
	})
}

// analyzeFeed contiene la implementación de AnalyzeFeed. Debe llamarse con el
// lock del feed tomado.
func (c *Controller) analyzeFeed(all bool, force bool, progress func(string)) error {
	feed, err := c.loadFeed()
	if err != nil {
		return err
	}
	c.log().Info("Analyzing feed: %d items found", len(feed))

	changedByIndexing := false
	if c.Config.Paths.IndexWallpapers && c.Config.Paths.Wallpapers != "" {
		added, reconciled, removed, err := c.indexLocalWallpapers(&feed)
		if err != nil {
			c.log().Error("Error indexing local wallpapers: %v", err)
			if progress != nil {
				progress(fmt.Sprintf("Error indexing local wallpapers: %v", err))
			}
		} else {
			c.log().Info("Local indexing results: %d added, %d reconciled, %d removed", added, reconciled, removed)
			if progress != nil {
				progress(fmt.Sprintf("Local indexing results: %d added, %d reconciled, %d removed", added, reconciled, removed))
			}
			changedByIndexing = added > 0 || reconciled > 0 || removed > 0
		}
	}

	appDir, err := GetAppDir()
	if err != nil {
		return err
	}
	thumbDir := filepath.Join(appDir, "cache", "thumbs")

	type job struct {
		Index  int
		Wp     models.Wallpaper
		Delete bool
	}

	jobs := make(chan job, len(feed))
	results := make(chan job, len(feed))

	workers := c.getWorkers()
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				newWp, changed, deleteItem := c.processWallpaperItem(j.Wp, force, all, thumbDir, progress)
				if deleteItem {
					results <- job{Index: j.Index, Delete: true}
				} else if changed {
					results <- job{Index: j.Index, Wp: newWp}
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
		if res.Delete {
			feed[res.Index].ID = ""
		} else {
			feed[res.Index] = res.Wp
		}
		updatedCount++
	}

	if updatedCount > 0 || changedByIndexing {
		newFeed := make([]models.Wallpaper, 0, len(feed))
		for _, wp := range feed {
			if wp.ID != "" {
				newFeed = append(newFeed, wp)
			}
		}
		feed = newFeed

		if err := c.saveFeed(feed); err != nil {
			return err
		}
	}

	return c.rebuildColorsIndex(feed)
}

// indexLocalWallpapers scans the configured wallpapers directory and updates the feed.
func (c *Controller) indexLocalWallpapers(feed *[]models.Wallpaper) (int, int, int, error) {
	localDir := c.Config.Paths.Wallpapers
	c.log().Debug("Scanning local directory: %s", localDir)
	files, err := os.ReadDir(localDir)
	if err != nil {
		return 0, 0, 0, err
	}

	// Map existing items in feed by ID for quick lookup and reconciliation
	feedMap := make(map[string]int)
	for i, wp := range *feed {
		feedMap[wp.ID] = i
	}

	// Track found IDs to detect deletions of "local" source items
	foundIDs := make(map[string]bool)
	addedCount := 0
	reconciledCount := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ext := filepath.Ext(file.Name())
		lowerExt := strings.ToLower(ext)
		if lowerExt != ".jpg" && lowerExt != ".jpeg" && lowerExt != ".png" && lowerExt != ".webp" {
			continue
		}

		filename := file.Name()
		fullPath := filepath.Join(localDir, filename)

		// Potential ID is the filename without extension
		baseName := strings.TrimSuffix(filename, ext)

		// Remove theme tags if present ([d] or [l])
		idFromFilename := strings.TrimSuffix(baseName, " [d]")
		idFromFilename = strings.TrimSuffix(idFromFilename, " [l]")

		// Sanitize
		idFromFilename = strings.ReplaceAll(idFromFilename, " ", "_")

		// Check if this file matches an existing ID in the feed
		targetIdx := -1
		if idx, ok := feedMap[idFromFilename]; ok {
			targetIdx = idx
		} else if idx, ok := feedMap[filename]; ok { // Legacy check for local files where ID was full filename
			targetIdx = idx
		}

		if targetIdx != -1 {
			wp := &(*feed)[targetIdx]
			foundIDs[wp.ID] = true

			// Reconcile path. We prioritize the collection folder path.
			// If the current path is not already pointing to this specific file in the collection, update it.
			if wp.Path != fullPath {
				if wp.Path != "" {
					c.log().Info("Wallpaper %s: Found in collection folder. Updating path from '%s' to '%s'", wp.ID, wp.Path, fullPath)
				}
				wp.Path = fullPath
				reconciledCount++
			}
		} else {
			// Add new local wallpaper
			newWp := models.Wallpaper{
				ID:        idFromFilename,
				Source:    "local",
				URL:       fullPath,
				Thumbnail: fullPath,
				Seen:      false,
				Added:     time.Now().Unix(),
			}
			*feed = append(*feed, newWp)
			feedMap[newWp.ID] = len(*feed) - 1
			foundIDs[newWp.ID] = true
			addedCount++
		}
	}

	removedCount := 0
	for i := range *feed {
		wp := &(*feed)[i]
		if wp.Source == "local" && wp.ID != "" {
			if !foundIDs[wp.ID] {
				wp.ID = "" // Mark for deletion
				removedCount++
			}
		}
	}

	c.log().Info("Local indexing: %d added, %d reconciled, %d removed", addedCount, reconciledCount, removedCount)
	return addedCount, reconciledCount, removedCount, nil
}

// AnalyzeFavorites analyzes the favorites items, regenerates thumbnails/colors if needed.
func (c *Controller) AnalyzeFavorites(all bool, force bool, progress func(string)) error {
	if c == nil {
		return fmt.Errorf("controller is nil")
	}
	return c.withFavoritesLock(func() error {
		return c.analyzeFavorites(all, force, progress)
	})
}

// analyzeFavorites contiene la implementación de AnalyzeFavorites. Debe
// llamarse con el lock de favoritos tomado.
func (c *Controller) analyzeFavorites(all bool, force bool, progress func(string)) error {
	appDir, err := GetAppDir()
	if err != nil {
		return err
	}
	favPath := c.FeedService.getFavoritesPath()
	if favPath == "" {
		return fmt.Errorf("cannot resolve favorites path")
	}
	thumbDir := filepath.Join(appDir, "cache", "thumbs")

	var favorites []FavoriteWallpaper
	if err := c.feedManager.ReadJSON(favPath, &favorites); err != nil {
		return err
	}

	c.log().Info("Analyzing favorites: %d items found", len(favorites))

	type job struct {
		Index  int
		Fav    FavoriteWallpaper
		Delete bool
	}

	jobs := make(chan job, len(favorites))
	results := make(chan job, len(favorites))

	workers := c.getWorkers()
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				fav := j.Fav
				// Process the embedded Wallpaper
				newWp, changed, deleteItem := c.processWallpaperItem(fav.Wallpaper, force, all, thumbDir, progress)

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
		newFavorites := make([]FavoriteWallpaper, 0, len(favorites))
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
func (c *Controller) processWallpaperItem(wp models.Wallpaper, force, all bool, thumbDir string, progress func(string)) (models.Wallpaper, bool, bool) {
	changed := false

	// 0. Robust nil checks for Controller and its components
	if c == nil || c.ColorManager == nil {
		c.log().Error("CRITICAL: Controller or ColorManager is nil in processWallpaperItem for %s.", wp.ID)
		if progress != nil {
			progress(fmt.Sprintf("CRITICAL Error: Controller or ColorManager is nil for %s.", wp.ID))
		}
		return wp, false, true // Mark for deletion or skip
	}
	thumbPath := filepath.Join(thumbDir, utils.SafeFilename(wp.ID)+".jpg")

	var sourceFilePath string // This will hold the path to the wallpaper file we will use for analysis

	// Helper function to delete associated files
	deleteAssociatedFiles := func(wallpaper models.Wallpaper) {
		// Always delete thumbnail
		_ = os.Remove(filepath.Join(thumbDir, utils.SafeFilename(wallpaper.ID)+".jpg"))
		c.log().Info("Deleted thumbnail: %s", filepath.Join(thumbDir, utils.SafeFilename(wallpaper.ID)+".jpg"))

		// Delete main file ONLY if it's in the cache directory.
		// We should NEVER delete files from the user's configured wallpaper directory automatically.
		cacheExpectedPath, _ := c.GetWallpaperLocalPath(wallpaper)
		if sourceFilePath == cacheExpectedPath { // Only delete if it's the cached version
			_ = os.Remove(sourceFilePath)
			c.log().Info("Deleted cached wallpaper file: %s", sourceFilePath)
		}
	}

	// --- Determine the source file path for the wallpaper ---
	// 1. Check if wp.Path is already set and valid
	if wp.Path != "" {
		if _, err := os.Stat(wp.Path); err == nil {
			sourceFilePath = wp.Path
			c.log().Debug("Wallpaper %s: Using existing wp.Path '%s'.", wp.ID, sourceFilePath)
		} else {
			c.log().Debug("Wallpaper %s wp.Path '%s' is invalid or does not exist. Searching for file.", wp.ID, wp.Path)
		}
	}

	// 2. If indexing is enabled, check user's configured wallpapers directory.
	// We check this even if wp.Path was valid, to prioritize the persistent location over cache.
	if c.Config.Paths.IndexWallpapers && c.Config.Paths.Wallpapers != "" {
		if path, found := c.FindInCollection(wp); found {
			if sourceFilePath != path {
				sourceFilePath = path
				wp.Path = sourceFilePath
				changed = true
				c.log().Info("Wallpaper %s: Found in collection folder. Using path: %s", wp.ID, sourceFilePath)
				if progress != nil {
					progress(fmt.Sprintf("Found %s in collection folder.", wp.ID))
				}
			}
		}
	}

	// 3. If still not found, check the cache directory
	if sourceFilePath == "" {
		cachePath, err := c.GetWallpaperLocalPath(wp) // This is the cache path
		if err == nil {
			if _, err := os.Stat(cachePath); err == nil {
				sourceFilePath = cachePath
				wp.Path = sourceFilePath // Update wp.Path to point to the cache
				changed = true
				c.log().Info("Found wallpaper %s in cache: %s", wp.ID, cachePath)
				if progress != nil {
					progress(fmt.Sprintf("Found %s in cache.", wp.ID))
				}
			}
		}
	}

	// 4. If still not found, download it to the cache
	if sourceFilePath == "" {
		if progress != nil {
			progress(fmt.Sprintf("Downloading %s for analysis", wp.ID))
		}
		downloadedPath, err := c.DownloadWallpaper(wp) // This downloads to cache
		if err != nil {
			c.log().Error("Failed to download %s for analysis: %v", wp.ID, err)
			if progress != nil {
				progress(fmt.Sprintf("Error downloading %s for analysis: %v", wp.ID, err))
			}
			return wp, false, true // Mark for deletion if download fails
		}
		sourceFilePath = downloadedPath
		wp.Path = sourceFilePath // Update wp.Path to the newly downloaded file in cache
		changed = true
	}

	// At this point, sourceFilePath is guaranteed to be a local, existing file.
	// Use sourceFilePath for all subsequent operations (thumbnail, color analysis, validation).

	// 2. Verify/Regenerate Thumbnail and Color
	info, errStat := os.Stat(thumbPath)
	thumbExists := errStat == nil && info.Size() > 0

	// Check if existing thumbnail is valid image data (only if not forcing regeneration)
	if thumbExists && !force {
		if _, _, err := c.ColorManager.GetImageDimensions(thumbPath); err != nil {
			thumbExists = false // Treat as missing to force regeneration
			c.log().Info("Thumbnail for %s is corrupt or invalid. Regenerating...", wp.ID)
		}
	}

	if !thumbExists || force { // If thumbnail is missing or we are forcing regeneration
		c.log().Info("Generating thumbnail for %s from %s", wp.ID, sourceFilePath)
		if progress != nil {
			progress(fmt.Sprintf("Generating thumbnail for %s", wp.ID))
		}
		w, h, err := c.ColorManager.GenerateThumbnail(sourceFilePath, thumbPath)
		if err == nil {
			// Check validity immediately after generation
			if valid, reason := c.isValidImage(w, h, false); !valid { // Solo validar aspect_ratio
				c.log().Info("Removing invalid item %s (resolution %dx%d). Reason: %s", wp.ID, w, h, reason)
				if progress != nil {
					progress(fmt.Sprintf("Removing invalid item %s (resolution %dx%d). Reason: %s", wp.ID, w, h, reason))
				}
				deleteAssociatedFiles(wp)
				return wp, false, true
			}

			c.log().Info("Successfully generated thumbnail for %s", wp.ID)
			if progress != nil {
				progress(fmt.Sprintf("Successfully generated thumbnail for %s", wp.ID))
			}
			wp.Extension = ".jpg"
			changed = true
			// If we generated it, we can set ratio and dimension if missing
			if wp.Ratio == "" && w > 0 && h > 0 {
				wp.Ratio = calculateRatio(w, h)
				wp.Dimension = fmt.Sprintf("%dx%d", w, h) // Always set original dimension if available
				changed = true
			}
			// Color analysis will be done in the next step, or if DownloadWallpaper already did it, it's fine.
		} else {
			c.log().Error("Failed to generate thumbnail for %s from %s: %v", wp.ID, sourceFilePath, err)
			if progress != nil {
				progress(fmt.Sprintf("Failed to generate thumbnail for %s: %v", wp.ID, err))
			}
			return wp, false, true
		}
	} else {
		// Thumbnail exists and is valid, but ensure wp.Dimension is set from it if missing
		if wp.Dimension == "" {
			if w, h, err := c.ColorManager.GetImageDimensions(thumbPath); err == nil {
				wp.Dimension = fmt.Sprintf("%dx%d", w, h)
				changed = true
			}
		}
	}

	// 3. Re-evaluate color and theme if needed (e.g., if `all` is true or if they are missing)
	// This part is important because DownloadWallpaper might not update wp.Color/wp.Theme in the `wp` object passed to `processWallpaperItem`.
	// We need to ensure `wp.Color` and `wp.Theme` are correctly set based on the generated thumbnail.
	if all || wp.Color == "" || wp.Theme == "" {
		if color, err := c.ColorManager.AnalyzeColor(thumbPath); err == nil {
			if wp.Color != color || wp.Theme == "" || (c.ColorManager.IsDark(color) && wp.Theme != "dark") || (!c.ColorManager.IsDark(color) && wp.Theme != "light") {
				wp.Color = color
				if c.ColorManager.IsDark(color) {
					wp.Theme = "dark"
				} else {
					wp.Theme = "light"
				}
				changed = true
			}
		} else {
			c.log().Error("Failed to analyze color for %s: %v", wp.ID, err)
			if progress != nil {
				progress(fmt.Sprintf("Failed to analyze color for %s: %v", wp.ID, err))
			}
		}
	}

	// 4. Validate image dimensions and aspect ratio
	c.log().Debug("processWallpaperItem: Validating image %s. wp.Dimension='%s'.", wp.ID, wp.Dimension)
	var validationW, validationH int
	var validationErr error
	var validationSource string

	// Prioritize original dimensions if available and valid
	if wp.Dimension != "" { // Check if original dimension is stored
		w, h, err := utils.ParseResolution(wp.Dimension)
		if err == nil {
			validationW, validationH = w, h
			validationSource = "original resolution"
		} else {
			c.log().Error("Failed to parse original dimension '%s' for %s: %v. Attempting to use thumbnail dimensions for aspect ratio check.", wp.Dimension, wp.ID, err)
			validationW, validationH, validationErr = c.ColorManager.GetImageDimensions(thumbPath)
			validationSource = "thumbnail resolution (fallback from malformed original dimension)"
		}
	} else {
		// If wp.Dimension is empty, try to get dimensions from the thumbnail
		validationW, validationH, validationErr = c.ColorManager.GetImageDimensions(thumbPath) // Use thumbnail dimensions
		validationSource = "thumbnail resolution (wp.Dimension empty)"
	}

	if validationErr == nil {
		if valid, reason := c.isValidImage(validationW, validationH, false); !valid { // Solo validar aspect_ratio
			c.log().Info("Removing invalid item %s (%s %dx%d). Reason: %s", wp.ID, validationSource, validationW, validationH, reason)
			if progress != nil {
				progress(fmt.Sprintf("Removing invalid item %s (%s %dx%d). Reason: %s", wp.ID, validationSource, validationW, validationH, reason))
			}
			deleteAssociatedFiles(wp)
			return wp, false, true
		}
	} else {
		c.log().Error("Failed to get any dimensions for %s (original or thumbnail): %v. Marking as invalid.", wp.ID, validationErr)
		if progress != nil {
			progress(fmt.Sprintf("Removing invalid item %s (could not determine dimensions for aspect ratio check).", wp.ID))
		}
		deleteAssociatedFiles(wp)
		return wp, false, true
	}

	// Ensure extension is set if it's missing
	if wp.Extension == "" {
		wp.Extension = ".jpg"
		changed = true
	}

	// 5. Rename local file with [d] or [l] tag if enabled (for existing items)
	// This logic is currently inside the `if !thumbExists || force` block and also in the `else` block.
	// This applies to files in the user's collection folder.

	userWallpaperPath, _ := c.GetUserWallpaperPath(wp)
	if wp.Source == "local" && c.Config.Paths.IndexWallpapers && wp.Theme != "" && sourceFilePath == userWallpaperPath {
		localPath := wp.URL
		dir := filepath.Dir(localPath)
		filename := filepath.Base(localPath)
		ext := filepath.Ext(filename)
		nameWithoutExt := strings.TrimSuffix(filename, ext)

		cleanName := nameWithoutExt
		// Remove existing tags from end of filename
		if strings.HasSuffix(cleanName, " [d]") {
			cleanName = strings.TrimSuffix(cleanName, " [d]")
		} else if strings.HasSuffix(cleanName, " [l]") {
			cleanName = strings.TrimSuffix(cleanName, " [l]")
		}

		newTag := "[l]"
		if wp.Theme == "dark" {
			newTag = "[d]"
		}

		var newFilename string
		if cleanName == "" {
			newFilename = fmt.Sprintf("%s%s", newTag, ext)
		} else {
			newFilename = fmt.Sprintf("%s %s%s", cleanName, newTag, ext)
		}

		newPath := filepath.Join(dir, newFilename)
		if localPath != newPath {
			if err := os.Rename(localPath, newPath); err == nil {
				c.log().Info("Renaming local file: %s -> %s", filename, newFilename)
				wp.URL = newPath
				wp.Path = newPath      // Update wp.Path to the new renamed path
				wp.Thumbnail = newPath // Update thumbnail path if it was pointing to the old URL
				changed = true
			} else {
				c.log().Error("Failed to rename local file %s to %s: %v", localPath, newPath, err)
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

	// Use crypto/rand for better distribution and to avoid seeding issues in rapid calls
	n, err := crand.Int(crand.Reader, big.NewInt(int64(len(filteredFeed))))
	if err != nil { //nolint:gosec // G404: Cryptographically secure random is preferred, but math/rand fallback is acceptable for wallpaper selection.
		return filteredFeed[rand.Intn(len(filteredFeed))], nil
	}

	return filteredFeed[n.Int64()], nil
}

// GetFeedStats calculates and returns statistics about the feed.
func (c *Controller) GetFeedStats() (models.FeedStats, error) {
	var stats models.FeedStats
	err := c.withFeedLock(func() error {
		feed, err := c.loadFeed()
		if err != nil {
			return err
		}

		stats.Total = len(feed)
		for _, wp := range feed {
			switch strings.ToLower(wp.Theme) {
			case "dark":
				stats.DarkCount++
			case "light":
				stats.LightCount++
			}
			if wp.Added > 0 {
				added := time.Unix(wp.Added, 0)
				if added.After(stats.LastAdded) {
					stats.LastAdded = added
				}
			}
		}

		if c.FeedService != nil {
			if favorites, err := c.FeedService.loadFavorites(); err == nil {
				stats.FavoritesCount = len(favorites)
			}
		}
		return nil
	})
	return stats, err
}

// AddWallpaperToFeed adds a wallpaper to the feed.
func (c *Controller) AddWallpaperToFeed(wallpaper models.Wallpaper) error {
	return c.withFeedLock(func() error {
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

		for i, existingWp := range feed {
			if existingWp.ID == wallpaper.ID {
				if wallpaper.Added == 0 {
					wallpaper.Added = existingWp.Added
				}
				feed[i] = wallpaper
				return c.saveFeed(feed)
			}
		}

		if wallpaper.Added == 0 {
			wallpaper.Added = time.Now().Unix()
		}
		feed = append(feed, wallpaper)
		if err := c.saveFeed(feed); err != nil {
			return err
		}
		c.log().Info("Added wallpaper %s to feed", wallpaper.ID)
		return nil
	})
}

// AddWallpapersToFeed adds multiple wallpapers to the feed efficiently.
func (c *Controller) AddWallpapersToFeed(wallpapers []models.Wallpaper) (int, error) {
	addedCount := 0
	err := c.withFeedLock(func() error {
		feed, err := c.loadFeed()
		if err != nil {
			return err
		}

		blacklist, err := c.loadBlacklist()
		if err != nil {
			return err
		}
		blacklistMap := make(map[string]bool)
		for _, id := range blacklist {
			blacklistMap[id] = true
		}

		existing := make(map[string]bool)
		for _, wp := range feed {
			existing[wp.ID] = true
		}

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
			return c.saveFeed(feed)
		}
		return nil
	})
	return addedCount, err
}

// AddToBlacklist adds an ID to the blacklist.
func (c *Controller) AddToBlacklist(id string) error {
	err := c.withBlacklistLock(func() error {
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
	})
	if err != nil {
		return err
	}
	c.log().Info("Added wallpaper %s to blacklist", id)

	return c.RebuildColorIndex()
}

// RemoveFromBlacklist removes an ID from the blacklist.
func (c *Controller) RemoveFromBlacklist(id string) error {
	err := c.withBlacklistLock(func() error {
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
		return c.feedManager.WriteJSON(path, newBlacklist)
	})
	if err != nil {
		return err
	}
	c.log().Info("Removed wallpaper %s from blacklist", id)

	return c.RebuildColorIndex()
}

// GetBlacklist returns the current blacklist.
func (c *Controller) GetBlacklist() ([]string, error) {
	return c.loadBlacklist()
}

// RemoveFromFeed removes a wallpaper from the feed by ID.
func (c *Controller) RemoveFromFeed(id string) error {
	return c.withFeedLock(func() error {
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
		c.log().Info("Removed wallpaper %s from feed", id)
		return nil
	})
}

// DeleteWallpaper removes a wallpaper from the feed and optionally deletes the file from disk.
func (c *Controller) DeleteWallpaper(id string, deleteFile bool) error {
	wp, err := c.GetWallpaper(id) // This finds it in feed OR favorites
	if err != nil {
		return err
	}

	if err := c.RemoveFromFeed(id); err != nil {
		return err
	}

	if err := c.RemoveFromFavorites(id); err != nil {
		c.log().Error("Failed to remove wallpaper %s from favorites: %v", id, err)
	}

	if deleteFile {
		if wp.Source == "local" {
			// Un feed.json manipulado podría apuntar a cualquier fichero del
			// usuario. Solo se borra si está dentro del directorio configurado.
			if !utils.IsWithinDir(wp.URL, c.Config.Paths.Wallpapers) {
				return fmt.Errorf("refusing to delete %q: outside configured wallpapers directory", wp.URL)
			}
			if err := os.Remove(wp.URL); err != nil {
				return fmt.Errorf("failed to delete local file: %w", err)
			}
			c.log().Info("Deleted local file: %s", wp.URL)
		} else {
			if path, found := c.FindWallpaperCacheFile(*wp); found {
				_ = os.Remove(path)
			}
		}

		if appDir, err := GetAppDir(); err == nil {
			thumbPath := filepath.Join(appDir, "cache", "thumbs", utils.SafeFilename(wp.ID)+".jpg")
			_ = os.Remove(thumbPath)
		}
	}

	return c.RebuildColorIndex()
}

// RemoveFromFavorites removes a wallpaper from the favorites list by ID.
func (c *Controller) RemoveFromFavorites(id string) error {
	return c.withFavoritesLock(func() error {
		favPath := c.FeedService.getFavoritesPath()
		if favPath == "" {
			return fmt.Errorf("cannot resolve favorites path")
		}

		var favorites []FavoriteWallpaper
		if err := c.feedManager.ReadJSON(favPath, &favorites); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		newFavorites := make([]FavoriteWallpaper, 0, len(favorites))
		found := false
		for _, fav := range favorites {
			if fav.ID == id {
				found = true
				continue
			}
			newFavorites = append(newFavorites, fav)
		}

		if !found {
			return nil
		}

		c.log().Info("Removed wallpaper %s from favorites", id)
		return c.feedManager.WriteJSON(favPath, newFavorites)
	})
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
		for i := range feed {
			wp := feed[i]
			if wp.ID == id {
				return &wp, nil
			}
		}
	}

	// 2. Check Favorites
	if c.FeedService == nil {
		return nil, fmt.Errorf("wallpaper with ID %s not found", id)
	}
	favPath := c.FeedService.getFavoritesPath()
	if favPath == "" {
		return nil, fmt.Errorf("wallpaper with ID %s not found", id)
	}
	var favorites []FavoriteWallpaper
	if err := c.feedManager.ReadJSON(favPath, &favorites); err == nil {
		for i := range favorites {
			fav := favorites[i]
			if fav.ID == id {
				return &fav.Wallpaper, nil
			}
		}
	}

	return nil, fmt.Errorf("wallpaper with ID %s not found", id)
}

// GetWallpaperLocalPath returns the expected local path for a wallpaper without downloading it.
func (c *Controller) GetWallpaperLocalPath(wp models.Wallpaper) (string, error) {
	return c.DownloadService.GetWallpaperLocalPath(wp)
}

// GetUserWallpaperPath returns the expected path for a wallpaper in the user's configured wallpapers directory.
func (c *Controller) GetUserWallpaperPath(wp models.Wallpaper) (string, error) {
	return c.DownloadService.GetUserWallpaperPath(wp)
}

// FindInCollection attempts to find the wallpaper file in the user's configured collection directory.
func (c *Controller) FindInCollection(wp models.Wallpaper) (string, bool) {
	return c.DownloadService.FindInCollection(wp)
}

// DownloadWallpaper downloads the wallpaper image to the cache directory and returns the local path.
func (c *Controller) DownloadWallpaper(wp models.Wallpaper) (string, error) {
	return c.DownloadService.DownloadWallpaper(wp)
}

// GetCachedWallpapers retrieves wallpapers from feed (and optionally favorites) that are locally cached.
func (c *Controller) GetCachedWallpapers(includeFavorites bool, theme string) ([]models.Wallpaper, error) {
	return c.DownloadService.GetCachedWallpapers(includeFavorites, theme)
}

// FindWallpaperCacheFile finds the local cache file for a wallpaper, even if it has a bad name.
func (c *Controller) FindWallpaperCacheFile(wp models.Wallpaper) (string, bool) {
	return c.DownloadService.FindWallpaperCacheFile(wp)
}

// ParserSearch represents a search session stored in the parser cache.
type ParserSearch struct {
	Date    time.Time          `json:"date"`
	Query   string             `json:"query"`
	Results []models.Wallpaper `json:"results"`
}

// SaveParserSearch saves the search results to the provider's parser cache file.
// It enforces a 14-day retention policy for old searches.
func (c *Controller) SaveParserSearch(providerName, query string, results []models.Wallpaper) error {
	if c.FeedService == nil {
		return fmt.Errorf("feed service not initialized")
	}
	return c.FeedService.SaveParserSearch(providerName, query, results)
}

// SyncFeed processes parser cache files and populates the feed.
func (c *Controller) SyncFeed() (int, int, error) {
	var added, repaired int
	err := c.withFeedLock(func() error {
		var err error
		added, repaired, err = c.syncFeed()
		return err
	})
	return added, repaired, err
}

// syncFeed contiene la implementación de SyncFeed. Debe llamarse con el lock
// del feed tomado.
func (c *Controller) syncFeed() (int, int, error) {
	c.log().Info("Starting feed sync...")

	appDir, err := GetAppDir()
	if err != nil {
		return 0, 0, err
	}
	parserDir := filepath.Join(appDir, "data", "parser")

	// Un directorio de parser inexistente no es un error: el indexado local
	// puede seguir aportando wallpapers.
	files, err := os.ReadDir(parserDir)
	if err != nil && !os.IsNotExist(err) {
		return 0, 0, err
	}

	// Load existing feed and blacklist to avoid duplicates. No se ignoran los
	// errores: sobrescribir un feed ilegible perdería datos.
	feed, err := c.loadFeed()
	if err != nil {
		return 0, 0, err
	}
	blacklist, err := c.loadBlacklist()
	if err != nil {
		return 0, 0, err
	}

	// Index local wallpapers if enabled
	addedLocal := 0
	removedLocal := 0
	var newLocalWallpapers []models.Wallpaper

	if c.Config.Paths.IndexWallpapers {
		c.log().Debug("Local wallpaper indexing is ENABLED (Path: %s)", c.Config.Paths.Wallpapers)
	} else {
		c.log().Debug("Local wallpaper indexing is DISABLED")
	}

	if c.Config.Paths.IndexWallpapers && c.Config.Paths.Wallpapers != "" {
		prevLen := len(feed)
		var err error
		addedLocal, _, removedLocal, err = c.indexLocalWallpapers(&feed)
		if err != nil {
			c.log().Error("Error indexing local wallpapers: %v", err)
		}
		if addedLocal > 0 {
			newLocalWallpapers = feed[prevLen:]
		}
	}

	if removedLocal > 0 {
		newFeed := make([]models.Wallpaper, 0, len(feed))
		for _, wp := range feed {
			if wp.ID != "" {
				newFeed = append(newFeed, wp)
			}
		}
		feed = newFeed
	}

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
	thumbDir := filepath.Join(appDir, "cache", "thumbs")

	// 1. Recolectar candidatos únicos
	var candidates []models.Wallpaper

	// Añadir nuevos wallpapers locales para análisis inmediato (miniaturas/color)
	for _, wp := range newLocalWallpapers {
		candidates = append(candidates, wp)
		processed[wp.ID] = true
	}

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
					thumbPath := filepath.Join(thumbDir, utils.SafeFilename(wp.ID)+".jpg")
					if info, err := os.Stat(thumbPath); err == nil && info.Size() > 0 {
						// Check if valid image
						if _, _, err := c.ColorManager.GetImageDimensions(thumbPath); err == nil {
							// Exists and valid, skip
							processed[wp.ID] = true
							continue
						}
					}
					// Thumbnail missing, add to candidates to regenerate
				}

				candidates = append(candidates, wp)
				processed[wp.ID] = true
			}
		}
	}

	if len(candidates) == 0 {
		// Aunque no haya parser files, el indexado local puede haber añadido o
		// quitado entradas: hay que persistirlas en vez de descartarlas.
		if addedLocal > 0 || removedLocal > 0 {
			if err := c.rebuildColorsIndex(feed); err != nil {
				c.log().Error("Failed to rebuild color index: %v", err)
			}
			if err := c.saveFeed(feed); err != nil {
				return 0, 0, err
			}
			c.log().Info("Feed sync completed. Local changes: +%d/-%d", addedLocal, removedLocal)
			return addedLocal, 0, nil
		}
		return 0, 0, nil
	}

	// 2. Procesar concurrentemente (Worker Pool)
	workers := c.getWorkers()
	jobs := make(chan models.Wallpaper, len(candidates))
	results := make(chan models.Wallpaper, len(candidates))
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for wp := range jobs {
				// Filtrar por metadatos antes de intentar descargar nada

				thumbPath := filepath.Join(thumbDir, utils.SafeFilename(wp.ID)+".jpg")

				// Usar thumbnail URL si existe para ahorrar ancho de banda
				src := wp.Thumbnail
				if src == "" || src == wp.URL {
					src = wp.URL
				}

				// Generar thumbnail y analizar color
				width, height, err := c.ColorManager.GenerateThumbnail(src, thumbPath)
				if err == nil {
					// Validar Ratio antes de continuar
					// When creating thumbnails, we only validate aspect ratio, not absolute resolution.
					if valid, reason := c.isValidImage(width, height, false); !valid { // Solo validar aspect_ratio
						c.log().Info("Rejected item %s: dimensions %dx%d do not match aspect ratio criteria. Reason: %s. Removing thumbnail.", wp.ID, width, height, reason)
						_ = os.Remove(thumbPath) // Limpiar thumbnail generado
						continue
					}

					wp.Extension = ".jpg"

					// Calcular Ratio si falta
					if wp.Ratio == "" && width > 0 && height > 0 {
						wp.Ratio = calculateRatio(width, height)
					}

					// Populate wp.Dimension with original dimensions if not already set by provider
					if wp.Dimension == "" && width > 0 && height > 0 {
						wp.Dimension = fmt.Sprintf("%dx%d", width, height)
					}

					if color, err := c.ColorManager.AnalyzeColor(thumbPath); err == nil {
						wp.Color = color
						if c.ColorManager.IsDark(color) {
							wp.Theme = "dark"
						} else {
							wp.Theme = "light"
						}
					}

					// Auto-download full image if enabled
					if c.Config.Behavior.AutoDownload {
						if path, err := c.DownloadWallpaper(wp); err == nil {
							wp.Path = path
						}
					}

					// Marcar como no visto
					wp.Seen = false

					results <- wp
				} else {
					c.log().Info("Failed to download/generate thumbnail for %s: %v", wp.ID, err)
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

	// Recolectar resultados. Un índice por ID evita el O(n²) de recorrer el
	// feed completo para cada wallpaper procesado.
	feedIndex := make(map[string]int, len(feed))
	for i := range feed {
		feedIndex[feed[i].ID] = i
	}

	for wp := range results {
		if idx, ok := feedIndex[wp.ID]; ok {
			wp.Seen = feed[idx].Seen
			wp.Added = feed[idx].Added
			feed[idx] = wp
			repairedCount++
		} else {
			wp.Added = time.Now().Unix()
			feedIndex[wp.ID] = len(feed)
			feed = append(feed, wp)
			addedCount++
		}
	}

	// Reconstruir colors.json basado en el feed actualizado
	if err := c.rebuildColorsIndex(feed); err != nil {
		c.log().Error("Failed to rebuild color index: %v", err)
	}

	if addedCount > 0 || repairedCount > 0 || addedLocal > 0 || removedLocal > 0 {
		feed = c.applyHardLimit(feed)
		err := c.saveFeed(feed)
		if err == nil {
			c.log().Info("Feed sync completed. Added: %d (Local: %d), Repaired: %d", addedCount+addedLocal, addedLocal, repairedCount)
		}
		return addedCount + addedLocal, repairedCount, err
	}
	c.log().Info("Feed sync completed. No changes.")
	return 0, 0, nil
}

// applyHardLimit recorta el feed por antigüedad (no por posición en el slice) y
// elimina los ficheros cacheados de las entradas descartadas.
func (c *Controller) applyHardLimit(feed []models.Wallpaper) []models.Wallpaper {
	limit := 0
	if c.Config != nil {
		limit = c.Config.Limits.FeedHardLimit
	}
	if limit <= 0 || len(feed) <= limit {
		return feed
	}

	sort.SliceStable(feed, func(i, j int) bool {
		if feed[i].Added == feed[j].Added {
			return feed[i].ID < feed[j].ID
		}
		return feed[i].Added < feed[j].Added
	})

	removed := len(feed) - limit
	for i := 0; i < removed; i++ {
		c.removeCachedFiles(feed[i])
	}
	return feed[removed:]
}

// removeCachedFiles borra el fichero cacheado y el thumbnail de un wallpaper,
// asegurándose de no tocar ficheros fuera del directorio de caché.
func (c *Controller) removeCachedFiles(wp models.Wallpaper) {
	appDir, err := GetAppDir()
	if err != nil {
		return
	}
	cacheDir := filepath.Join(appDir, "cache", "wallpapers")
	if path, found := c.FindWallpaperCacheFile(wp); found && utils.IsWithinDir(path, cacheDir) {
		_ = os.Remove(path)
	}
	_ = os.Remove(filepath.Join(appDir, "cache", "thumbs", utils.SafeFilename(wp.ID)+".jpg"))
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
	return c.withFeedLock(func() error {
		feed, err := c.loadFeed()
		if err != nil {
			return err
		}

		blacklist, err := c.loadBlacklist()
		if err != nil {
			return err
		}
		blacklistMap := make(map[string]bool)
		for _, id := range blacklist {
			blacklistMap[id] = true
		}

		activeFeed := make([]models.Wallpaper, 0, len(feed))
		for _, wp := range feed {
			if !blacklistMap[wp.ID] {
				activeFeed = append(activeFeed, wp)
			}
		}
		return c.rebuildColorsIndex(activeFeed)
	})
}

func (c *Controller) rebuildColorsIndex(feed []models.Wallpaper) error {
	return c.AnalysisService.rebuildColorsIndex(feed)
}

// LoadColorPalettes loads the generated palettes from colors.json
func (c *Controller) LoadColorPalettes() ([]string, []string, error) {
	return c.AnalysisService.LoadColorPalettes()
}

func (c *Controller) isValidImage(width, height int, checkResolution bool) (bool, string) {
	return c.AnalysisService.isValidImage(width, height, checkResolution)
}

// GetLastProviderUpdateTime returns the modification time of the most recently updated provider cache file.
func (c *Controller) GetLastProviderUpdateTime() (time.Time, error) {
	appDir, err := GetAppDir()
	if err != nil {
		return time.Time{}, err
	}
	parserDir := filepath.Join(appDir, "data", "parser")

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

// UpdateWallpaperPath updates the local path for a wallpaper in both Feed and Favorites if they exist.
func (c *Controller) UpdateWallpaperPath(id, path string) error {
	// 1. Update Feed
	if err := c.withFeedLock(func() error {
		feed, err := c.loadFeed()
		if err != nil {
			return err
		}
		changed := false
		for i := range feed {
			if feed[i].ID == id {
				if feed[i].Path != path {
					feed[i].Path = path
					changed = true
				}
				break
			}
		}
		if changed {
			return c.saveFeed(feed)
		}
		return nil
	}); err != nil {
		c.log().Error("Failed to save feed after updating wallpaper path: %v", err)
	}

	// 2. Update Favorites
	if c.FeedService == nil {
		return nil
	}
	favPath := c.FeedService.getFavoritesPath()
	if favPath == "" {
		return nil
	}
	return c.withFavoritesLock(func() error {
		var favorites []FavoriteWallpaper
		if err := c.feedManager.ReadJSON(favPath, &favorites); err != nil {
			return nil
		}
		changed := false
		for i := range favorites {
			if favorites[i].ID == id {
				if favorites[i].Path != path {
					favorites[i].Path = path
					changed = true
				}
				break
			}
		}
		if changed {
			return c.feedManager.WriteJSON(favPath, favorites)
		}
		return nil
	})
}
