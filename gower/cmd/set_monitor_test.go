package cmd

import (
	"bytes"
	"fmt"
	"gower/internal/core"
	"gower/pkg/models"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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
	MockFavorites        []models.Favorite
	MockFavoritesError   error
	MockWallpaperChanger *MockWallpaperChanger // Embed our mock changer
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
type MockWallpaperChanger struct {
	core.WallpaperChanger
	MockMonitors      []core.Monitor
	MockDetectError   error
	SetWallpapersCalls []struct {
		Paths   []string
		Monitors []core.Monitor
		Mode    string
	}
	MockSetWallpapersError error
}

func (m *MockWallpaperChanger) DetectMonitors() ([]core.Monitor, error) {
	return m.MockMonitors, m.MockDetectError
}

func (m *MockWallpaperChanger) SetWallpapers(paths []string, monitors []core.Monitor, multiMonitor string) error {
	m.SetWallpapersCalls = append(m.SetWallpapersCalls, struct {
		Paths   []string
		Monitors []core.Monitor
		Mode    string
	}{Paths: paths, Monitors: monitors, Mode: multiMonitor})
	return m.MockSetWallpapersError
}

// Override NewController to return our mock
var originalNewController = core.NewController
var originalNewWallpaperChanger = core.NewWallpaperChanger

func setupMocks(t *testing.T) (*MockController, *MockWallpaperChanger) {
	mockChanger := &MockWallpaperChanger{}
	mockController := &MockController{
		MockWallpaperChanger: mockChanger,
	}

	core.NewController = func(cfg *models.Config) *core.Controller {
		return &mockController.Controller
	}
	core.NewWallpaperChanger = func(desktopEnv string, respectDarkMode ...bool) *core.WallpaperChanger {
		return &mockChanger.WallpaperChanger
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
	originalSaveState := (*State).saveState
	(*State).saveState = func(s *State) error { return nil }

	t.Cleanup(func() {
		core.NewController = originalNewController
		core.NewWallpaperChanger = originalNewWallpaperChanger
		loadConfig = originalLoadConfig
		loadState = originalLoadState
		(*State).saveState = originalSaveState
		// Reset flags
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

	// Mock a wallpaper and download path
	testWallpaper := models.Wallpaper{ID: "test_id", URL: "http://example.com/test.jpg", Source: "test"}
	mockController.MockWallpaper = &testWallpaper
	mockController.MockDownloadPath = "/tmp/test.jpg"

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
	if len(call.Paths) != 1 || call.Paths[0] != mockController.MockDownloadPath {
		t.Errorf("SetWallpapers called with incorrect paths: %v", call.Paths)
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
	mockController.MockDownloadPath = "/tmp/test.jpg"

	// Mock monitors
	mockChanger.MockMonitors = []core.Monitor{
		{ID: "eDP-1", Name: "eDP-1", Primary: true},
	}

	// Set flags for the test
	setID = "test_id"
	setTargetMonitor = "invalid_monitor"

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Execute the command
	rootCmd.SetArgs([]string{"set", "--id", "test_id", "--target-monitor", "invalid_monitor"})
	err := rootCmd.Execute()

	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stderr = oldStderr
	stderrOutput := string(out)

	if err == nil {
		t.Fatal("Expected an error for invalid monitor, but got none.")
	}
	if !strings.Contains(stderrOutput, "monitor 'invalid_monitor' not found") {
		t.Errorf("Expected 'monitor not found' error, got: %s", stderrOutput)
	}
}

func TestSetDistinctRandomMultiMonitor(t *testing.T) {
	mockController, mockChanger := setupMocks(t)

	// Mock random wallpapers
	mockController.MockRandomWallpaper = models.Wallpaper{ID: "random_id_1", URL: "http://example.com/rand1.jpg", Source: "test"}
	mockController.MockDownloadPath = "/tmp/rand1.jpg" // This will be overwritten in the loop

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
