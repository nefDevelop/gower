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

func resetStatusFlags() {
	statusJSON = false
	statusProviders = false
	statusStorage = false
	statusDaemon = false
	statusSystem = false
	statusMonitors = false
	statusWallpaper = false
}

// Mock Controller for testing purposes
type MockController struct {
	core.Controller
	MockWallpapers map[string]*models.Wallpaper
}

func (m *MockController) GetWallpaper(id string) (*models.Wallpaper, error) {
	if wp, ok := m.MockWallpapers[id]; ok {
		return wp, nil
	}
	return nil, nil
}

// Override NewController to return our mock
var originalNewController = core.NewController

func setupStatusMocks(t *testing.T) (*MockController, func()) {
	// Setup temp home for config
	tmpDir, err := os.MkdirTemp("", "gower-test-status")
	if err != nil {
		t.Fatal(err)
	}
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	// Create config file
	configDir := filepath.Join(tmpDir, ".gower")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "config.json"), []byte("{}"), 0644)

	mockController := &MockController{
		MockWallpapers: make(map[string]*models.Wallpaper),
	}

	core.NewController = func(cfg *models.Config) *core.Controller {
		// We need a real controller for some parts, but override GetWallpaper
		realCtrl := originalNewController(cfg)
		mockController.Controller = *realCtrl
		return &mockController.Controller
	}

	// Mock loadConfig and loadState to avoid file system access during tests
	originalLoadConfig := loadConfig
	loadConfig = func() (*models.Config, error) {
		return &models.Config{}, nil
	}
	originalLoadState := loadState
	loadState = func() (*State, error) {
		return &State{}, nil
	}
	originalSaveState := saveState
	saveState = func(s *State) error { return nil }

	cleanup := func() {
		core.NewController = originalNewController
		loadConfig = originalLoadConfig
		loadState = originalLoadState
		saveState = originalSaveState
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tmpDir)
	}

	return mockController, cleanup
}

func TestStatusAll(t *testing.T) {
	resetStatusFlags()
	mockController, cleanup := setupStatusMocks(t)
	defer cleanup()

	// Manually create state.json for wallpaper status
	statePath := filepath.Join(os.Getenv("HOME"), ".gower", "state.json")
	stateData := `{"current_wallpaper_id": "wall_1", "current_wallpapers": ["wall_1", "wall_2"]}`
	os.WriteFile(statePath, []byte(stateData), 0644)

	// Add mock wallpapers
	mockController.MockWallpapers["wall_1"] = &models.Wallpaper{
		ID: "wall_1", Path: "/path/to/wall_1.jpg", Source: "test", URL: "http://example.com/wall_1.jpg", Dimension: "1920x1080", Color: "#FFFFFF", Theme: "light",
	}
	mockController.MockWallpapers["wall_2"] = &models.Wallpaper{
		ID: "wall_2", Path: "/path/to/wall_2.png", Source: "test", URL: "http://example.com/wall_2.png", Dimension: "2560x1440", Color: "#000000", Theme: "dark",
	}

	output, err := executeCommand(rootCmd, "status")
	if err != nil {
		t.Fatalf("Error executing status: %v", err)
	}

	if !strings.Contains(output, "--- System ---") {
		t.Errorf("Expected System section")
	}
	if !strings.Contains(output, "Desktop Environment") {
		t.Errorf("Expected 'Desktop Environment:' in output, got: %s", output)
	}
	if !strings.Contains(output, "--- Daemon ---") {
		t.Errorf("Expected Daemon section")
	}
	if !strings.Contains(output, "--- Providers ---") {
		t.Errorf("Expected Providers section")
	}
	if !strings.Contains(output, "--- Storage ---") {
		t.Errorf("Expected Storage section")
	}
	if !strings.Contains(output, "--- Wallpaper ---") {
		t.Errorf("Expected Wallpaper section")
	}
	if !strings.Contains(output, "Monitor 1 ID:\twall_1") {
		t.Errorf("Expected Monitor 1 ID, got: %s", output)
	}
	if !strings.Contains(output, "Monitor 1 Path:\t/path/to/wall_1.jpg") {
		t.Errorf("Expected Monitor 1 Path, got: %s", output)
	}
	if !strings.Contains(output, "Monitor 2 ID:\twall_2") {
		t.Errorf("Expected Monitor 2 ID, got: %s", output)
	}
	if !strings.Contains(output, "Monitor 2 Path:\t/path/to/wall_2.png") {
		t.Errorf("Expected Monitor 2 Path, got: %s", output)
	}
}

func TestStatusJSON(t *testing.T) {
	resetStatusFlags()
	mockController, cleanup := setupStatusMocks(t)
	defer cleanup()

	// Manually create state.json for wallpaper status
	statePath := filepath.Join(os.Getenv("HOME"), ".gower", "state.json")
	stateData := `{"current_wallpaper_id": "wall_1", "current_wallpapers": ["wall_1", "wall_2"]}`
	os.WriteFile(statePath, []byte(stateData), 0644)

	// Add mock wallpapers
	mockController.MockWallpapers["wall_1"] = &models.Wallpaper{
		ID: "wall_1", Path: "/path/to/wall_1.jpg", Source: "test", URL: "http://example.com/wall_1.jpg", Dimension: "1920x1080", Color: "#FFFFFF", Theme: "light",
	}
	mockController.MockWallpapers["wall_2"] = &models.Wallpaper{
		ID: "wall_2", Path: "/path/to/wall_2.png", Source: "test", URL: "http://example.com/wall_2.png", Dimension: "2560x1440", Color: "#000000", Theme: "dark",
	}

	output, err := executeCommand(rootCmd, "status", "--json")
	if err != nil {
		t.Fatalf("Error executing status --json: %v", err)
	}

	var statusOutput StatusOutput
	err = json.Unmarshal([]byte(output), &statusOutput)
	if err != nil {
		t.Fatalf("Error unmarshalling JSON output: %v", err)
	}

	if statusOutput.System == nil {
		t.Errorf("Expected JSON output containing 'system'")
	}
	if statusOutput.Wallpaper == nil || len(statusOutput.Wallpaper.Wallpapers) != 2 {
		t.Errorf("Expected JSON output containing 2 wallpapers, got %v", statusOutput.Wallpaper)
	}
	if statusOutput.Wallpaper.Wallpapers[0].ID != "wall_1" || statusOutput.Wallpaper.Wallpapers[0].Path != "/path/to/wall_1.jpg" {
		t.Errorf("Expected wall_1 details, got: %+v", statusOutput.Wallpaper.Wallpapers[0])
	}
	if statusOutput.Wallpaper.Wallpapers[1].ID != "wall_2" || statusOutput.Wallpaper.Wallpapers[1].Path != "/path/to/wall_2.png" {
		t.Errorf("Expected wall_2 details, got: %+v", statusOutput.Wallpaper.Wallpapers[1])
	}
}

func TestStatusFlags(t *testing.T) {
	resetStatusFlags()
	mockController, cleanup := setupStatusMocks(t)
	defer cleanup()

	// Manually create state.json for wallpaper status
	statePath := filepath.Join(os.Getenv("HOME"), ".gower", "state.json")
	stateData := `{"current_wallpaper_id": "wall_1", "current_wallpapers": ["wall_1"]}`
	os.WriteFile(statePath, []byte(stateData), 0644)

	// Add mock wallpaper
	mockController.MockWallpapers["wall_1"] = &models.Wallpaper{
		ID: "wall_1", Path: "/path/to/wall_1.jpg", Source: "test", URL: "http://example.com/wall_1.jpg", Dimension: "1920x1080", Color: "#FFFFFF", Theme: "light",
	}

	// Test --providers
	output, err := executeCommand(rootCmd, "status", "--providers")
	if err != nil {
		t.Fatalf("Error executing status --providers: %v", err)
	}
	if !strings.Contains(output, "--- Providers ---") {
		t.Errorf("Expected Providers section")
	}
	if strings.Contains(output, "--- System ---") {
		t.Errorf("Did not expect System section")
	}
	if strings.Contains(output, "--- Wallpaper ---") {
		t.Errorf("Did not expect Wallpaper section")
	}

	// Test --storage
	// Create some dummy file to check size
	cacheDir := filepath.Join(os.Getenv("HOME"), ".gower", "cache")
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(filepath.Join(cacheDir, "test"), []byte("test"), 0644)

	resetStatusFlags() // Reset again to clear providers flag
	output, err = executeCommand(rootCmd, "status", "--storage")
	if err != nil {
		t.Fatalf("Error executing status --storage: %v", err)
	}
	if !strings.Contains(output, "--- Storage ---") {
		t.Errorf("Expected Storage section")
	}
	if strings.Contains(output, "--- Wallpaper ---") {
		t.Errorf("Did not expect Wallpaper section")
	}

	// Test --wallpapers
	resetStatusFlags()
	output, err = executeCommand(rootCmd, "status", "--wallpapers")
	if err != nil {
		t.Fatalf("Error executing status --wallpapers: %v", err)
	}
	if !strings.Contains(output, "--- Wallpaper ---") {
		t.Errorf("Expected Wallpaper section")
	}
	if !strings.Contains(output, "Monitor 1 ID:\twall_1") {
		t.Errorf("Expected Monitor 1 ID, got: %s", output)
	}
	if !strings.Contains(output, "Monitor 1 Path:\t/path/to/wall_1.jpg") {
		t.Errorf("Expected Monitor 1 Path, got: %s", output)
	}
	if strings.Contains(output, "--- System ---") {
		t.Errorf("Did not expect System section")
	}
}

func TestStatusWallpaperNoWallpapers(t *testing.T) {
	resetStatusFlags()
	_, cleanup := setupStatusMocks(t)
	defer cleanup()

	// No state.json or empty state.json
	statePath := filepath.Join(os.Getenv("HOME"), ".gower", "state.json")
	os.WriteFile(statePath, []byte(`{}`), 0644)

	output, err := executeCommand(rootCmd, "status", "--wallpapers")
	if err != nil {
		t.Fatalf("Error executing status --wallpapers: %v", err)
	}

	if !strings.Contains(output, "--- Wallpaper ---") {
		t.Errorf("Expected Wallpaper section")
	}
	if !strings.Contains(output, "No wallpapers currently set.") {
		t.Errorf("Expected 'No wallpapers currently set.' message, got: %s", output)
	}
	if strings.Contains(output, "Monitor 1 ID:") {
		t.Errorf("Did not expect wallpaper details when none are set, got: %s", output)
	}

	// Test JSON output for no wallpapers
	output, err = executeCommand(rootCmd, "status", "--wallpapers", "--json")
	if err != nil {
		t.Fatalf("Error executing status --wallpapers --json: %v", err)
	}

	var statusOutput StatusOutput
	err = json.Unmarshal([]byte(output), &statusOutput)
	if err != nil {
		t.Fatalf("Error unmarshalling JSON output: %v", err)
	}

	if statusOutput.Wallpaper == nil || len(statusOutput.Wallpaper.Wallpapers) != 0 {
		t.Errorf("Expected empty wallpapers array in JSON, got: %+v", statusOutput.Wallpaper)
	}
}
