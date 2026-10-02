package core

import (
	"os"
	"path/filepath"
	"testing"

	"gower/internal/utils"
	"gower/pkg/models"
)

func setupFeedService(t *testing.T) (*FeedService, string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)
	// os.UserConfigDir() da prioridad a XDG_CONFIG_HOME sobre HOME: sin
	// neutralizarlo el test opera sobre el config dir real.
	t.Setenv("XDG_CONFIG_HOME", "")

	dataDir := filepath.Join(tmpDir, ".config", "gower", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	feedPath := filepath.Join(dataDir, "feed.json")
	if err := os.WriteFile(feedPath, []byte("[]"), 0644); err != nil {
		t.Fatal(err)
	}

	favPath := filepath.Join(dataDir, "favorites.json")
	if err := os.WriteFile(favPath, []byte("[]"), 0644); err != nil {
		t.Fatal(err)
	}

	blPath := filepath.Join(dataDir, "blacklist.json")
	if err := os.WriteFile(blPath, []byte("[]"), 0644); err != nil {
		t.Fatal(err)
	}

	parserDir := filepath.Join(tmpDir, ".config", "gower", "data", "parser")
	if err := os.MkdirAll(parserDir, 0755); err != nil {
		t.Fatal(err)
	}

	fm := utils.NewSecureJSONManager()
	cfg := &models.Config{
		Limits: models.LimitsConfig{
			FeedSoftLimit: 100,
			FeedHardLimit: 500,
		},
	}
	fs := NewFeedService(cfg, fm)

	return fs, dataDir
}

func TestFeedService_LoadSaveFeed(t *testing.T) {
	fs, _ := setupFeedService(t)

	feed, err := fs.loadFeed()
	if err != nil {
		t.Fatal(err)
	}
	if len(feed) != 0 {
		t.Errorf("Expected empty feed, got %d items", len(feed))
	}

	wps := []models.Wallpaper{
		{ID: "wh_1", URL: "url1"},
		{ID: "wh_2", URL: "url2"},
	}
	if err := fs.saveFeed(wps); err != nil {
		t.Fatal(err)
	}

	loaded, err := fs.loadFeed()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Errorf("Expected 2 items, got %d", len(loaded))
	}
}

func TestFeedService_LoadSaveBlacklist(t *testing.T) {
	fs, _ := setupFeedService(t)

	bl, err := fs.loadBlacklist()
	if err != nil {
		t.Fatal(err)
	}
	if len(bl) != 0 {
		t.Errorf("Expected empty blacklist, got %v", bl)
	}

	if err := fs.saveBlacklist([]string{"wh_bad1", "wh_bad2"}); err != nil {
		t.Fatal(err)
	}

	loaded, err := fs.loadBlacklist()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Errorf("Expected 2 items, got %d", len(loaded))
	}
}

func TestFeedService_LoadSaveFavorites(t *testing.T) {
	fs, _ := setupFeedService(t)

	favs, err := fs.loadFavorites()
	if err != nil {
		t.Fatal(err)
	}
	if len(favs) != 0 {
		t.Errorf("Expected empty favorites, got %d", len(favs))
	}

	testFavs := []FavoriteWallpaper{
		{Wallpaper: models.Wallpaper{ID: "wh_1"}},
	}
	if err := fs.saveFavorites(testFavs); err != nil {
		t.Fatal(err)
	}

	loaded, err := fs.loadFavorites()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 {
		t.Errorf("Expected 1 favorite, got %d", len(loaded))
	}
}

func TestFeedService_SaveParserSearch(t *testing.T) {
	fs, _ := setupFeedService(t)

	if err := fs.SaveParserSearch("wallhaven", "test_query", []models.Wallpaper{
		{ID: "wh_1"},
	}); err != nil {
		t.Fatal(err)
	}

	results, found := fs.LoadParserSearch("wallhaven", "test_query")
	if !found {
		t.Fatal("Expected parser search to be found")
	}
	if len(results) != 1 || results[0].ID != "wh_1" {
		t.Errorf("Expected 1 result with ID wh_1, got %v", results)
	}
}

func TestFeedService_GetFeedPathString(t *testing.T) {
	fs, _ := setupFeedService(t)
	path := fs.getFeedPathString()
	if path == "" {
		t.Error("Expected non-empty feed path")
	}
	if !stringsSuffix(path, "/.config/gower/data/feed.json") {
		t.Errorf("Expected path to end with /.config/gower/data/feed.json, got %s", path)
	}
}
