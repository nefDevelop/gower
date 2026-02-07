package cmd

import (
	"gower/internal/core"
	"gower/pkg/models"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Mock Controller for testing purposes
type MockController struct {
	core.Controller
	MockWallpaper        *models.Wallpaper
	MockWallpaperError   error
	MockDownloadPath     string
	MockDownloadError    error
	MockRandomWallpaper  models.Wallpaper
	MockRandomError      error
	MockFavorites        []FavoriteWallpaper
	MockFavoritesError   error
	MockWallpaperChanger *MockSetWallpaperChanger // Embed our mock changer
}

func (m *MockController) GetWallpaper(id string) (*models.Wallpaper, error) {
	return m.MockWallpaper, m.MockWallpaperError
}

func (m *MockController) DownloadWallpaper(wp models.Wallpaper) (string, error) {
	return m.MockDownloadPath, m.MockDownloadError
}

func (m *MockController) GetRandomFromFeed(theme string) (models.Wallpaper, error) {
	return m.MockRandomWallpaper, m.MockRandomError
}

// Mock WallpaperChanger for testing purposes (same as in status_monitor_test.go)
type MockSetWallpaperChanger struct {
	core.WallpaperChanger
	MockMonitors       []core.Monitor
	MockDetectError    error
	SetWallpapersCalls []struct {
		Paths    []string
		Monitors []core.Monitor
		Mode     string
	}
	MockSetWallpapersError error
}

func (m *MockSetWallpaperChanger) DetectMonitors() ([]core.Monitor, error) {
	return m.MockMonitors, m.MockDetectError
}

func (m *MockSetWallpaperChanger) SetWallpapers(paths []string, monitors []core.Monitor, multiMonitor string) error {
	m.SetWallpapersCalls = append(m.SetWallpapersCalls, struct {
		Paths    []string
		Monitors []core.Monitor
		Mode     string
	}{Paths: paths, Monitors: monitors, Mode: multiMonitor})
	return m.MockSetWallpapersError
}

// Override NewController to return our mock
var originalNewController = core.NewController
var originalNewWallpaperChanger = core.NewWallpaperChanger

func setupMocks(t *testing.T) (*MockController, *MockSetWallpaperChanger) {
	// Setup temp home for config
	tmpDir, err := os.MkdirTemp("", "gower-test-set-monitor")
	if err != nil {
		t.Fatal(err)
	}
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// Create config file
	configDir := filepath.Join(tmpDir, ".gower")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{}"), 0644)

	// Initialize real controller to get valid managers
	cfg := &models.Config{
		Behavior: models.BehaviorConfig{
			MultiMonitor: "clone",
		},
	}
	realCtrl := originalNewController(cfg)

	mockChanger := &MockSetWallpaperChanger{}
	mockController := &MockController{
		MockWallpaperChanger: mockChanger,
		Controller:           *realCtrl,
	}

	core.NewController = func(cfg *models.Config) *core.Controller {
		return &mockController.Controller
	}
	core.NewWallpaperChanger = func(desktopEnv string, respectDarkMode ...bool) *core.WallpaperChanger {
		wc := &core.WallpaperChanger{Env: desktopEnv}
		wc.SetWallpapersFunc = mockChanger.SetWallpapers
		wc.DetectMonitorsFunc = mockChanger.DetectMonitors
		return wc
	}

	// Mock loadConfig and loadState to avoid file system access during tests
	originalLoadConfig := loadConfig
	loadConfig = func() (*models.Config, error) {
		return &models.Config{
			Behavior: models.BehaviorConfig{
				MultiMonitor: "clone", // Default behavior
			},
		}, nil
	}
	originalLoadState := loadState
	loadState = func() (*State, error) {
		return &State{}, nil
	}
	originalSaveState := saveState
	saveState = func(s *State) error { return nil }

	t.Cleanup(func() {
		core.NewController = originalNewController
		core.NewWallpaperChanger = originalNewWallpaperChanger
		loadConfig = originalLoadConfig
		loadState = originalLoadState
		saveState = originalSaveState
		// Reset flags
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tmpDir)
		setID = ""
		setURL = ""
		setRandom = false
		setTheme = ""
		setFromFavorites = false
		setMultiMonitor = ""
		setCommand = ""
		setNoDownload = false
		setTargetMonitor = ""
	})

	return mockController, mockChanger
}

func TestSetTargetMonitor(t *testing.T) {
	mockController, mockChanger := setupMocks(t)

	// Mock server for image download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return a 1x1 pixel PNG
		w.Write([]byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
			0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
			0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
			0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
		})
	}))
	defer server.Close()

	// Mock a wallpaper and download path
	testWallpaper := models.Wallpaper{ID: "test_id", URL: server.URL + "/test.jpg", Source: "test"}
	mockController.MockWallpaper = &testWallpaper

	// Add wallpaper to feed so GetWallpaper finds it
	mockController.Controller.AddWallpaperToFeed(testWallpaper)

	// Mock monitors
	mockChanger.MockMonitors = []core.Monitor{
		{ID: "eDP-1", Name: "eDP-1", Primary: true},
		{ID: "DP-1", Name: "DP-1", Primary: false},
	}

	// Set flags for the test
	setID = "test_id"
	setTargetMonitor = "DP-1"

	// Execute the command
	rootCmd.SetArgs([]string{"set", "--id", "test_id", "--target-monitor", "DP-1"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("set command failed: %v", err)
	}

	// Assertions
	if len(mockChanger.SetWallpapersCalls) != 1 {
		t.Fatalf("SetWallpapers not called or called multiple times. Calls: %d", len(mockChanger.SetWallpapersCalls))
	}

	call := mockChanger.SetWallpapersCalls[0]
	// The real DownloadWallpaper will be called, so we expect the cache path
	expectedPath, _ := mockController.Controller.GetWallpaperLocalPath(testWallpaper)
	if len(call.Paths) != 1 || call.Paths[0] != expectedPath {
		t.Errorf("SetWallpapers called with incorrect paths. Expected %s, got %v", expectedPath, call.Paths)
	}
	if len(call.Monitors) != 1 || call.Monitors[0].ID != "DP-1" {
		t.Errorf("SetWallpapers called with incorrect monitors: %v", call.Monitors)
	}
	if call.Mode != "clone" { // Default mode when target-monitor is used for single wallpaper
		t.Errorf("SetWallpapers called with incorrect mode: %s", call.Mode)
	}
}

func TestSetTargetMonitorNotFound(t *testing.T) {
	mockController, mockChanger := setupMocks(t)

	// Mock a wallpaper
	testWallpaper := models.Wallpaper{ID: "test_id", URL: "http://example.com/test.jpg", Source: "test"}
	mockController.MockWallpaper = &testWallpaper

	// Add to feed
	mockController.Controller.AddWallpaperToFeed(testWallpaper)

	// Mock monitors
	mockChanger.MockMonitors = []core.Monitor{
		{ID: "eDP-1", Name: "eDP-1", Primary: true},
	}

	// Set flags for the test
	setID = "test_id"
	setTargetMonitor = "invalid_monitor"

	// Execute the command
	rootCmd.SetArgs([]string{"set", "--id", "test_id", "--target-monitor", "invalid_monitor"})
	err := rootCmd.Execute()

	if err == nil {
		t.Fatal("Expected an error for invalid monitor, but got none.")
	}
	if !strings.Contains(err.Error(), "monitor 'invalid_monitor' not found") {
		t.Errorf("Expected 'monitor not found' error, got: %v", err)
	}
}

func TestSetDistinctRandomMultiMonitor(t *testing.T) {
	mockController, mockChanger := setupMocks(t)

	// Mock server for image download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
			0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
			0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
			0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
		})
	}))
	defer server.Close()

	// Mock random wallpapers
	mockController.MockRandomWallpaper = models.Wallpaper{ID: "random_id_1", URL: server.URL + "/rand1.jpg", Source: "test"}

	// Add random wallpapers to feed so GetRandomFromFeed finds them
	mockController.Controller.AddWallpaperToFeed(mockController.MockRandomWallpaper)
	// Add a second one for the second monitor
	wp2 := mockController.MockRandomWallpaper
	wp2.ID = "random_id_2"
	wp2.URL = server.URL + "/rand2.jpg"
	mockController.Controller.AddWallpaperToFeed(wp2)

	// Mock monitors
	mockChanger.MockMonitors = []core.Monitor{
		{ID: "eDP-1", Name: "eDP-1", Primary: true},
		{ID: "DP-1", Name: "DP-1", Primary: false},
	}

	// Set flags for the test
	setRandom = true
	setMultiMonitor = "distinct"

	// Execute the command
	rootCmd.SetArgs([]string{"set", "--random", "--multi-monitor", "distinct"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("set command failed: %v", err)
	}

	// Assertions
	if len(mockChanger.SetWallpapersCalls) != 1 {
		t.Fatalf("SetWallpapers not called or called multiple times. Calls: %d", len(mockChanger.SetWallpapersCalls))
	}

	call := mockChanger.SetWallpapersCalls[0]
	if len(call.Paths) != 2 {
		t.Errorf("SetWallpapers called with incorrect number of paths: %d", len(call.Paths))
	}
	if len(call.Monitors) != 2 {
		t.Errorf("SetWallpapers called with incorrect number of monitors: %d", len(call.Monitors))
	}
	if call.Mode != "distinct" {
		t.Errorf("SetWallpapers called with incorrect mode: %s", call.Mode)
	}
}
