package cmd

import (
	"gower/internal/core"
	"io"
	"os"
	"strings"
	"testing"
)

// Mock DetectMonitors for testing purposes
type MockWallpaperChanger struct {
	core.WallpaperChanger
	MockMonitors []core.Monitor
	MockError    error
}

func (m *MockWallpaperChanger) DetectMonitors() ([]core.Monitor, error) {
	return m.MockMonitors, m.MockError
}

func TestStatusMonitors(t *testing.T) {
	setupTestEnv(t)
	// Save original os.Stdout and restore it after test
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Ensure cobra writes to our pipe
	originalOut := rootCmd.OutOrStdout()
	originalErr := rootCmd.ErrOrStderr()
	rootCmd.SetOut(w)
	rootCmd.SetErr(w)

	// Create a mock changer
	mockChanger := &MockWallpaperChanger{
		MockMonitors: []core.Monitor{
			{ID: "eDP-1", Name: "eDP-1", Width: 1920, Height: 1080, X: 0, Y: 0, Primary: true},
			{ID: "DP-1", Name: "DP-1", Width: 1920, Height: 1080, X: 1920, Y: 0, Primary: false},
		},
		MockError: nil,
	}

	// Temporarily replace core.NewWallpaperChanger to return our mock
	originalNewWallpaperChanger := core.NewWallpaperChanger
	core.NewWallpaperChanger = func(desktopEnv string, respectDarkMode ...bool) *core.WallpaperChanger {
		wc := &core.WallpaperChanger{Env: desktopEnv}
		wc.DetectMonitorsFunc = func() ([]core.Monitor, error) {
			return mockChanger.MockMonitors, mockChanger.MockError
		}
		return wc
	}
	defer func() { core.NewWallpaperChanger = originalNewWallpaperChanger }()

	// Set flags for the test
	statusMonitors = true
	statusJSON = false // Ensure text output

	// Execute the command
	rootCmd.SetArgs([]string{"status", "--monitors"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("status command failed: %v", err)
	}

	// Close the write end of the pipe
	_ = w.Close()
	// Read all output from the read end
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout // Restore original Stdout
	rootCmd.SetOut(originalOut)
	rootCmd.SetErr(originalErr)

	output := string(out)

	if !strings.Contains(output, "--- Monitors ---") {
		t.Errorf("Output missing '--- Monitors ---' section.\nOutput: %s", output)
	}
	if !strings.Contains(output, "Monitor 1: eDP-1 (Primary)") {
		t.Errorf("Output missing primary monitor info.\nOutput: %s", output)
	}
	if !strings.Contains(output, "Resolution") || !strings.Contains(output, "1920x1080") {
		t.Errorf("Output missing resolution info.\nOutput: %s", output)
	}
	if !strings.Contains(output, "Monitor 2: DP-1") {
		t.Errorf("Output missing secondary monitor info.\nOutput: %s", output)
	}
}

func TestStatusMonitorsJSON(t *testing.T) {
	setupTestEnv(t)
	// Save original os.Stdout and restore it after test
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Ensure cobra writes to our pipe
	originalOut := rootCmd.OutOrStdout()
	originalErr := rootCmd.ErrOrStderr()
	rootCmd.SetOut(w)
	rootCmd.SetErr(w)

	// Create a mock changer
	mockChanger := &MockWallpaperChanger{
		MockMonitors: []core.Monitor{
			{ID: "eDP-1", Name: "eDP-1", Width: 1920, Height: 1080, X: 0, Y: 0, Primary: true},
		},
		MockError: nil,
	}

	// Temporarily replace core.NewWallpaperChanger to return our mock
	originalNewWallpaperChanger := core.NewWallpaperChanger
	core.NewWallpaperChanger = func(desktopEnv string, respectDarkMode ...bool) *core.WallpaperChanger {
		wc := &core.WallpaperChanger{Env: desktopEnv}
		wc.DetectMonitorsFunc = func() ([]core.Monitor, error) {
			return mockChanger.MockMonitors, mockChanger.MockError
		}
		return wc
	}
	defer func() { core.NewWallpaperChanger = originalNewWallpaperChanger }()

	// Set flags for the test
	statusMonitors = true
	statusJSON = true // Ensure JSON output

	// Execute the command
	rootCmd.SetArgs([]string{"status", "--monitors", "--json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("status command failed: %v", err)
	}

	// Close the write end of the pipe
	_ = w.Close()
	// Read all output from the read end
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout // Restore original Stdout
	rootCmd.SetOut(originalOut)
	rootCmd.SetErr(originalErr)

	output := string(out)

	if !strings.Contains(output, "\"monitors\": [") {
		t.Errorf("JSON output missing 'monitors' array.\nOutput: %s", output)
	}
	if !strings.Contains(output, "\"ID\": \"eDP-1\"") {
		t.Errorf("JSON output missing monitor ID.\nOutput: %s", output)
	}
	if !strings.Contains(output, "\"Primary\": true") {
		t.Errorf("JSON output missing primary status.\nOutput: %s", output)
	}
}
