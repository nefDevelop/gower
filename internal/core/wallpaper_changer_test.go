package core

import (
	"os"
	"strings"
	"testing"
)

func TestNewWallpaperChanger_Manual(t *testing.T) {
	wc := NewWallpaperChanger("kde")
	if wc.Env != "kde" {
		t.Errorf("Expected Env to be 'kde', got '%s'", wc.Env)
	}

	wc = NewWallpaperChanger("gnome")
	if wc.Env != "gnome" {
		t.Errorf("Expected Env to be 'gnome', got '%s'", wc.Env)
	}

	// Test normalization
	wc = NewWallpaperChanger("KDE Plasma")
	if wc.Env != "kde" {
		t.Errorf("Expected Env to be normalized to 'kde', got '%s'", wc.Env)
	}
}

func TestDetectDesktopEnv(t *testing.T) {
	// Mock isProcessRunning
	originalIsProcessRunning := isProcessRunning
	defer func() { isProcessRunning = originalIsProcessRunning }()
	isProcessRunning = func(name string) bool { return false }

	// Mock commandExists
	originalCommandExists := commandExists
	defer func() { commandExists = originalCommandExists }()
	commandExists = func(name string) bool { return false }

	// Save original env var
	originalEnv := os.Getenv("XDG_CURRENT_DESKTOP")
	defer func() { _ = os.Setenv("XDG_CURRENT_DESKTOP", originalEnv) }()

	// Test GNOME
	_ = os.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	env := DetectDesktopEnv()
	if !strings.Contains(env, "gnome") {
		t.Errorf("Expected to detect 'gnome', got '%s'", env)
	}

	// Test KDE
	_ = os.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	env = DetectDesktopEnv()
	if !strings.Contains(env, "kde") {
		t.Errorf("Expected to detect 'kde', got '%s'", env)
	}
}

func TestSetWallpaper(t *testing.T) {
	// Mock isProcessRunning and commandExists to avoid running actual commands
	originalIsProcessRunning := isProcessRunning
	originalCommandExists := commandExists
	defer func() {
		isProcessRunning = originalIsProcessRunning
		commandExists = originalCommandExists
	}()
	isProcessRunning = func(name string) bool { return false }
	commandExists = func(name string) bool { return true } // Pretend all commands exist

	// Create a dummy file to act as the wallpaper
	tmpfile, err := os.CreateTemp("", "wallpaper.*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	_ = tmpfile.Close()

	testCases := []string{"kde", "gnome", "feh", "nitrogen", "sway", "niri", "dms", "swww", "awww", "unsupported"}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			wc := NewWallpaperChanger(tc)
			// Mock SetWallpapersFunc to avoid actual command execution while still testing logic flow
			wc.SetWallpapersFunc = func(paths []string, monitors []Monitor, multiMonitor string) error {
				if tc == "unsupported" {
					return os.ErrInvalid // Simulate error for unsupported
				}
				return nil
			}

			err := wc.SetWallpapers([]string{tmpfile.Name()}, []Monitor{}, "clone")

			if tc == "unsupported" {
				if err == nil {
					t.Errorf("Expected an error for unsupported environment, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("SetWallpaper for '%s' failed unexpectedly: %v", tc, err)
				}
			}
		})
	}
}

func TestIsSystemInDarkMode(t *testing.T) {
	// Mock commandExists
	originalCommandExists := commandExists
	defer func() { commandExists = originalCommandExists }()

	// Test with no tools available (should be false)
	commandExists = func(name string) bool { return false }
	if IsSystemInDarkMode() {
		t.Error("Expected false when no tools are available")
	}

	// Test with environment variable
	t.Setenv("GTK_THEME", "Adwaita-dark")
	if !IsSystemInDarkMode() {
		t.Error("Expected true when GTK_THEME contains 'dark'")
	}
}

func TestFilterXWaylandMonitors(t *testing.T) {
	tests := []struct {
		name     string
		input    []Monitor
		expected int
	}{
		{
			name: "Mixed monitors",
			input: []Monitor{
				{Name: "DP-1"},
				{Name: "HDMI-1"},
				{Name: "XWAYLAND0"},
			},
			expected: 2, // Should remove XWAYLAND0
		},
		{
			name: "Only real monitors",
			input: []Monitor{
				{Name: "eDP-1"},
			},
			expected: 1,
		},
		{
			name: "Only XWayland (fallback)",
			input: []Monitor{
				{Name: "XWAYLAND0"},
			},
			expected: 1, // Should keep it if it's the only one
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterXWaylandMonitors(tt.input)
			if len(result) != tt.expected {
				t.Errorf("Expected %d monitors, got %d", tt.expected, len(result))
			}
			for _, m := range result {
				if len(result) > 1 && strings.HasPrefix(m.Name, "XWAYLAND") {
					t.Errorf("Result contains XWAYLAND monitor when real monitors exist")
				}
			}
		})
	}
}
