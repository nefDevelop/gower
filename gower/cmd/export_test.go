package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gower/internal/core"
	"gower/pkg/models"
)

func TestExportConfig(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")
	executeCommand(rootCmd, "config", "set", "behavior.theme=light")

	exportFile := filepath.Join(tmpDir, "exported_config.json")
	output, err := executeCommand(rootCmd, "export", "config", "--file", exportFile)
	if err != nil {
		t.Fatalf("Error executing export config: %v", err)
	}
	if !strings.Contains(output, "Configuration exported to:") {
		t.Errorf("Expected 'Configuration exported to:', got: %s", output)
	}

	exportedData, err := ioutil.ReadFile(exportFile)
	if err != nil {
		t.Fatalf("Error reading exported config file: %v", err)
	}

	var exportedConfig models.Config
	if err := json.Unmarshal(exportedData, &exportedConfig); err != nil {
		t.Fatalf("Error unmarshalling exported config: %v", err)
	}

	if exportedConfig.Behavior.Theme != "light" {
		t.Errorf("Expected exported config theme to be 'light', got '%s'", exportedConfig.Behavior.Theme)
	}
}

func TestExportFeed(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")
	// Add some dummy wallpapers to the feed
	cfg, _ := loadConfig()
	controller := core.NewController(cfg)
	controller.AddWallpaperToFeed(models.Wallpaper{ID: "feed-1", URL: "url-1", Source: "test", Theme: "dark"})
	controller.AddWallpaperToFeed(models.Wallpaper{ID: "feed-2", URL: "url-2", Source: "test", Theme: "light"})

	exportFile := filepath.Join(tmpDir, "exported_feed.json")
	output, err := executeCommand(rootCmd, "export", "feed", "--file", exportFile)
	if err != nil {
		t.Fatalf("Error executing export feed: %v", err)
	}
	if !strings.Contains(output, "Feed exported to:") {
		t.Errorf("Expected 'Feed exported to:', got: %s", output)
	}

	exportedData, err := ioutil.ReadFile(exportFile)
	if err != nil {
		t.Fatalf("Error reading exported feed file: %v", err)
	}

	var exportedFeed []models.Wallpaper
	if err := json.Unmarshal(exportedData, &exportedFeed); err != nil {
		t.Fatalf("Error unmarshalling exported feed: %v", err)
	}

	if len(exportedFeed) != 2 {
		t.Errorf("Expected 2 wallpapers in exported feed, got %d", len(exportedFeed))
	}
	if exportedFeed[0].ID != "feed-1" || exportedFeed[1].ID != "feed-2" {
		t.Errorf("Exported feed content mismatch")
	}
}

func TestExportAllZip(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Create dummy cache file to test image inclusion
	cacheDir := filepath.Join(tmpDir, ".gower", "cache", "wallpapers")
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(filepath.Join(cacheDir, "test_img.jpg"), []byte("fake image"), 0644)

	exportFile := filepath.Join(tmpDir, "export.zip")
	output, err := executeCommand(rootCmd, "export", "all", "--file", exportFile, "--include-images")
	if err != nil {
		t.Fatalf("Error executing export all zip: %v", err)
	}
	if !strings.Contains(output, "All data exported to:") {
		t.Errorf("Expected 'All data exported to:', got: %s", output)
	}
	if _, err := os.Stat(exportFile); os.IsNotExist(err) {
		t.Errorf("Export zip file was not created")
	}
}

func TestExportAll(t *testing.T) {
	exportFile = "" // Reset global flag variable to avoid pollution from previous tests
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")
	executeCommand(rootCmd, "config", "set", "behavior.theme=dark")
	executeCommand(rootCmd, "favorites", "add", "fav-1")

	// Add some dummy wallpapers to the feed
	cfg, _ := loadConfig()
	controller := core.NewController(cfg)
	controller.AddWallpaperToFeed(models.Wallpaper{ID: "feed-all-1", URL: "url-all-1", Source: "test"})

	exportDir := filepath.Join(tmpDir, "gower_all_export")
	output, err := executeCommand(rootCmd, "export", "all", exportDir)
	if err != nil {
		t.Fatalf("Error executing export all: %v", err)
	}
	if !strings.Contains(output, "All data exported to directory:") {
		t.Errorf("Expected 'All data exported to directory:', got: %s", output)
	}

	// Verify config.json
	configPath := filepath.Join(exportDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config.json not found in exported directory")
	}
	configData, _ := ioutil.ReadFile(configPath)
	var exportedConfig models.Config
	json.Unmarshal(configData, &exportedConfig)
	if exportedConfig.Behavior.Theme != "dark" {
		t.Errorf("Exported config theme mismatch")
	}

	// Verify favorites.json
	favoritesPath := filepath.Join(exportDir, "favorites.json")
	if _, err := os.Stat(favoritesPath); os.IsNotExist(err) {
		t.Errorf("favorites.json not found in exported directory")
	}
	favoritesData, _ := ioutil.ReadFile(favoritesPath)
	var exportedFavorites []models.Wallpaper
	json.Unmarshal(favoritesData, &exportedFavorites)
	if len(exportedFavorites) != 1 || exportedFavorites[0].ID != "fav-1" {
		t.Errorf("Exported favorites content mismatch")
	}

	// Verify feed.json
	feedPath := filepath.Join(exportDir, "feed.json")
	if _, err := os.Stat(feedPath); os.IsNotExist(err) {
		t.Errorf("feed.json not found in exported directory")
	}
	feedData, _ := ioutil.ReadFile(feedPath)
	var exportedFeed []models.Wallpaper
	json.Unmarshal(feedData, &exportedFeed)
	if len(exportedFeed) != 1 || exportedFeed[0].ID != "feed-all-1" {
		t.Errorf("Exported feed content mismatch")
	}
}
