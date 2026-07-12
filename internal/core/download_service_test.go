package core

import (
	"gower/internal/utils"
	"gower/pkg/models"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadService_GetWallpaperLocalPath(t *testing.T) {
	cfg := &models.Config{}
	cm := NewColorManager()
	fm := utils.NewSecureJSONManager()
	ds := NewDownloadService(cfg, cm, fm)

	wp := models.Wallpaper{
		ID:  "wh_test123",
		URL: "https://example.com/image.jpg",
	}

	path, err := ds.GetWallpaperLocalPath(wp)
	if err != nil {
		t.Fatalf("GetWallpaperLocalPath failed: %v", err)
	}

	expectedSuffix := "cache/wallpapers/wh_test123.jpg"
	if !stringsSuffix(path, expectedSuffix) {
		t.Errorf("Expected path to end with %s, got %s", expectedSuffix, path)
	}
}

func TestDownloadService_GetWallpaperLocalPath_NoExt(t *testing.T) {
	cfg := &models.Config{}
	cm := NewColorManager()
	fm := utils.NewSecureJSONManager()
	ds := NewDownloadService(cfg, cm, fm)

	wp := models.Wallpaper{
		ID:  "ns_2024-01-01",
		URL: "https://example.com/image",
	}

	path, err := ds.GetWallpaperLocalPath(wp)
	if err != nil {
		t.Fatalf("GetWallpaperLocalPath failed: %v", err)
	}

	if !stringsSuffix(path, "/ns_2024-01-01.jpg") {
		t.Errorf("Expected path to end with .jpg, got %s", path)
	}
}

func TestDownloadService_FindInCollection_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &models.Config{
		Paths: models.PathsConfig{
			Wallpapers: tmpDir,
		},
	}
	cm := NewColorManager()
	fm := utils.NewSecureJSONManager()
	ds := NewDownloadService(cfg, cm, fm)

	wp := models.Wallpaper{
		ID:  "wh_nonexistent",
		URL: "https://example.com/img.jpg",
	}

	_, found := ds.FindInCollection(wp)
	if found {
		t.Error("Expected file not to be found in empty collection")
	}
}

func TestDownloadService_FindInCollection_ExactMatch(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "wh_test123.jpg")
	if err := os.WriteFile(imgPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &models.Config{
		Paths: models.PathsConfig{
			Wallpapers: tmpDir,
		},
	}
	cm := NewColorManager()
	fm := utils.NewSecureJSONManager()
	ds := NewDownloadService(cfg, cm, fm)

	wp := models.Wallpaper{
		ID:  "wh_test123",
		URL: "https://example.com/img.jpg",
	}

	path, found := ds.FindInCollection(wp)
	if !found {
		t.Fatal("Expected file to be found in collection")
	}
	if path != imgPath {
		t.Errorf("Expected path %s, got %s", imgPath, path)
	}
}

func TestDownloadService_FindInCollection_TaggedMatch(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "wh_test123 [d].jpg")
	if err := os.WriteFile(imgPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &models.Config{
		Paths: models.PathsConfig{
			Wallpapers: tmpDir,
		},
	}
	cm := NewColorManager()
	fm := utils.NewSecureJSONManager()
	ds := NewDownloadService(cfg, cm, fm)

	wp := models.Wallpaper{
		ID:  "wh_test123",
		URL: "https://example.com/img.jpg",
	}

	path, found := ds.FindInCollection(wp)
	if !found {
		t.Fatal("Expected tagged file to be found")
	}
	if path != imgPath {
		t.Errorf("Expected path %s, got %s", imgPath, path)
	}
}

func stringsSuffix(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
