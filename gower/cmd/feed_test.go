package cmd

import (
	"gower/internal/core"
	"gower/pkg/models"
	"os"
	"strings"
	"testing"
)

func resetFeedFlags() {
	feedPage = 1
	feedLimit = 20
	feedTheme = ""
	feedColor = ""
	feedRefresh = false
	feedForce = false
	feedDetailed = false
	feedAll = false
	feedFromFavorites = false
}

func TestFeedShow(t *testing.T) {
	resetFeedFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Pre-populate feed
	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "test-1", Theme: "dark"})
	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "test-2", Theme: "light"})

	// Test show all
	output, err := executeCommand(rootCmd, "feed", "show")
	if err != nil {
		t.Fatalf("Error executing feed show: %v", err)
	}
	if !strings.Contains(output, "Displaying table") {
		t.Errorf("Expected table output, got: %s", output)
	}
	// Note: displayTable prints the struct with %+v, so IDs should be visible
	if !strings.Contains(output, "test-1") || !strings.Contains(output, "test-2") {
		t.Errorf("Expected wallpapers in output, got: %s", output)
	}

	// Test filter
	resetFeedFlags()
	output, err = executeCommand(rootCmd, "feed", "show", "--theme", "dark")
	if err != nil {
		t.Fatalf("Error executing feed show --theme: %v", err)
	}
	// We expect test-1 (dark) to be present.
	if !strings.Contains(output, "test-1") {
		t.Errorf("Expected test-1 in output")
	}
}

func TestFeedStats(t *testing.T) {
	resetFeedFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)
	executeCommand(rootCmd, "config", "init")

	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "1", Theme: "dark"})

	output, err := executeCommand(rootCmd, "feed", "stats")
	if err != nil {
		t.Fatalf("Error executing feed stats: %v", err)
	}
	if !strings.Contains(output, "Total wallpapers: 1") {
		t.Errorf("Expected Total wallpapers: 1, got: %s", output)
	}
	if !strings.Contains(output, "Dark theme: 1") {
		t.Errorf("Expected Dark theme: 1, got: %s", output)
	}
}

func TestFeedPurge(t *testing.T) {
	resetFeedFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)
	executeCommand(rootCmd, "config", "init")

	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "1"})

	// Purge without force
	output, err := executeCommand(rootCmd, "feed", "purge")
	if err != nil {
		t.Fatalf("Error executing feed purge: %v", err)
	}
	if !strings.Contains(output, "Use --force to confirm") {
		t.Errorf("Expected confirmation message, got: %s", output)
	}

	// Purge with force
	output, err = executeCommand(rootCmd, "feed", "purge", "--force")
	if err != nil {
		t.Fatalf("Error executing feed purge --force: %v", err)
	}
	if !strings.Contains(output, "Feed purged successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify empty
	stats, _ := ctrl.GetFeedStats()
	if stats.Total != 0 {
		t.Errorf("Feed not empty after purge")
	}
}

func TestFeedRandom(t *testing.T) {
	resetFeedFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)
	executeCommand(rootCmd, "config", "init")

	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "1"})

	output, err := executeCommand(rootCmd, "feed", "random")
	if err != nil {
		t.Fatalf("Error executing feed random: %v", err)
	}
	if !strings.Contains(output, "Displaying wallpaper") {
		t.Errorf("Expected wallpaper display, got: %s", output)
	}
}
