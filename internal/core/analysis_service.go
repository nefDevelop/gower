package core

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

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

func NewAnalysisService(cfg *models.Config, cm *ColorManager, fm *utils.SecureJSONManager) *AnalysisService {
	return &AnalysisService{
		Config:       cfg,
		ColorManager: cm,
		feedManager:  fm,
	}
}

// rebuildColorsIndex genera las paletas del feed y de favoritos y las guarda en
// colors.json.
func (s *AnalysisService) rebuildColorsIndex(feed []models.Wallpaper) error {
	appDir, err := GetAppDir()
	if err != nil {
		return err
	}
	path := filepath.Join(appDir, "data", "colors.json")

	feedColors := make([]string, 0, len(feed))
	for _, wp := range feed {
		if wp.Color != "" {
			feedColors = append(feedColors, wp.Color)
		}
	}

	favPath := filepath.Join(appDir, "data", "favorites.json")
	var favorites []FavoriteWallpaper
	var favColors []string
	if err := s.feedManager.ReadJSON(favPath, &favorites); err == nil {
		for _, fav := range favorites {
			if fav.Color != "" {
				favColors = append(favColors, fav.Color)
			}
		}
	}

	output := struct {
		FeedPalette      []string `json:"feed_palette"`
		FavoritesPalette []string `json:"favorites_palette"`
	}{
		FeedPalette:      s.ColorManager.GenerateDynamicPalette(feedColors, 16),
		FavoritesPalette: s.ColorManager.GenerateDynamicPalette(favColors, 16),
	}

	return utils.WithFileLock(path, func() error {
		return s.feedManager.WriteJSON(path, output)
	})
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

// isValidImage valida resolución y relación de aspecto. Si la configuración de
// aspect_ratio es inválida se registra el problema y no se filtra, en vez de
// desactivar la validación en silencio.
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
		if len(parts) != 2 {
			s.log().Error("Invalid aspect_ratio config %q: expected W:H", target)
			return true, "malformed aspect ratio config"
		}
		w, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		h, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 != nil || err2 != nil || h == 0 {
			s.log().Error("Invalid aspect_ratio config %q: values must be numeric and non-zero", target)
			return true, "malformed aspect ratio config"
		}
		targetRatio = w / h
	} else {
		parsed, err := strconv.ParseFloat(target, 64)
		if err != nil {
			s.log().Error("Invalid aspect_ratio config %q: %v", target, err)
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
