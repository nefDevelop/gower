package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gower/internal/core"
	"gower/pkg/models"
)

func TestCacheCleanCmd(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "gower-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the user home directory to the temporary directory
	// Use t.Setenv for automatic restoration
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir) // for Windows

	// Create some dummy files and directories in the cache
	cacheDir := filepath.Join(tmpDir, ".gower", "cache")
	wallpapersDir := filepath.Join(cacheDir, "wallpapers")
	thumbsDir := filepath.Join(cacheDir, "thumbs")
	os.MkdirAll(wallpapersDir, 0755)
	os.MkdirAll(thumbsDir, 0755)

	dummyFile, err := os.Create(filepath.Join(wallpapersDir, "dummy.jpg"))
	if err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}
	_ = dummyFile.Close()

	// Execute the command
	_, err = executeCommand(rootCmd, "system", "cache", "clean")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Check if the dummy file is gone
	_, err = os.Stat(filepath.Join(wallpapersDir, "dummy.jpg"))
	if !os.IsNotExist(err) {
		t.Errorf("Expected dummy file to be removed, but it still exists")
	}

	// Check if the directories are recreated
	_, err = os.Stat(wallpapersDir)
	if os.IsNotExist(err) {
		t.Errorf("Expected wallpapers directory to be recreated, but it's missing")
	}
	_, err = os.Stat(thumbsDir)
	if os.IsNotExist(err) {
		t.Errorf("Expected thumbs directory to be recreated, but it's missing")
	}
}

func TestCacheSizeCmd(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "gower-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the user home directory to the temporary directory
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir) // for Windows

	// Create a dummy file with a known size
	cacheDir := filepath.Join(tmpDir, ".gower", "cache")
	wallpapersDir := filepath.Join(cacheDir, "wallpapers")
	os.MkdirAll(wallpapersDir, 0755)

	dummyFile, err := os.Create(filepath.Join(wallpapersDir, "dummy.jpg"))
	if err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}
	// Write 1MB of data
	oneMB := make([]byte, 1024*1024)
	_, err = dummyFile.Write(oneMB)
	if err != nil {
		t.Fatalf("Failed to write to dummy file: %v", err)
	}
	_ = dummyFile.Close()

	// Execute the command
	output, err := executeCommand(rootCmd, "system", "cache", "size")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Check the output
	expectedSize := "Cache size: 1.00 MB"
	if !strings.Contains(output, expectedSize) {
		t.Errorf("Expected output to contain '%s', but got '%s'", expectedSize, output)
	}
}

func TestCachePruneCmd(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Setup:
	// - 1 wallpaper in feed, with a cached file
	// - 1 orphaned wallpaper file
	// - 1 orphaned thumbnail file

	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	wpInFeed := models.Wallpaper{ID: "keep_me", URL: "http://example.com/keep.jpg"}
	_ = ctrl.AddWallpaperToFeed(wpInFeed)

	appDir, _ := core.GetAppDir()
	wallpapersDir := filepath.Join(appDir, "cache", "wallpapers")
	thumbsDir := filepath.Join(appDir, "cache", "thumbs")
	// Create files
	_ = os.MkdirAll(wallpapersDir, 0755)
	_ = os.MkdirAll(thumbsDir, 0755)

	// Create files
	_ = os.WriteFile(filepath.Join(wallpapersDir, "keep_me.jpg"), []byte("data"), 0644)
	_ = os.WriteFile(filepath.Join(wallpapersDir, "delete_me.jpg"), []byte("data"), 0644)
	_ = os.WriteFile(filepath.Join(thumbsDir, "keep_me.jpg"), []byte("data"), 0644)
	_ = os.WriteFile(filepath.Join(thumbsDir, "delete_me_thumb.jpg"), []byte("data"), 0644)

	// Execute prune
	output, err := executeCommand(rootCmd, "system", "cache", "prune")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if !strings.Contains(output, "Pruning complete. Removed 2 orphaned file(s).") {
		t.Errorf("Expected success message with 2 files removed, got: %s", output)
	}

	// Verify files
	if _, err := os.Stat(filepath.Join(wallpapersDir, "keep_me.jpg")); os.IsNotExist(err) {
		t.Error("Expected keep_me.jpg to exist, but it was deleted")
	}
	if _, err := os.Stat(filepath.Join(wallpapersDir, "delete_me.jpg")); !os.IsNotExist(err) {
		t.Error("Expected delete_me.jpg to be deleted, but it exists")
	}
	if _, err := os.Stat(filepath.Join(thumbsDir, "keep_me.jpg")); os.IsNotExist(err) {
		t.Error("Expected keep_me.jpg thumb to exist, but it was deleted")
	}
	if _, err := os.Stat(filepath.Join(thumbsDir, "delete_me_thumb.jpg")); !os.IsNotExist(err) {
		t.Error("Expected delete_me_thumb.jpg to be deleted, but it exists")
	}
}
