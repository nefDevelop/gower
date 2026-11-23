package core

import (
	"context"
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

func (s *DownloadService) localFilename(wp models.Wallpaper) string {
	urlStr := wp.URL
	if qIndex := strings.Index(urlStr, "?"); qIndex != -1 {
		urlStr = urlStr[:qIndex]
	}
	ext := filepath.Ext(urlStr)
	if ext == "" {
		ext = ".jpg"
	}
	return utils.SafeFilename(wp.ID) + ext
}

func (s *DownloadService) GetWallpaperLocalPath(wp models.Wallpaper) (string, error) {
	cacheDir, err := s.getCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, s.localFilename(wp)), nil
}

func (s *DownloadService) GetUserWallpaperPath(wp models.Wallpaper) (string, error) {
	if s.Config.Paths.Wallpapers == "" {
		return "", fmt.Errorf("user wallpapers directory not configured")
	}
	return filepath.Join(s.Config.Paths.Wallpapers, s.localFilename(wp)), nil
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

	safeID := utils.SafeFilename(wp.ID)
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

// DownloadWallpaper descarga (o copia) la imagen al cache y analiza su color.
func (s *DownloadService) DownloadWallpaper(wp models.Wallpaper) (string, error) {
	return s.DownloadWallpaperContext(context.Background(), wp)
}

// DownloadWallpaperContext permite cancelar la descarga desde el llamante.
func (s *DownloadService) DownloadWallpaperContext(ctx context.Context, wp models.Wallpaper) (string, error) {
	filePath, err := s.GetWallpaperLocalPath(wp)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", err
	}

	// Un fichero de tamaño cero es una descarga interrumpida: no se reutiliza.
	if info, err := os.Stat(filePath); err == nil && info.Size() > 0 {
		return filePath, nil
	}

	if !strings.HasPrefix(wp.URL, "http://") && !strings.HasPrefix(wp.URL, "https://") {
		srcPath := wp.URL
		if decoded, err := url.QueryUnescape(srcPath); err == nil {
			srcPath = decoded
		}

		src, err := os.Open(srcPath)
		if err != nil {
			return "", fmt.Errorf("unsupported protocol or missing local file: %s", wp.URL)
		}
		defer func() { _ = src.Close() }()

		if err := writeFileAtomic(filePath, src); err != nil {
			return "", err
		}
	} else {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, wp.URL, nil)
		if err != nil {
			return "", err
		}

		resp, err := utils.DownloadHTTPClient.Do(req) //nolint:bodyclose // cerrado con utils.CloseResponse
		if err != nil {
			return "", err
		}
		defer utils.CloseResponse(resp)

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to download wallpaper: status %d", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
			return "", fmt.Errorf("unexpected content type for wallpaper: %s", ct)
		}

		limited := io.LimitReader(resp.Body, utils.MaxImageBytes)
		if err := writeFileAtomic(filePath, limited); err != nil {
			return "", err
		}
	}

	appDir, err := GetAppDir()
	if err == nil {
		thumbDir := filepath.Join(appDir, "cache", "thumbs")
		thumbPath := filepath.Join(thumbDir, utils.SafeFilename(wp.ID)+".jpg")
		if _, _, err := s.ColorManager.GenerateThumbnail(filePath, thumbPath); err != nil {
			s.log().Error("Failed to generate thumbnail for %s: %v", wp.ID, err)
		}
	}

	if _, err := s.ColorManager.AnalyzeColor(filePath); err != nil {
		s.log().Debug("Color analysis failed/skipped for %s: %v", wp.ID, err)
	}

	s.log().Info("Downloaded wallpaper %s to %s", wp.ID, filePath)
	return filePath, nil
}

// writeFileAtomic escribe el reader en un temporal del mismo directorio y lo
// renombra: una descarga interrumpida nunca deja un fichero parcial válido.
func writeFileAtomic(filePath string, r io.Reader) error {
	dir := filepath.Dir(filePath)
	tmp, err := os.CreateTemp(dir, ".gower-download-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filePath)
}

func (s *DownloadService) FindWallpaperCacheFile(wp models.Wallpaper) (string, bool) {
	appDir, err := GetAppDir()
	if err != nil {
		return "", false
	}
	wallpaperCacheDir := filepath.Join(appDir, "cache", "wallpapers")
	base := filepath.Join(wallpaperCacheDir, utils.SafeFilename(wp.ID))

	// Se exige un punto tras el ID para no confundir "abc" con "abcd.jpg".
	if matches, err := filepath.Glob(base + ".*"); err == nil && len(matches) > 0 {
		return matches[0], true
	}

	if _, err := os.Stat(wp.URL); err == nil {
		return wp.URL, true
	}
	return "", false
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
		var favorites []FavoriteWallpaper
		if err := s.feedManager.ReadJSON(favPath, &favorites); err == nil {
			for _, f := range favorites {
				candidates = append(candidates, f.Wallpaper)
			}
		}
	}

	result := make([]models.Wallpaper, 0, len(candidates))
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
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			result = append(result, wp)
		}
	}
	return result, nil
}
