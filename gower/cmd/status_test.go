package cmd

import (
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

func TestStatusAll(t *testing.T) {
	resetStatusFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	output, err := executeCommand(rootCmd, "status")
	if err != nil {
		t.Fatalf("Error executing status: %v", err)
	}

	if !strings.Contains(output, "--- System ---") {
		t.Errorf("Expected System section")
	}
	if !strings.Contains(output, "Desktop Environment:") {
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
}

func TestStatusJSON(t *testing.T) {
	resetStatusFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	output, err := executeCommand(rootCmd, "status", "--json")
	if err != nil {
		t.Fatalf("Error executing status --json: %v", err)
	}

	if !strings.Contains(output, "\"system\":") {
		t.Errorf("Expected JSON output containing 'system'")
	}
	if !strings.Contains(output, "\"desktop_env\":") {
		t.Errorf("Expected JSON output containing 'desktop_env'")
	}
	if !strings.Contains(output, "\"os\":") {
		t.Errorf("Expected JSON output containing 'os'")
	}
}

func TestStatusFlags(t *testing.T) {
	resetStatusFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

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

	// Test --storage
	// Create some dummy file to check size
	cacheDir := filepath.Join(tmpDir, ".config", "gower", "cache")
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
}

func TestStatusWallpaper(t *testing.T) {
	resetStatusFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Manually create state.json
	statePath := filepath.Join(tmpDir, ".config", "gower", "state.json")
	stateData := `{"current_wallpaper_id": "wall_1", "current_wallpapers": ["wall_1", "wall_2"]}`
	os.WriteFile(statePath, []byte(stateData), 0644)

	output, err := executeCommand(rootCmd, "status")
	if err != nil {
		t.Fatalf("Error executing status: %v", err)
	}

	if !strings.Contains(output, "--- Wallpaper ---") {
		t.Errorf("Expected Wallpaper section")
	}
	if !strings.Contains(output, "Monitor 1: wall_1") {
		t.Errorf("Expected Monitor 1: wall_1, got: %s", output)
	}
	if !strings.Contains(output, "Monitor 2: wall_2") {
		t.Errorf("Expected Monitor 2: wall_2, got: %s", output)
	}
}
