package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gower/internal/utils"
	"gower/pkg/models"
)

type DownloadService struct {
	Config       *models.Config
	ColorManager *ColorManager
	feedManager  *utils.SecureJSONManager
	Log          *utils.Logger
}

func (s *DownloadService) log() *utils.Logger {
	if s.Log != nil {
		return s.Log
	}
	return utils.Log
}

func NewDownloadService(cfg *models.Config, cm *ColorManager, fm *utils.SecureJSONManager) *DownloadService {
	return &DownloadService{
		Config:       cfg,
		ColorManager: cm,
		feedManager:  fm,
	}
}

func (s *DownloadService) getCacheDir() (string, error) {
	appDir, err := GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "cache", "wallpapers"), nil
}

func (s *DownloadService) GetWallpaperLocalPath(wp models.Wallpaper) (string, error) {
	cacheDir, err := s.getCacheDir()
	if err != nil {
		return "", err
	}

	urlStr := wp.URL
	if qIndex := strings.Index(urlStr, "?"); qIndex != -1 {
		urlStr = urlStr[:qIndex]
	}
	ext := filepath.Ext(urlStr)
	if ext == "" {
		ext = ".jpg"
	}

	safeID := strings.ReplaceAll(wp.ID, "/", "_")
	filename := fmt.Sprintf("%s%s", safeID, ext)
	return filepath.Join(cacheDir, filename), nil
}

func (s *DownloadService) GetUserWallpaperPath(wp models.Wallpaper) (string, error) {
	if s.Config.Paths.Wallpapers == "" {
		return "", fmt.Errorf("user wallpapers directory not configured")
	}

	urlStr := wp.URL
	if qIndex := strings.Index(urlStr, "?"); qIndex != -1 {
		urlStr = urlStr[:qIndex]
	}
	ext := filepath.Ext(urlStr)
	if ext == "" {
		ext = ".jpg"
	}

	safeID := strings.ReplaceAll(wp.ID, "/", "_")
	filename := fmt.Sprintf("%s%s", safeID, ext)
	return filepath.Join(s.Config.Paths.Wallpapers, filename), nil
}

func (s *DownloadService) FindInCollection(wp models.Wallpaper) (string, bool) {
	if s.Config.Paths.Wallpapers == "" {
		return "", false
	}

	if path, err := s.GetUserWallpaperPath(wp); err == nil {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	safeID := strings.ReplaceAll(wp.ID, "/", "_")

	urlStr := wp.URL
	if qIndex := strings.Index(urlStr, "?"); qIndex != -1 {
		urlStr = urlStr[:qIndex]
	}
	ext := filepath.Ext(urlStr)
	if ext == "" {
		ext = ".jpg"
	}

	for _, tag := range []string{" [d]", " [l]"} {
		path := filepath.Join(s.Config.Paths.Wallpapers, safeID+tag+ext)
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	return "", false
}

func (s *DownloadService) DownloadWallpaper(wp models.Wallpaper) (string, error) {
	filePath, err := s.GetWallpaperLocalPath(wp)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}

	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}

	if !strings.HasPrefix(wp.URL, "http://") && !strings.HasPrefix(wp.URL, "https://") {
		srcPath := wp.URL
		if decoded, err := url.QueryUnescape(srcPath); err == nil {
			srcPath = decoded
		}

		if _, err := os.Stat(srcPath); err == nil {
			input, err := os.ReadFile(srcPath)
			if err != nil {
				return "", err
			}
			if err := os.WriteFile(filePath, input, 0644); err != nil {
				return "", err
			}
		} else {
			return "", fmt.Errorf("unsupported protocol or missing local file: %s", wp.URL)
		}
	} else {
		resp, err := http.Get(wp.URL)
		if err != nil {
			return "", err
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to download wallpaper: status %d", resp.StatusCode)
		}

		out, err := os.Create(filePath)
		if err != nil {
			return "", err
		}
		defer func() { _ = out.Close() }()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return "", err
		}
	}

	thumbDir := filepath.Join(filepath.Dir(filepath.Dir(filePath)), "thumbs")
	thumbPath := filepath.Join(thumbDir, wp.ID+".jpg")
	if _, _, err := s.ColorManager.GenerateThumbnail(filePath, thumbPath); err != nil {
		s.log().Error("Failed to generate thumbnail for %s: %v", wp.ID, err)
	}

	_, err = s.ColorManager.AnalyzeColor(filePath)
	if err != nil {
		s.log().Debug("Color analysis failed/skipped for %s: %v", wp.ID, err)
	}

	s.log().Info("Downloaded wallpaper %s to %s", wp.ID, filePath)
	return filePath, nil
}

func (s *DownloadService) FindWallpaperCacheFile(wp models.Wallpaper) (string, bool) {
	appDir, err := GetAppDir()
	if err != nil {
		return "", false
	}
	wallpaperCacheDir := filepath.Join(appDir, "cache", "wallpapers")
	safeID := strings.ReplaceAll(wp.ID, "/", "_")

	matches, err := filepath.Glob(filepath.Join(wallpaperCacheDir, safeID+"*"))
	if err != nil || len(matches) == 0 {
		if _, err := os.Stat(wp.URL); err == nil {
			return wp.URL, true
		}
		return "", false
	}

	return matches[0], true
}

func (s *DownloadService) GetCachedWallpapers(includeFavorites bool, theme string) ([]models.Wallpaper, error) {
	var candidates []models.Wallpaper

	appDir, err := GetAppDir()
	if err != nil {
		return nil, err
	}

	feedPath := filepath.Join(appDir, "data", "feed.json")
	var feed []models.Wallpaper
	if err := s.feedManager.ReadJSON(feedPath, &feed); err == nil {
		candidates = append(candidates, feed...)
	}

	if includeFavorites {
		favPath := filepath.Join(appDir, "data", "favorites.json")
		var favorites []struct {
			models.Wallpaper
			Notes string `json:"notes,omitempty"`
		}
		if err := s.feedManager.ReadJSON(favPath, &favorites); err == nil {
			for _, f := range favorites {
				candidates = append(candidates, f.Wallpaper)
			}
		}
	}

	var result []models.Wallpaper
	seen := make(map[string]bool)

	for _, wp := range candidates {
		if seen[wp.ID] {
			continue
		}
		seen[wp.ID] = true

		if theme != "" && theme != "auto" && !strings.EqualFold(wp.Theme, theme) {
			continue
		}

		path, err := s.GetWallpaperLocalPath(wp)
		if err != nil {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			result = append(result, wp)
		}
	}
	return result, nil
}
