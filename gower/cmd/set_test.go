package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func resetSetFlags() {
	setID = ""
	setURL = ""
	setRandom = false
	setTheme = ""
	setFromFavorites = false
	setMultiMonitor = ""
	setCommand = ""
	setTargetMonitor = ""
}

func TestController_GetWallpaperAndDownload(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("image"))
	}))
	defer server.Close()

	cfg := &models.Config{}
	ctrl := core.NewController(cfg)
	wp := models.Wallpaper{ID: "test-1", URL: server.URL + "/img.jpg"}
	ctrl.AddWallpaperToFeed(wp)

	// Test GetWallpaper
	got, err := ctrl.GetWallpaper("test-1")
	assert.NoError(t, err)
	assert.Equal(t, wp.URL, got.URL)

	// Test DownloadWallpaper
	path, err := ctrl.DownloadWallpaper(*got)
	assert.NoError(t, err)
	_, err = os.Stat(path)
	assert.NoError(t, err, "Downloaded file should exist at %s", path)
}

func TestSetUndoCommand(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "test")
	resetSetFlags()
	_, cleanup := setupTestHomeWithState(t, &State{
		CurrentWallpaperID:  "current-wp",
		PreviousWallpaperID: "previous-wp",
		PreviousWallpapers:  []string{"previous-wp", "previous-wp-2"},
	})
	defer cleanup()

	// Mock server for image download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake image content"))
	}))
	defer server.Close()

	// Populate feed with the wallpaper we expect to be set
	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	ctrl.AddWallpaperToFeed(models.Wallpaper{
		ID:     "previous-wp",
		URL:    server.URL + "/image.jpg",
		Source: "test",
	})
	ctrl.AddWallpaperToFeed(models.Wallpaper{
		ID:     "previous-wp-2",
		URL:    server.URL + "/image2.jpg",
		Source: "test",
	})

	// Execute the undo command and capture output
	// We need to re-initialize the root command for each test run to avoid state leakage
	testRootCmd, _, _ := newTestRootCmd()
	output, err := executeCommand(testRootCmd, "set", "undo")

	assert.NoError(t, err)
	assert.Contains(t, output, "Wallpaper(s) set successfully")
	assert.Contains(t, output, "Preparing wallpaper: previous-wp")
	assert.Contains(t, output, "Preparing wallpaper: previous-wp-2")
}

// setupTestHomeWithState is a helper for tests that need a pre-configured state.json
func setupTestHomeWithState(t *testing.T, state *State) (string, func()) {
	tempDir, err := os.MkdirTemp("", "gower-test-home-")
	assert.NoError(t, err)

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	// Create .gower dir and write state
	gowerDir := filepath.Join(tempDir, ".gower")
	err = os.MkdirAll(gowerDir, 0755)
	assert.NoError(t, err)

	statePath := filepath.Join(gowerDir, "state.json")
	stateData, err := json.Marshal(state)
	assert.NoError(t, err)
	err = os.WriteFile(statePath, stateData, 0644)
	assert.NoError(t, err)

	// Create a dummy config to satisfy ensureConfig
	configPath := filepath.Join(gowerDir, "config.json")
	err = os.WriteFile(configPath, []byte("{}"), 0644)
	assert.NoError(t, err)

	cleanup := func() {
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// newTestRootCmd creates a fresh instance of the root command for isolated testing.
func newTestRootCmd() (*cobra.Command, *CLIConfig, *bytes.Buffer) {
	rootCmd := &cobra.Command{Use: "gower"}
	var cfg CLIConfig
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)

	// Add all commands to the new root
	rootCmd.AddCommand(setCmd)
	rootCmd.AddCommand(exploreCmd)
	rootCmd.AddCommand(configCmd)

	// Reset output of subcommands to ensure they inherit from rootCmd
	setCmd.SetOut(nil)
	setCmd.SetErr(nil)
	exploreCmd.SetOut(nil)
	exploreCmd.SetErr(nil)
	configCmd.SetOut(nil)
	configCmd.SetErr(nil)

	// Re-initialize flags for subcommands if necessary
	// This is a simplified setup. A full setup would re-run all init() functions
	// or use a factory pattern for commands.
	return rootCmd, &cfg, &out
}
