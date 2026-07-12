package core

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gower/internal/utils"
	"gower/pkg/models"
)

type AnalysisService struct {
	Config       *models.Config
	ColorManager *ColorManager
	feedManager  *utils.SecureJSONManager
	Log          *utils.Logger
}

func (s *AnalysisService) log() *utils.Logger {
	if s.Log != nil {
		return s.Log
	}
	return utils.Log
}

func (s *AnalysisService) getWorkers() int {
	if s.Config != nil && s.Config.Limits.AnalysisWorkers > 0 {
		return s.Config.Limits.AnalysisWorkers
	}
	return 5
}

func NewAnalysisService(cfg *models.Config, cm *ColorManager, fm *utils.SecureJSONManager) *AnalysisService {
	return &AnalysisService{
		Config:       cfg,
		ColorManager: cm,
		feedManager:  fm,
	}
}

func (s *AnalysisService) AnalyzeFeed(feed []models.Wallpaper, f func(string)) []models.Wallpaper {
	appDir, err := GetAppDir()
	if err != nil {
		return feed
	}
	thumbDir := filepath.Join(appDir, "cache", "thumbs")

	type job struct {
		Index  int
		Wp     models.Wallpaper
		Delete bool
	}

	jobs := make(chan job, len(feed))
	results := make(chan job, len(feed))

	workers := s.getWorkers()
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				newWp, changed, deleteItem := s.processWallpaperItem(j.Wp, thumbDir, f)
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

	for res := range results {
		if res.Delete {
			feed[res.Index].ID = ""
		} else {
			feed[res.Index] = res.Wp
		}
	}

	var newFeed []models.Wallpaper
	for _, wp := range feed {
		if wp.ID != "" {
			newFeed = append(newFeed, wp)
		}
	}
	return newFeed
}

func (s *AnalysisService) processWallpaperItem(wp models.Wallpaper, thumbDir string, progress func(string)) (models.Wallpaper, bool, bool) {
	if progress != nil {
		progress(fmt.Sprintf("Analyzing: %s", wp.ID))
	}

	appDir, err := GetAppDir()
	if err != nil {
		return wp, false, false
	}
	cacheDir := filepath.Join(appDir, "cache", "wallpapers")

	imagePath := wp.Path
	if imagePath == "" {
		safeID := strings.ReplaceAll(wp.ID, "/", "_")
		urlStr := wp.URL
		if qIndex := strings.Index(urlStr, "?"); qIndex != -1 {
			urlStr = urlStr[:qIndex]
		}
		ext := filepath.Ext(urlStr)
		if ext == "" {
			ext = ".jpg"
		}
		expectedPath := filepath.Join(cacheDir, safeID+ext)

		matches, err := filepath.Glob(filepath.Join(cacheDir, safeID+".*"))
		if err == nil && len(matches) > 0 {
			imagePath = matches[0]
		} else {
			imagePath = expectedPath
		}
	}

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return wp, false, true
	}

	thumbPath := filepath.Join(thumbDir, wp.ID+".jpg")
	if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
		if _, _, err := s.ColorManager.GenerateThumbnail(imagePath, thumbPath); err != nil {
			s.log().Error("Thumbnail generation failed for %s: %v", wp.ID, err)
			return wp, false, true
		}
	}

	changed := false

	newColor, err := s.ColorManager.AnalyzeColor(imagePath)
	if err != nil {
		s.log().Debug("Color analysis failed for %s: %v", wp.ID, err)
	} else if newColor != wp.Color {
		wp.Color = newColor
		changed = true
	}

	newTheme := ""
	if s.ColorManager.IsDark(wp.Color) {
		newTheme = "dark"
	} else {
		newTheme = "light"
	}
	if newTheme != wp.Theme {
		wp.Theme = newTheme
		changed = true
	}

	width, height, err := s.ColorManager.GetImageDimensions(imagePath)
	if err == nil {
		dim := fmt.Sprintf("%dx%d", width, height)
		if dim != wp.Dimension {
			wp.Dimension = dim
			changed = true
		}
		if s.Config != nil {
			valid, _ := s.isValidImage(width, height, true)
			if !valid {
				if progress != nil {
					progress(fmt.Sprintf("Removing (invalid dimensions): %s (%dx%d)", wp.ID, width, height))
				}
				return wp, false, true
			}
		}

		ratio := ""
		if width > 0 && height > 0 {
			r := float64(width) / float64(height)
			if r > 1.7 {
				ratio = "ultrawide"
			} else if r > 1.5 {
				ratio = "wide"
			} else if r > 1.3 {
				ratio = "standard"
			} else {
				ratio = "portrait"
			}
		}
		if ratio != wp.Ratio {
			wp.Ratio = ratio
			changed = true
		}
	}

	if progress != nil {
		progress(fmt.Sprintf("Analyzed: %s", wp.ID))
	}

	return wp, changed, false
}

func (s *AnalysisService) AnalyzeFavorites(favorites []FavoriteWallpaper, thumbDir string, progress func(string)) []FavoriteWallpaper {
	type job struct {
		Index  int
		Wp     models.Wallpaper
		Delete bool
	}

	jobs := make(chan job, len(favorites))
	results := make(chan job, len(favorites))

	workers := s.getWorkers()
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				newWp, changed, deleteItem := s.processWallpaperItem(j.Wp, thumbDir, progress)
				if deleteItem {
					results <- job{Index: j.Index, Delete: true}
				} else if changed {
					results <- job{Index: j.Index, Wp: newWp}
				}
			}
		}()
	}

	for i, f := range favorites {
		jobs <- job{Index: i, Wp: f.Wallpaper}
	}
	close(jobs)
	wg.Wait()
	close(results)

	for res := range results {
		if res.Delete {
			favorites[res.Index].ID = ""
		} else {
			favorites[res.Index].Wallpaper = res.Wp
		}
	}

	var newFavorites []FavoriteWallpaper
	for _, f := range favorites {
		if f.ID != "" {
			newFavorites = append(newFavorites, f)
		}
	}
	return newFavorites
}

func (s *AnalysisService) rebuildColorsIndex(feed []models.Wallpaper) error {
	appDir, err := GetAppDir()
	if err != nil {
		return err
	}
	path := filepath.Join(appDir, "data", "colors.json")

	var feedColors []string
	for _, wp := range feed {
		if wp.Color != "" {
			feedColors = append(feedColors, wp.Color)
		}
	}

	favPath := filepath.Join(appDir, "data", "favorites.json")
	var favorites []struct {
		models.Wallpaper
		Notes string `json:"notes,omitempty"`
	}
	var favColors []string
	if err := s.feedManager.ReadJSON(favPath, &favorites); err == nil {
		for _, fav := range favorites {
			if fav.Color != "" {
				favColors = append(favColors, fav.Color)
			}
		}
	}

	feedPalette := s.ColorManager.GenerateDynamicPalette(feedColors, 16)
	favPalette := s.ColorManager.GenerateDynamicPalette(favColors, 16)

	output := struct {
		FeedPalette      []string `json:"feed_palette"`
		FavoritesPalette []string `json:"favorites_palette"`
	}{
		FeedPalette:      feedPalette,
		FavoritesPalette: favPalette,
	}
	return s.feedManager.WriteJSON(path, output)
}

func (s *AnalysisService) LoadColorPalettes() ([]string, []string, error) {
	appDir, err := GetAppDir()
	if err != nil {
		return nil, nil, err
	}
	path := filepath.Join(appDir, "data", "colors.json")

	var data struct {
		FeedPalette      []string `json:"feed_palette"`
		FavoritesPalette []string `json:"favorites_palette"`
	}
	if err := s.feedManager.ReadJSON(path, &data); err != nil {
		return nil, nil, err
	}
	return data.FeedPalette, data.FavoritesPalette, nil
}

func (s *AnalysisService) isValidImage(width, height int, checkResolution bool) (bool, string) {
	if s.Config == nil {
		return true, ""
	}
	if checkResolution && s.Config.Search.MinWidth > 0 && width < s.Config.Search.MinWidth {
		return false, fmt.Sprintf("width %d is less than min_width %d", width, s.Config.Search.MinWidth)
	}
	if checkResolution && s.Config.Search.MinHeight > 0 && height < s.Config.Search.MinHeight {
		return false, fmt.Sprintf("height %d is less than min_height %d", height, s.Config.Search.MinHeight)
	}
	if height == 0 {
		return false, "height is zero, cannot calculate aspect ratio"
	}
	if s.Config.Search.AspectRatio == "" {
		return true, ""
	}

	target := s.Config.Search.AspectRatio
	var targetRatio float64

	if strings.Contains(target, ":") {
		parts := strings.Split(target, ":")
		if len(parts) == 2 {
			w, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			h, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil && h != 0 {
				targetRatio = w / h
			} else {
				return true, "malformed aspect ratio config"
			}
		} else {
			return true, "malformed aspect ratio config"
		}
	} else {
		parsed, err := strconv.ParseFloat(target, 64)
		if err != nil {
			return true, "malformed aspect ratio config"
		}
		targetRatio = parsed
	}

	if targetRatio > 0 {
		actual := float64(width) / float64(height)
		tolerance := s.Config.Search.Tolerance
		if tolerance <= 0 {
			tolerance = 0.05
		}
		if actual < targetRatio-tolerance || actual > targetRatio+tolerance {
			return false, fmt.Sprintf("aspect ratio mismatch: %.2f (expected ~%.2f ±%.2f)", actual, targetRatio, tolerance)
		}
	}

	return true, ""
}
