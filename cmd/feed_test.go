package cmd

import (
	"encoding/json"
	"gower/internal/core"
	"gower/pkg/models"
	"os"
	"path/filepath"
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
	_ = ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "test-seen", Theme: "dark", Seen: true})
	_ = ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "test-unseen", Theme: "light", Seen: false})

	// Test show all - first time
	output, err := executeCommand(rootCmd, "feed", "show")
	if err != nil {
		t.Fatalf("Error executing feed show: %v", err)
	}

	// The order is now randomized, so we can't check for a specific order.
	// We just check that the header and the correct lines are present.
	if !strings.Contains(output, "ID") || !strings.Contains(output, "SEEN") {
		t.Errorf("Expected table header, got: %s", output)
	}
	if !strings.Contains(output, "test-seen") || !strings.Contains(output, "true") {
		t.Errorf("Expected seen wallpaper to be listed correctly, got: %s", output)
	}
	if !strings.Contains(output, "test-unseen") || !strings.Contains(output, "false") {
		t.Errorf("Expected unseen wallpaper to be listed correctly, got: %s", output)
	}

	// Test show all - second time, the unseen should now be seen
	output2, err := executeCommand(rootCmd, "feed", "show")
	if err != nil {
		t.Fatalf("Error executing feed show (2nd time): %v", err)
	}

	if strings.Contains(output2, "false") {
		t.Errorf("Expected no unseen wallpapers on second run, but found some. Output:\n%s", output2)
	}

	// Test filter
	resetFeedFlags()
	_ = ctrl.PurgeFeed()
	_ = ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "test-dark", Theme: "dark", Seen: false})
	_ = ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "test-light", Theme: "light", Seen: false})

	output, err = executeCommand(rootCmd, "feed", "show", "--theme", "dark")
	if err != nil {
		t.Fatalf("Error executing feed show --theme: %v", err)
	}
	if !strings.Contains(output, "test-dark") {
		t.Errorf("Expected test-dark in output")
	}
	if strings.Contains(output, "test-light") {
		t.Errorf("Did not expect test-light in output")
	}
}

func TestFeedStats(t *testing.T) {
	resetFeedFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)
	executeCommand(rootCmd, "config", "init")

	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	_ = ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "1", Theme: "dark"})

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
	_ = ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "1"})

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
	_ = ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "1"})

	output, err := executeCommand(rootCmd, "feed", "random")
	if err != nil {
		t.Fatalf("Error executing feed random: %v", err)
	}
	if !strings.Contains(output, "Displaying wallpaper") {
		t.Errorf("Expected wallpaper display, got: %s", output)
	}
}

func TestFeedGetColors(t *testing.T) {
	resetFeedFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Manually create colors.json with feed palette entries
	colorsPath := filepath.Join(tmpDir, ".config", "gower", "data", "colors.json")
	expectedColors := []string{"#AAAAAA", "#BBBBBB", "#CCCCCC"}
	paletteJSON := `{"feed_palette": ["#AAAAAA", "#BBBBBB", "#CCCCCC"], "favorites_palette": []}`
	if err := os.WriteFile(colorsPath, []byte(paletteJSON), 0644); err != nil {
		t.Fatalf("Failed to write colors.json for test: %v", err)
	}

	// Test plain output
	output, err := executeCommand(rootCmd, "feed", "get", "colors")
	if err != nil {
		t.Fatalf("Error executing feed get colors: %v", err)
	}

	for _, color := range expectedColors {
		if !strings.Contains(output, color) {
			t.Errorf("Expected color %s in output, got: %s", color, output)
		}
	}

	// Test JSON output
	config.JSONOutput = true
	output, err = executeCommand(rootCmd, "feed", "get", "colors")
	config.JSONOutput = false // Reset for other tests
	if err != nil {
		t.Fatalf("Error executing feed get colors with JSON output: %v", err)
	}

	var actualColors []string
	if err := json.Unmarshal([]byte(output), &actualColors); err != nil {
		t.Fatalf("Failed to unmarshal JSON output: %v", err)
	}

	if len(actualColors) != len(expectedColors) {
		t.Errorf("Expected %d colors, got %d", len(expectedColors), len(actualColors))
	}
	for i, color := range expectedColors {
		if actualColors[i] != color {
			t.Errorf("Expected color %s at index %d, got %s", color, i, actualColors[i])
		}
	}
}
