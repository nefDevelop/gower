// internal/core/wallpaper_changer.go
package core

import (
	"fmt"
	"gower/internal/utils"
	"os"
	"os/exec"
	"strings"
)

type WallpaperChanger struct {
	Env string
}

func NewWallpaperChanger(desktopEnv string) *WallpaperChanger {
	env := strings.ToLower(desktopEnv)
	if env == "" {
		env = detectDesktopEnv()
	} else {
		// Normalizar entrada manual
		if strings.Contains(env, "kde") {
			env = "kde"
		} else if strings.Contains(env, "gnome") {
			env = "gnome"
		}
	}
	return &WallpaperChanger{Env: env}
}

func (wc *WallpaperChanger) SetWallpaper(path string, multiMonitor string) error {
	utils.Log.Info("Setting wallpaper: %s (Env: %s)", path, wc.Env)
	var cmd *exec.Cmd

	switch wc.Env {
	case "kde":
		script := fmt.Sprintf(`
			var allDesktops = desktops();
			for (i=0;i<allDesktops.length;i++) {
				d = allDesktops[i];
				d.wallpaperPlugin = "org.kde.image";
				d.currentConfigGroup = Array("Wallpaper", "org.kde.image", "General");
				d.writeConfig("Image", "file://%s");
			}
		`, path)
		cmd = exec.Command("dbus-send", "--session", "--dest=org.kde.plasmashell",
			"--type=method_call", "/PlasmaShell",
			"org.kde.PlasmaShell.evaluateScript",
			"string:"+script)

	case "gnome":
		uri := "file://" + path
		// Establecer tanto para modo claro como oscuro en versiones modernas de Gnome
		exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri).Run()
		cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri)

	case "sway":
		// swaybg is a good generic for sway-like compositors (like niri)
		// Using -m fill to replicate --bg-fill behavior from feh
		// Note: swaybg forks, so we don't wait for it to exit.
		cmd = exec.Command("swaybg", "-i", path, "-m", "fill")

	case "feh":
		args := []string{"--bg-fill", path}
		if multiMonitor == "distinct" {
			// Nota: Para distinct real se necesitarían múltiples rutas, aquí asumimos clonado/fill
			args = []string{"--no-fehbg", "--bg-fill", path}
		}
		cmd = exec.Command("feh", args...)

	case "nitrogen":
		cmd = exec.Command("nitrogen", "--set-auto", "--save", path)

	case "dms":
		cmd = exec.Command("dms", "ipc", "call", "wallpaper", "set", path)

	case "swww":
		cmd = exec.Command("swww", "img", path)

	case "awww":
		cmd = exec.Command("awww", path)

	case "test":
		return nil

	default:
		return fmt.Errorf("unsupported or undetected desktop environment: %s", wc.Env)
	}

	return cmd.Run()
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func detectDesktopEnv() string {
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	if desktop == "test" {
		return "test"
	}
	if strings.Contains(desktop, "gnome") {
		return "gnome"
	}
	if strings.Contains(desktop, "kde") {
		return "kde"
	}
	// Check for sway/niri before falling back to generic X11 tools
	if (strings.Contains(desktop, "sway") || strings.Contains(desktop, "niri")) && commandExists("swaybg") {
		return "sway"
	}
	// Other popular Wayland wallpaper tools
	if commandExists("swww") {
		return "swww"
	}
	if commandExists("awww") {
		return "awww"
	}

	if commandExists("dms") {
		return "dms"
	}
	if commandExists("feh") {
		return "feh"
	}
	if commandExists("nitrogen") {
		return "nitrogen"
	}
	// Last resort check for swaybg
	if commandExists("swaybg") {
		return "sway"
	}
	return "unknown"
}
