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

var NewWallpaperChanger = func(desktopEnv string, respectDarkMode ...bool) *WallpaperChanger {
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

func (wc *WallpaperChanger) SetWallpapers(paths []string, monitors []Monitor, multiMonitor string) error {
	utils.Log.Info("Setting wallpapers for %d monitors (Env: %s, Mode: %s)", len(monitors), wc.Env, multiMonitor)

	if (len(monitors) == 1 && multiMonitor != "distinct") || multiMonitor == "clone" || len(monitors) == 0 {
		path := paths[0]
		var targetMonitors []Monitor
		if len(monitors) == 1 { // Specific monitor targeted
			targetMonitors = monitors
		} else { // Clone mode or no specific target, apply to all detected
			var err error
			targetMonitors, err = wc.DetectMonitors()
			if err != nil {
				return fmt.Errorf("failed to detect monitors for clone mode: %w", err)
			}
		}

		var allErrs []error
		for _, monitor := range targetMonitors {
			utils.Log.Info("Setting wallpaper for monitor %s: %s", monitor.Name, path)
			var cmd *exec.Cmd
			switch wc.Env {
			case "kde":
				script := fmt.Sprintf(`
					var allDesktops = desktops();
					for (i=0;i<allDesktops.length;i++) {
						d = allDesktops[i];
						if (d.name == "%s" || d.id == %d) { // Target specific desktop
							d.wallpaperPlugin = "org.kde.image";
							d.currentConfigGroup = Array("Wallpaper", "org.kde.image", "General");
							d.writeConfig("Image", "file://%s");
						}
					}
				`, monitor.Name, monitor.ID, path)
				cmd = exec.Command("dbus-send", "--session", "--dest=org.kde.plasmashell",
					"--type=method_call", "/PlasmaShell",
					"org.kde.PlasmaShell.evaluateScript",
					"string:"+script)

			case "gnome":
				// GNOME's gsettings doesn't directly support per-monitor distinct wallpapers easily.
				// It's usually a single wallpaper stretched/scaled across all.
				// For now, we'll apply to the primary monitor or all if no specific target.
				uri := "file://" + path
				if wc.RespectDarkMode {
					if IsSystemInDarkMode() {
						cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri)
					} else {
						cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri)
					}
				} else {
					exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri).Run()
					cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri)
				}

			case "niri":
				cmd = exec.Command("niri", "msg", "output", monitor.Name, "wallpaper", path)

			case "sway":
				cmd = exec.Command("swaybg", "-i", path, "-m", "fill")

			case "feh":
				display := fmt.Sprintf(":%d.%d", 0, 0) // Default to :0.0 for single target
				if monitor.ID != "default" {
					// Attempt to map monitor ID to X display if possible, otherwise use default
					// This is a simplification; a robust solution would parse xrandr output more thoroughly.
					// For now, we assume the first detected monitor is :0.0, second :0.1, etc.
					for idx, m := range targetMonitors {
						if m.ID == monitor.ID {
							display = fmt.Sprintf(":%d.%d", 0, idx)
							break
						}
					}
				}
				cmd = exec.Command("feh", "--bg-fill", "--no-fehbg", "--display", display, path)

			case "nitrogen":
				cmd = exec.Command("nitrogen", "--set-auto", "--save", path)

			case "dms":
				ipcCmd := exec.Command("dms", "ipc", "call", "wallpaper", "set", path)
				err := ipcCmd.Run()
				if err == nil {
					continue
				}
				utils.Log.Info("Warning: DMS IPC call failed (error: %v). Falling back to quickshell.", err)
				cmd = exec.Command("quickshell", "-w", path)

			case "swww":
				cmd = exec.Command("swww", "img", "-o", monitor.Name, path)

			case "awww":
				cmd = exec.Command("awww", "-o", monitor.Name, path)

			case "test":
				utils.Log.Info("Test environment: Would set wallpaper %s for monitor %s", path, monitor.Name)
				continue

			default:
				err := fmt.Errorf("unsupported or undetected desktop environment '%s' for single/clone mode", wc.Env)
				allErrs = append(allErrs, err)
				utils.Log.Error("Error: %v", err)
				continue
			}

			if cmd != nil {
				if err := cmd.Run(); err != nil {
					allErrs = append(allErrs, fmt.Errorf("failed to set wallpaper for monitor %s: %w", monitor.Name, err))
					utils.Log.Error("Failed to set wallpaper for monitor %s: %v", monitor.Name, err)
				}
			}
		}
		if len(allErrs) > 0 {
			return fmt.Errorf("errors occurred while setting wallpapers: %v", allErrs)
		}
		return nil
	} else if multiMonitor == "distinct" {
		if len(paths) < len(monitors) {
			return fmt.Errorf("not enough wallpapers (%d) for %d distinct monitors", len(paths), len(monitors))
		}

		var allErrs []error
		for i, monitor := range monitors {
			path := paths[i%len(paths)] // Cycle through wallpapers if fewer than monitors
			utils.Log.Info("Setting wallpaper for monitor %s: %s", monitor.Name, path)

			var cmd *exec.Cmd
			switch wc.Env {
			case "kde":
				// KDE's script already iterates through desktops, so we need to be careful.
				// For distinct, we'd ideally set per-desktop. This is a complex task for dbus-send.
				// For now, we'll log a warning and fall back to clone behavior for KDE distinct.
				utils.Log.Info("Warning: KDE does not easily support distinct wallpapers per monitor via current method. Falling back to clone for monitor %s.", monitor.Name)
				script := fmt.Sprintf(`
					var allDesktops = desktops();
					for (i=0;i<allDesktops.length;i++) {
						d = allDesktops[i];
						if (d.name == "%s" || d.id == %s) { // Attempt to target specific desktop
							d.wallpaperPlugin = "org.kde.image";
							d.currentConfigGroup = Array("Wallpaper", "org.kde.image", "General");
							d.writeConfig("Image", "file://%s");
						}
					}
				`, monitor.Name, monitor.ID, path) // monitor.ID is int, so no quotes
				cmd = exec.Command("dbus-send", "--session", "--dest=org.kde.plasmashell",
					"--type=method_call", "/PlasmaShell",
					"org.kde.PlasmaShell.evaluateScript",
					"string:"+script)

			case "gnome":
				// GNOME's gsettings doesn't directly support per-monitor distinct wallpapers easily.
				// It's usually a single wallpaper stretched/scaled across all.
				// For true distinct, one might need extensions or more complex dconf interactions.
				// For now, we'll log a warning and fall back to clone behavior for GNOME distinct.
				utils.Log.Info("Warning: GNOME does not easily support distinct wallpapers per monitor via current method. Falling back to clone for monitor %s.", monitor.Name)
				uri := "file://" + path
				if wc.RespectDarkMode {
					if IsSystemInDarkMode() {
						cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri)
					} else {
						cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri)
					}
				} else {
					exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri).Run()
					cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri)
				}

			case "feh":
				// feh can set per monitor. We need to know the X display for each monitor.
				// xrandr output gives us monitor names like "eDP-1", "DP-1", etc.
				// feh --bg-fill --no-fehbg --display :0.0 --image /path/to/wallpaper
				// This assumes X display :0.0 for the first monitor, :0.1 for the second, etc.
				// This mapping is not always straightforward or reliable.
				// A more robust solution would involve parsing xrandr output for display names.
				// For now, a simplified approach:
				display := fmt.Sprintf(":%d.%d", 0, i) // Assuming :0.0, :0.1, :0.2 etc.
				cmd = exec.Command("feh", "--bg-fill", "--no-fehbg", "--display", display, path)

			case "sway":
				// Sway uses swaymsg for per-output wallpaper.
				// `swaymsg output <monitor_name> bg <path> fill`
				cmd = exec.Command("swaymsg", "output", monitor.Name, "bg", path, "fill")

			case "niri":
				// niri uses niri msg output <monitor_name> wallpaper <path>
				cmd = exec.Command("niri", "msg", "output", monitor.Name, "wallpaper", path)

			case "swww":
				// swww can set per monitor.
				// `swww img -o <monitor_name> <path>`
				cmd = exec.Command("swww", "img", "-o", monitor.Name, path)

			case "awww":
				// awww can set per monitor.
				// `awww -o <monitor_name> <path>`
				cmd = exec.Command("awww", "-o", monitor.Name, path)

			case "nitrogen":
				// Nitrogen handles multi-monitor itself, but typically with one config.
				// Setting distinct per monitor with nitrogen programmatically is complex.
				utils.Log.Info("Warning: Nitrogen does not easily support distinct wallpapers per monitor programmatically. Falling back to clone for monitor %s.", monitor.Name)
				cmd = exec.Command("nitrogen", "--set-auto", "--save", path)

			case "dms":
				// DMS IPC call for per-monitor is not directly available.
				utils.Log.Info("Warning: DMS does not easily support distinct wallpapers per monitor via current method. Falling back to clone for monitor %s.", monitor.Name)
				ipcCmd := exec.Command("dms", "ipc", "call", "wallpaper", "set", path)
				err := ipcCmd.Run()
				if err == nil {
					continue // Successfully set for this monitor
				}
				utils.Log.Info("Warning: DMS IPC call failed for monitor %s (error: %v). Falling back to quickshell.", monitor.Name, err)
				cmd = exec.Command("quickshell", "-w", path)

			case "test":
				utils.Log.Info("Test environment: Would set wallpaper %s for monitor %s", path, monitor.Name)
				continue

			default:
				err := fmt.Errorf("unsupported or undetected desktop environment '%s' for distinct multi-monitor mode", wc.Env)
				allErrs = append(allErrs, err)
				utils.Log.Error("Error: %v", err)
				continue
			}

			if cmd != nil {
				if err := cmd.Run(); err != nil {
					allErrs = append(allErrs, fmt.Errorf("failed to set wallpaper for monitor %s: %w", monitor.Name, err))
					utils.Log.Error("Failed to set wallpaper for monitor %s: %v", monitor.Name, err)
				}
			}
		}
		if len(allErrs) > 0 {
			return fmt.Errorf("errors occurred while setting distinct wallpapers: %v", allErrs)
		}
		return nil
	}

	return fmt.Errorf("invalid multi-monitor mode: %s", multiMonitor)
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
	return ""
}

// Monitor represents a detected display monitor.
type Monitor struct {
	ID      string
	Name    string
	Width   int
	Height  int
	X       int
	Y       int
	Primary bool
}

// DetectMonitors detects and returns a list of connected monitors.
func (wc *WallpaperChanger) DetectMonitors() ([]Monitor, error) {
	utils.Log.Info("Detecting monitors for environment: %s", wc.Env)
	var monitors []Monitor

	switch wc.Env {
	case "gnome", "kde", "feh", "nitrogen", "swww", "awww", "dms", "unknown": // X11-based or compatible
		if commandExists("xrandr") {
			cmd := exec.Command("xrandr", "--query")
			output, err := cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("failed to query xrandr: %w", err)
			}

			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, " connected") {
					parts := strings.Fields(line)
					name := parts[0]
					primary := strings.Contains(line, " primary")

					// Attempt to parse resolution and position
					resPos := ""
					for _, part := range parts {
						if strings.Contains(part, "+") && strings.Contains(part, "x") {
							resPos = part
							break
						}
					}

					width, height, x, y := 0, 0, 0, 0
					if resPos != "" {
						resPosParts := strings.Split(resPos, "+")
						if len(resPosParts) == 3 { // e.g., 1920x1080+0+0
							dim := strings.Split(resPosParts[0], "x")
							if len(dim) == 2 {
								fmt.Sscanf(dim[0], "%d", &width)
								fmt.Sscanf(dim[1], "%d", &height)
							}
							fmt.Sscanf(resPosParts[1], "%d", &x)
							fmt.Sscanf(resPosParts[2], "%d", &y)
						} else if len(resPosParts) == 2 { // e.g., 1920x1080+0
							dim := strings.Split(resPosParts[0], "x")
							if len(dim) == 2 {
								fmt.Sscanf(dim[0], "%d", &width)
								fmt.Sscanf(dim[1], "%d", &height)
							}
							fmt.Sscanf(resPosParts[1], "%d", &x)
						}
					}

					monitors = append(monitors, Monitor{
						ID:      name,
						Name:    name,
						Width:   width,
						Height:  height,
						X:       x,
						Y:       y,
						Primary: primary,
					})
				}
			}
		} else {
			utils.Log.Info("Warning: xrandr not found. Cannot detect monitors accurately for X11 environment.")
			// Fallback to a single monitor if xrandr is not available
			monitors = append(monitors, Monitor{ID: "default", Name: "default", Primary: true})
		}

	case "sway", "niri": // Wayland compositors
		// For Sway/Niri, we would typically use their IPC.
		// This is a placeholder for future implementation.
		utils.Log.Info("Warning: Multi-monitor detection for %s is not yet fully implemented. Assuming single monitor.", wc.Env)
		monitors = append(monitors, Monitor{ID: "default", Name: "default", Primary: true})

	default:
		utils.Log.Info("Warning: Multi-monitor detection for environment '%s' is not supported. Assuming single monitor.", wc.Env)
		monitors = append(monitors, Monitor{ID: "default", Name: "default", Primary: true})
	}

	if len(monitors) == 0 {
		return nil, fmt.Errorf("no monitors detected")
	}
	return monitors, nil
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
