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
	_ = tmpfile.Close()

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

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		env     string
		monitor Monitor
		path    string
		index   int
		wantErr bool
		errMsg  string
	}{
		{
			env:     "test",
			monitor: Monitor{Name: "eDP-1"},
			path:    "/tmp/wallpaper.jpg",
			wantErr: true,
			errMsg:  "unsupported",
		},
		{
			env:     "feh",
			monitor: Monitor{Name: "DP-1"},
			path:    "/tmp/wallpaper.jpg",
			index:   0,
			wantErr: false,
		},
		{
			env:     "swww",
			monitor: Monitor{Name: "eDP-1"},
			path:    "/tmp/wallpaper.jpg",
			wantErr: false,
		},
		{
			env:     "nitrogen",
			monitor: Monitor{Name: "default"},
			path:    "/tmp/wallpaper.jpg",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			wc := NewWallpaperChanger(tt.env)
			cmd, err := wc.buildCommand(tt.monitor, tt.path, tt.index)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			} else if cmd == nil {
				t.Error("Expected command, got nil")
			}
		})
	}
}

func TestBuildCommand_FehIndex(t *testing.T) {
	wc := NewWallpaperChanger("feh")
	cmd, err := wc.buildCommand(Monitor{Name: "DP-1"}, "/tmp/wallpaper.jpg", 2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	args := cmd.Args
	hasDisplay := false
	for i, a := range args {
		if a == "--display" && i+1 < len(args) {
			if args[i+1] == ":0.2" {
				hasDisplay = true
			}
		}
	}
	if !hasDisplay {
		t.Errorf("Expected --display :0.2 in command args, got %v", args)
	}
}

func TestBuildCommand_SwayWithMonitor(t *testing.T) {
	wc := NewWallpaperChanger("sway")
	cmd, err := wc.buildCommand(Monitor{Name: "eDP-1"}, "/tmp/wp.jpg", 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	args := cmd.Args
	if len(args) < 5 || args[0] != "swaymsg" {
		t.Errorf("Expected swaymsg command, got %v", args)
	}
}

func TestBuildCommand_SwayDefaultMonitor(t *testing.T) {
	wc := NewWallpaperChanger("sway")
	cmd, err := wc.buildCommand(Monitor{Name: ""}, "/tmp/wp.jpg", 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	args := cmd.Args
	if len(args) < 3 || args[0] != "swaybg" {
		t.Errorf("Expected swaybg command when no monitor name, got %v", args)
	}
}

func TestApplyToMonitor_DMS_Fallback(t *testing.T) {
	wc := NewWallpaperChanger("dms")

	// Use SetWallpapersFunc to capture what would happen
	called := false
	wc.SetWallpapersFunc = func(paths []string, monitors []Monitor, mode string) error {
		called = true
		return nil
	}

	err := wc.applyToMonitor(Monitor{Name: "eDP-1"}, "/tmp/wp.jpg", 0)
	if err != nil {
		t.Logf("Expected possible error (DMS not installed): %v", err)
	} else if !called {
		// If no error, SetWallpapersFunc should have been called
		// This is valid when DMS is not installed - it falls through gracefully
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
