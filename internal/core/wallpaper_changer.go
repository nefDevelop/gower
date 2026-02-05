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
	Env             string
	RespectDarkMode bool
}

func NewWallpaperChanger(desktopEnv string, respectDarkMode ...bool) *WallpaperChanger {
	env := strings.ToLower(desktopEnv)
	if env == "" {
		env = DetectDesktopEnv()
	} else {
		// Normalizar entrada manual
		if strings.Contains(env, "kde") {
			env = "kde"
		} else if strings.Contains(env, "gnome") {
			env = "gnome"
		}
	}
	respect := true
	if len(respectDarkMode) > 0 {
		respect = respectDarkMode[0]
	}
	return &WallpaperChanger{Env: env, RespectDarkMode: respect}
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
		if wc.RespectDarkMode {
			// Si respetamos el modo oscuro, solo establecemos la clave correspondiente al modo actual
			// para no sobrescribir la configuración del otro modo.
			if IsSystemInDarkMode() {
				cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri)
			} else {
				cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri)
			}
		} else {
			// Si NO respetamos el modo oscuro, forzamos ambos para asegurar que se vea
			exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri).Run()
			cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri)
		}

	case "niri":
		// niri has its own IPC for setting wallpapers
		cmd = exec.Command("niri", "msg", "output", "*", "wallpaper", path)

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
		// For Dank Material Shell, the primary method is the IPC call,
		// which allows DMS to handle theming.
		ipcCmd := exec.Command("dms", "ipc", "call", "wallpaper", "set", path)
		err := ipcCmd.Run()
		if err == nil {
			// The IPC call was successful. We don't need to run anything else.
			return nil
		}

		// Fallback: If the IPC call fails (e.g., due to internal matugen
		// errors in DMS), try setting the wallpaper directly with quickshell.
		// This might not trigger theme updates but should change the image.
		utils.Log.Info("Warning: DMS IPC call failed (error: %v). Falling back to quickshell.", err)
		cmd = exec.Command("quickshell", "-w", path)

	case "swww":
		cmd = exec.Command("swww", "img", path)

	case "awww":
		cmd = exec.Command("awww", path)

	case "test":
		return nil

	default:
		return fmt.Errorf("unsupported or undetected desktop environment: '%s'", wc.Env)
	}

	return cmd.Run()
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// isProcessRunning checks if a process with the exact given name is running.
func isProcessRunning(processName string) bool {
	// pgrep is a standard utility on most Linux systems for this.
	cmd := exec.Command("pgrep", "-x", processName)
	// We don't care about the output, just the exit code.
	// Exit code 0 means process was found.
	if err := cmd.Run(); err == nil {
		return true
	}
	return false
}

// DetectDesktopEnv tries to determine the current desktop environment.
func DetectDesktopEnv() string {
	// 1. Check for running processes (most reliable indicator of an active session)
	// Give priority to dedicated wallpaper managers like dms (Dank Material Shell) if they are running.
	if isProcessRunning("dms") && commandExists("quickshell") {
		return "dms"
	}
	if isProcessRunning("niri") && commandExists("niri") {
		return "niri"
	}
	if isProcessRunning("sway") && commandExists("swaybg") {
		return "sway"
	}
	if isProcessRunning("gnome-shell") {
		return "gnome"
	}
	if isProcessRunning("plasmashell") {
		return "kde"
	}

	// 2. Check environment variables (less reliable, but good fallback)
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	if desktop == "test" {
		return "test"
	}
	// Check for specific DEs first to avoid broad matches like "gnome" in "ubuntu:GNOME"
	if strings.Contains(desktop, "niri") && commandExists("niri") {
		return "niri"
	}
	if strings.Contains(desktop, "sway") && commandExists("swaybg") {
		return "sway"
	}
	if strings.Contains(desktop, "gnome") {
		return "gnome"
	}
	if strings.Contains(desktop, "kde") || strings.Contains(desktop, "plasma") {
		return "kde"
	}

	desktopSession := strings.ToLower(os.Getenv("DESKTOP_SESSION"))
	if strings.Contains(desktopSession, "niri") && commandExists("niri") {
		return "niri"
	}
	if strings.Contains(desktopSession, "sway") && commandExists("swaybg") {
		return "sway"
	}
	if strings.Contains(desktopSession, "gnome") {
		return "gnome"
	}
	if strings.Contains(desktopSession, "plasma") || strings.Contains(desktopSession, "kde") {
		return "kde"
	}

	// 3. Last resort: check for installed command-line tools as a hint
	// These are generic and might not reflect the current session, so they have the lowest priority.
	if commandExists("swww") {
		return "swww"
	}
	if commandExists("awww") {
		return "awww"
	}
	if commandExists("quickshell") { // For dms
		return "dms"
	}
	if commandExists("feh") {
		return "feh"
	}
	if commandExists("nitrogen") {
		return "nitrogen"
	}
	// swaybg and gsettings are checked here as a last resort if the processes aren't running
	if commandExists("swaybg") {
		return "sway"
	}
	if commandExists("gsettings") { // Good hint for GNOME-based, but low priority
		return "gnome"
	}
	return "unknown"
}

func IsSystemInDarkMode() bool {
	// Detección para GNOME
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output()
	if err == nil {
		s := strings.TrimSpace(string(out))
		s = strings.Trim(s, "'")
		return s == "prefer-dark"
	}
	return false
}
