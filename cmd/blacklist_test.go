package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBlacklistAdd(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer func() { _ = os.RemoveAll(tmpDir) }() // No need to check error in test cleanup

	executeCommand(rootCmd, "config", "init")

	// Add to blacklist
	output, err := executeCommand(rootCmd, "blacklist", "add", "bad-id")
	if err != nil {
		t.Fatalf("Error executing blacklist: %v", err)
	}

	if !strings.Contains(output, "Wallpaper bad-id added to blacklist.") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify file content
	blacklistPath := filepath.Join(tmpDir, ".config", "gower", "data", "blacklist.json")
	content, err := os.ReadFile(blacklistPath)
	if err != nil {
		t.Fatalf("Error reading blacklist file: %v", err)
	}
	if !strings.Contains(string(content), "bad-id") {
		t.Errorf("Blacklist file does not contain bad-id")
	}
}
