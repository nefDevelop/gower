package core

import (
	"encoding/json"
	"gower/pkg/models"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestHome(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "gower-core-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir) // Windows

	// Create data directory
	if err := os.MkdirAll(filepath.Join(tmpDir, ".gower", "data"), 0755); err != nil {
		t.Fatal(err)
	}

	return tmpDir
}

func TestController_AddAndGetFeed(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	cfg := &models.Config{}
	ctrl := NewController(cfg)

	wp1 := models.Wallpaper{ID: "1", Theme: "dark", Source: "test"}
	wp2 := models.Wallpaper{ID: "2", Theme: "light", Source: "test"}

	if err := ctrl.AddWallpaperToFeed(wp1); err != nil {
		t.Fatalf("Failed to add wp1: %v", err)
	}
	if err := ctrl.AddWallpaperToFeed(wp2); err != nil {
		t.Fatalf("Failed to add wp2: %v", err)
	}

	// Test GetFeed with pagination
	feed, err := ctrl.GetFeed(1, 10, "", "", "")
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}
	if len(feed) != 2 {
		t.Errorf("Expected 2 wallpapers, got %d", len(feed))
	}

	// Test GetFeed with theme filter
	feedDark, err := ctrl.GetFeed(1, 10, "", "dark", "")
	if err != nil {
		t.Fatalf("GetFeed dark failed: %v", err)
	}
	if len(feedDark) != 1 || feedDark[0].ID != "1" {
		t.Errorf("Expected 1 dark wallpaper (ID 1), got %v", feedDark)
	}
}

func TestController_PurgeFeed(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	cfg := &models.Config{}
	ctrl := NewController(cfg)

	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "1"})

	if err := ctrl.PurgeFeed(); err != nil {
		t.Fatalf("PurgeFeed failed: %v", err)
	}

	feed, _ := ctrl.GetFeed(1, 10, "", "", "")
	if len(feed) != 0 {
		t.Errorf("Feed not empty after purge")
	}
}

func TestController_GetFeedStats(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	cfg := &models.Config{}
	ctrl := NewController(cfg)

	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "1", Theme: "dark"})
	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "2", Theme: "light"})
	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "3", Theme: "dark"})

	stats, err := ctrl.GetFeedStats()
	if err != nil {
		t.Fatalf("GetFeedStats failed: %v", err)
	}

	if stats.Total != 3 {
		t.Errorf("Expected total 3, got %d", stats.Total)
	}
	if stats.DarkCount != 2 {
		t.Errorf("Expected dark 2, got %d", stats.DarkCount)
	}
	if stats.LightCount != 1 {
		t.Errorf("Expected light 1, got %d", stats.LightCount)
	}
}

func TestController_Blacklist(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	cfg := &models.Config{}
	ctrl := NewController(cfg)

	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "1"})
	ctrl.AddWallpaperToFeed(models.Wallpaper{ID: "2"})

	// Manually write blacklist file
	blacklistPath := filepath.Join(tmpDir, ".gower", "data", "blacklist.json")
	if err := os.WriteFile(blacklistPath, []byte(`["1"]`), 0644); err != nil {
		t.Fatalf("Failed to write blacklist: %v", err)
	}

	feed, err := ctrl.GetFeed(1, 10, "", "", "")
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}

	if len(feed) != 1 {
		t.Errorf("Expected 1 wallpaper after blacklist, got %d", len(feed))
	}
	if len(feed) > 0 && feed[0].ID != "2" {
		t.Errorf("Expected ID 2, got %s", feed[0].ID)
	}
}

func createDummyImage(t *testing.T, path string) {
	width := 100
	height := 100
	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}
	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	// Fill with red
	red := color.RGBA{255, 0, 0, 255}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, red)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestController_SyncFeed(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	cfg := &models.Config{
		Limits: models.LimitsConfig{
			FeedHardLimit: 100,
		},
		Search: models.SearchConfig{
			Tolerance: 0.1,
		},
	}
	ctrl := NewController(cfg)

	// Create a dummy image
	imgPath := filepath.Join(tmpDir, "test_image.png")
	createDummyImage(t, imgPath)

	// Create a parser cache file
	parserDir := filepath.Join(tmpDir, ".gower", "data", "parser")
	if err := os.MkdirAll(parserDir, 0755); err != nil {
		t.Fatal(err)
	}

	searchResult := ParserSearch{
		Date:  time.Now(),
		Query: "test",
		Results: []models.Wallpaper{
			{
				ID:        "test_1",
				URL:       imgPath, // Local path
				Thumbnail: imgPath, // Use same for thumb source
				Source:    "test",
			},
		},
	}

	data, _ := json.Marshal([]ParserSearch{searchResult})
	if err := os.WriteFile(filepath.Join(parserDir, "test_provider.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	// Run SyncFeed
	added, repaired, err := ctrl.SyncFeed()
	if err != nil {
		t.Fatalf("SyncFeed failed: %v", err)
	}

	if added != 1 {
		t.Errorf("Expected 1 added wallpaper, got %d", added)
	}
	if repaired != 0 {
		t.Errorf("Expected 0 repaired wallpapers, got %d", repaired)
	}

	// Verify feed
	feed, _ := ctrl.GetFeed(1, 10, "", "", "")
	if len(feed) != 1 {
		t.Fatalf("Expected 1 item in feed, got %d", len(feed))
	}
	wp := feed[0]
	if wp.ID != "test_1" {
		t.Errorf("Expected ID test_1, got %s", wp.ID)
	}
	if wp.Color == "" {
		t.Error("Expected color to be analyzed")
	}
	// Since image is red, it should be close to #FF0000
	// JPEG compression might cause slight variation (e.g. #FE0000)
	if wp.Color != "#FF0000" && wp.Color != "#FE0000" {
		t.Errorf("Expected color #FF0000 or #FE0000, got %s", wp.Color)
	}
	if wp.Ratio == "" {
		t.Error("Expected ratio to be calculated")
	}

	// Verify colors.json
	colorsPath := filepath.Join(tmpDir, ".gower", "data", "colors.json")
	colorsData, _ := os.ReadFile(colorsPath)
	if !strings.Contains(string(colorsData), "#FF0000") && !strings.Contains(string(colorsData), "#FE0000") {
		t.Error("Expected colors.json to contain #FF0000 or #FE0000")
	}
}

func TestController_GetWallpaper(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	cfg := &models.Config{}
	ctrl := NewController(cfg)

	// 1. Setup mock data files
	feedPath := filepath.Join(tmpDir, ".gower", "data", "feed.json")
	favPath := filepath.Join(tmpDir, ".gower", "data", "favorites.json")

	feedWallpapers := []models.Wallpaper{{ID: "feed_wall", URL: "http://example.com/feed.jpg"}}
	favWallpapers := []struct {
		models.Wallpaper
		Notes string `json:"notes,omitempty"`
	}{{Wallpaper: models.Wallpaper{ID: "fav_wall", URL: "http://example.com/fav.jpg"}}}

	feedData, _ := json.Marshal(feedWallpapers)
	favData, _ := json.Marshal(favWallpapers)

	os.WriteFile(feedPath, feedData, 0644)
	os.WriteFile(favPath, favData, 0644)

	// 2. Run tests
	t.Run("finds wallpaper in feed", func(t *testing.T) {
		wp, err := ctrl.GetWallpaper("feed_wall")
		if err != nil {
			t.Fatalf("Expected to find wallpaper, but got error: %v", err)
		}
		if wp.ID != "feed_wall" {
			t.Errorf("Expected ID 'feed_wall', got '%s'", wp.ID)
		}
	})

	t.Run("finds wallpaper in favorites", func(t *testing.T) {
		wp, err := ctrl.GetWallpaper("fav_wall")
		if err != nil {
			t.Fatalf("Expected to find wallpaper, but got error: %v", err)
		}
		if wp.ID != "fav_wall" {
			t.Errorf("Expected ID 'fav_wall', got '%s'", wp.ID)
		}
	})

	t.Run("returns error when not found", func(t *testing.T) {
		_, err := ctrl.GetWallpaper("not_found_wall")
		if err == nil {
			t.Fatal("Expected an error for a wallpaper that does not exist, but got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected error message to contain 'not found', got '%v'", err)
		}
	})
}

func TestController_AnalyzeFeed(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	cfg := &models.Config{}
	ctrl := NewController(cfg)

	// 1. Create a dummy source image
	srcImgPath := filepath.Join(tmpDir, "source.png")
	createDummyImage(t, srcImgPath)

	// 2. Add item to feed
	wpID := "test_analyze"
	wp := models.Wallpaper{
		ID:        wpID,
		URL:       srcImgPath,
		Thumbnail: srcImgPath,
		Theme:     "dark",
	}
	if err := ctrl.AddWallpaperToFeed(wp); err != nil {
		t.Fatalf("Failed to add wallpaper: %v", err)
	}

	// 3. Ensure thumbnail does NOT exist initially
	thumbPath := filepath.Join(tmpDir, ".gower", "cache", "thumbs", wpID+".jpg")

	// Case 1: AnalyzeFeed(false, false) - Should generate missing thumbnail
	if err := ctrl.AnalyzeFeed(false, false, nil); err != nil {
		t.Fatalf("AnalyzeFeed(false, false) failed: %v", err)
	}

	info1, err := os.Stat(thumbPath)
	if os.IsNotExist(err) {
		t.Fatal("Thumbnail should have been generated")
	}

	// Case 2: AnalyzeFeed(true, true) - Force regeneration
	time.Sleep(50 * time.Millisecond) // Ensure fs timestamp difference
	if err := ctrl.AnalyzeFeed(true, true, nil); err != nil {
		t.Fatalf("AnalyzeFeed(true, true) failed: %v", err)
	}

	info2, err := os.Stat(thumbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info2.ModTime().After(info1.ModTime()) {
		t.Error("Thumbnail should have been regenerated with force=true and all=true")
	}
}

func TestController_AnalyzeFavorites(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	cfg := &models.Config{}
	ctrl := NewController(cfg)

	// Create dummy image
	srcImgPath := filepath.Join(tmpDir, "fav_source.png")
	createDummyImage(t, srcImgPath)

	// Manually create favorites.json
	favPath := filepath.Join(tmpDir, ".gower", "data", "favorites.json")
	favContent := `[{"id":"fav1","url":"` + srcImgPath + `","source":"test"}]`
	os.WriteFile(favPath, []byte(favContent), 0644)

	// Analyze
	if err := ctrl.AnalyzeFavorites(false, false, nil); err != nil {
		t.Fatalf("AnalyzeFavorites failed: %v", err)
	}

	// Check thumbnail
	thumbPath := filepath.Join(tmpDir, ".gower", "cache", "thumbs", "fav1.jpg")
	if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
		t.Error("Thumbnail for favorite should have been generated")
	}
}

func TestController_GetFeed_Algorithm(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	cfg := &models.Config{}
	ctrl := NewController(cfg)

	// 1. Create a feed with mixed seen/unseen items
	initialFeed := []models.Wallpaper{
		{ID: "seen_1", Seen: true},
		{ID: "unseen_1", Seen: false},
		{ID: "seen_2", Seen: true},
		{ID: "unseen_2", Seen: false},
	}
	if err := ctrl.saveFeed(initialFeed); err != nil {
		t.Fatalf("Failed to save initial feed: %v", err)
	}

	// 2. Get the feed - first call
	result1, err := ctrl.GetFeed(1, 4, "", "", "")
	if err != nil {
		t.Fatalf("GetFeed failed: %v", err)
	}

	if len(result1) != 4 {
		t.Fatalf("Expected 4 items, got %d", len(result1))
	}

	// 3. Verify that the returned items have their original 'seen' status
	unseenCount := 0
	for _, wp := range result1 {
		if !wp.Seen {
			unseenCount++
		}
	}
	if unseenCount != 2 {
		t.Errorf("Expected 2 items with Seen:false in the result, got %d", unseenCount)
	}

	// Now check the file on disk to see if they were marked as seen
	savedFeed, err := ctrl.loadFeed()
	if err != nil {
		t.Fatalf("Failed to load feed after GetFeed: %v", err)
	}
	for _, wp := range savedFeed {
		if !wp.Seen {
			t.Errorf("Expected all items to be marked as seen in the file, but %s is not", wp.ID)
		}
	}

	// 4. Get the feed again to check for stable order
	// First, reset the feed file to the initial state
	if err := ctrl.saveFeed(initialFeed); err != nil {
		t.Fatalf("Failed to reset feed: %v", err)
	}
	result2, err := ctrl.GetFeed(1, 4, "", "", "")
	if err != nil {
		t.Fatalf("Second GetFeed failed: %v", err)
	}

	// Check if orders are identical
	for i := range result1 {
		if result1[i].ID != result2[i].ID {
			t.Errorf("Expected stable order, but it changed. Pos %d: %s vs %s", i, result1[i].ID, result2[i].ID)
		}
	}
}
