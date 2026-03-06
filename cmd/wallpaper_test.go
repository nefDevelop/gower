package cmd

import (
	"gower/internal/core"
	"gower/pkg/models"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWallpaperShow(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Add wallpaper to feed
	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	wp := models.Wallpaper{ID: "test-show", URL: "http://example.com/show.jpg", Source: "test", Dimension: "1920x1080"}
	ctrl.AddWallpaperToFeed(wp)

	// Execute command
	output, err := executeCommand(rootCmd, "wallpaper", "test-show")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if !strings.Contains(output, "Details for Wallpaper: test-show") {
		t.Errorf("Expected details header, got: %s", output)
	}
	if !strings.Contains(output, "Dimension: 1920x1080") {
		t.Errorf("Expected dimension info, got: %s", output)
	}
}

func TestWallpaperDeleteFromFeed(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Add wallpaper to feed
	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	wp := models.Wallpaper{ID: "test-delete", URL: "http://example.com/delete.jpg"}
	ctrl.AddWallpaperToFeed(wp)

	// Execute delete command
	output, err := executeCommand(rootCmd, "wallpaper", "test-delete", "--delete")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if !strings.Contains(output, "Wallpaper test-delete removed from feed.") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify it's gone from feed
	feed, _ := ctrl.GetFeedWallpapers()
	for _, item := range feed {
		if item.ID == "test-delete" {
			t.Errorf("Wallpaper was not removed from feed")
		}
	}
}

func TestWallpaperDeleteFile(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Create a dummy local file
	localFilesDir := filepath.Join(tmpDir, "my_wallpapers")
	os.MkdirAll(localFilesDir, 0755)
	localFilePath := filepath.Join(localFilesDir, "local_image.jpg")
	os.WriteFile(localFilePath, []byte("dummy data"), 0644)

	// Add to feed
	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	wp := models.Wallpaper{ID: "local_image.jpg", URL: localFilePath, Source: "local"}
	ctrl.AddWallpaperToFeed(wp)

	// Execute delete with --file and --force (to avoid interactive prompt in test)
	output, err := executeCommand(rootCmd, "wallpaper", "local_image.jpg", "--delete", "--file", "--force")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if !strings.Contains(output, "File deleted from disk.") {
		t.Errorf("Expected file deletion message, got: %s", output)
	}

	// Verify file is gone
	if _, err := os.Stat(localFilePath); !os.IsNotExist(err) {
		t.Errorf("Local file was not deleted")
	}

	// Verify it's gone from feed
	feed, _ := ctrl.GetFeedWallpapers()
	if len(feed) != 0 {
		t.Errorf("Wallpaper was not removed from feed after file deletion")
	}
}

func TestWallpaperDeleteCachedFile(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Create a dummy cached file
	appDir, _ := core.GetAppDir()
	cacheDir := filepath.Join(appDir, "cache", "wallpapers")
	os.MkdirAll(cacheDir, 0755)
	cachedFilePath := filepath.Join(cacheDir, "remote_test.jpg")
	os.WriteFile(cachedFilePath, []byte("dummy data"), 0644)

	// Add to feed
	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	wp := models.Wallpaper{ID: "remote_test", URL: "http://example.com/remote.jpg", Source: "wallhaven"}
	ctrl.AddWallpaperToFeed(wp)

	// Execute delete with --file
	output, err := executeCommand(rootCmd, "wallpaper", "remote_test", "--delete", "--file")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if !strings.Contains(output, "Cached file deleted.") {
		t.Errorf("Expected cache deletion message, got: %s", output)
	}

	// Verify file is gone
	if _, err := os.Stat(cachedFilePath); !os.IsNotExist(err) {
		t.Errorf("Cached file was not deleted")
	}
}
