// internal/core/wallpaper_changer.go
package core

import (
	"os"
	"os/exec"
	"strings"
)

type WallpaperChanger struct {
	Command string
	Args    []string
}

func NewWallpaperChanger(desktopEnv string) *WallpaperChanger {
	var cmd string
	var args []string

	switch {
	case strings.Contains(strings.ToLower(desktopEnv), "kde"):
		cmd = "dbus-send"
		args = []string{"--session", "--dest=org.kde.plasmashell",
			"--type=method_call", "/PlasmaShell",
			"org.kde.PlasmaShell.evaluateScript",
			"string:...KDE script..."}
	case strings.Contains(strings.ToLower(desktopEnv), "gnome"):
		cmd = "gsettings"
		args = []string{"set", "org.gnome.desktop.background",
			"picture-uri", "file://"}
	case commandExists("feh"):
		cmd = "feh"
		args = []string{"--bg-fill"}
	case commandExists("nitrogen"):
		cmd = "nitrogen"
		args = []string{"--set-auto"}
	default:
		// Intentar detectar automáticamente
		cmd = detectWallpaperCommand()
	}

	return &WallpaperChanger{Command: cmd, Args: args}
}

func (wc *WallpaperChanger) SetWallpaper(path string, multiMonitor string) error {
	var finalArgs []string

	// Copiar args base
	finalArgs = append(finalArgs, wc.Args...)

	// Manejar multimonitor
	if multiMonitor == "distinct" && wc.Command == "feh" {
		finalArgs = append(finalArgs, "--no-fehbg")
		// Lógica para múltiples wallpapers
	}

	finalArgs = append(finalArgs, path)

	cmd := exec.Command(wc.Command, finalArgs...)
	return cmd.Run()
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func detectWallpaperCommand() string {
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	if strings.Contains(desktop, "gnome") {
		return "gsettings"
	}
	if strings.Contains(desktop, "kde") {
		return "dbus-send"
	}
	return ""
}
