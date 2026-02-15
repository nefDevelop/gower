//go:build !windows

package core

import "fmt"

// setWallpaperWindows es un stub para sistemas no-Windows.
func setWallpaperWindows(_ string) error {
	return fmt.Errorf("setWallpaperWindows is only supported on Windows")
}
