package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheCleanCmd(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "gower-cache-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the user home directory to the temporary directory
	os.Setenv("HOME", tmpDir)

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
	dummyFile.Close()

	// Execute the command
	cacheCleanCmd.Run(cacheCleanCmd, []string{})

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
	os.Setenv("HOME", tmpDir)

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
	dummyFile.Close()

	// Capture the output
	var buf bytes.Buffer
	cacheSizeCmd.SetOut(&buf)

	// Execute the command
	cacheSizeCmd.Run(cacheSizeCmd, []string{})

	// Check the output
	expectedSize := "Cache size: 1.00 MB"
	if !strings.Contains(buf.String(), expectedSize) {
		t.Errorf("Expected output to contain '%s', but got '%s'", expectedSize, buf.String())
	}
}
