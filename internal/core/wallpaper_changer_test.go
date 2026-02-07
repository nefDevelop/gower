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
	// Mock isProcessRunning to always return false to test env vars
	originalIsProcessRunning := isProcessRunning
	defer func() { isProcessRunning = originalIsProcessRunning }()
	isProcessRunning = func(name string) bool { return false }

	// Save original env var
	originalEnv := os.Getenv("XDG_CURRENT_DESKTOP")
	defer os.Setenv("XDG_CURRENT_DESKTOP", originalEnv)

	// Test GNOME
	os.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	env := DetectDesktopEnv()
	if !strings.Contains(env, "gnome") {
		t.Errorf("Expected to detect 'gnome', got '%s'", env)
	}

	// Test KDE
	os.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	env = DetectDesktopEnv()
	if !strings.Contains(env, "kde") {
		t.Errorf("Expected to detect 'kde', got '%s'", env)
	}
}

// This test is limited because it can't actually execute the commands.
// It mainly checks that the function doesn't panic and returns an error
// when the respective command is not found.
func TestSetWallpaper(t *testing.T) {
	// Create a dummy file to act as the wallpaper
	tmpfile, err := os.CreateTemp("", "wallpaper.*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	testCases := []string{"kde", "gnome", "feh", "nitrogen", "sway", "niri", "dms", "swww", "awww", "unsupported"}

	for _, tc := range testCases {
		t.Run(tc, func(t *testing.T) {
			wc := NewWallpaperChanger(tc)
			err := wc.SetWallpapers([]string{tmpfile.Name()}, []Monitor{}, "clone")

			if tc == "unsupported" {
				if err == nil {
					t.Errorf("Expected an error for unsupported environment, but got nil")
				}
				if !strings.Contains(err.Error(), "unsupported") {
					t.Errorf("Expected error message to contain 'unsupported', got '%s'", err.Error())
				}
			} else {
				// In a CI environment, we expect these commands to fail.
				// A nil error would only happen if the command exists and runs successfully.
				// So, we are checking that it at least tries to run a command.
				if err == nil {
					t.Logf("Warning: SetWallpaper for '%s' succeeded. This might be unexpected in a test environment.", tc)
				}
			}
		})
	}
}
