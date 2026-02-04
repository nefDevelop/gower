package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gower/internal/core"
	"gower/pkg/models"
)

func resetDownloadFlags() {
	downloadOutput = ""
	downloadRandom = false
	downloadTheme = ""
	downloadFromFavorites = false
	downloadTag = false
}

func TestDownloadCommand(t *testing.T) {
	resetDownloadFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake image content"))
	}))
	defer server.Close()

	// Setup config and feed
	executeCommand(rootCmd, "config", "init")
	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	ctrl.AddWallpaperToFeed(models.Wallpaper{
		ID:     "test-dl",
		URL:    server.URL + "/img.jpg",
		Source: "test",
		Theme:  "dark",
	})

	// Test download by ID
	output, err := executeCommand(rootCmd, "download", "test-dl")
	if err != nil {
		t.Fatalf("Error executing download: %v", err)
	}
	if !strings.Contains(output, "Downloaded to cache") {
		t.Errorf("Expected download success message, got: %s", output)
	}

	// Test download with output
	outFile := filepath.Join(tmpDir, "my_wallpaper.jpg")
	resetDownloadFlags()
	output, err = executeCommand(rootCmd, "download", "test-dl", "--output", outFile)
	if err != nil {
		t.Fatalf("Error executing download with output: %v", err)
	}
	if !strings.Contains(output, "Saved to:") {
		t.Errorf("Expected saved to message, got: %s", output)
	}
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Errorf("Output file not created at %s", outFile)
	}

	// Test download with tag
	resetDownloadFlags()
	output, err = executeCommand(rootCmd, "download", "test-dl", "--output", tmpDir, "--tag")
	if err != nil {
		t.Fatalf("Error executing download with tag: %v", err)
	}

	matches, _ := filepath.Glob(filepath.Join(tmpDir, "*[d]*"))
	if len(matches) == 0 {
		t.Errorf("Expected file with tag [d] in %s", tmpDir)
	}
}
