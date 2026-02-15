// internal/core/wallpaper_changer.go
package core

import (
	"encoding/json"
	"fmt"
	"gower/internal/utils"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type WallpaperChanger struct {
	Env                string
	RespectDarkMode    bool
	DetectMonitorsFunc func() ([]Monitor, error)
	SetWallpapersFunc  func([]string, []Monitor, string) error
}

var NewWallpaperChanger = func(desktopEnv string, respectDarkMode ...bool) *WallpaperChanger {
	env := strings.ToLower(desktopEnv)
	if runtime.GOOS == "windows" {
		env = "windows"
	}
	if env == "" {
		env = DetectDesktopEnv()
		utils.Log.Debug("Auto-detected desktop environment: %s", env)
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
	utils.Log.Debug("Monitors detected: %+v", monitors)
	utils.Log.Debug("Wallpapers provided: %v", paths)

	if wc.SetWallpapersFunc != nil {
		return wc.SetWallpapersFunc(paths, monitors, multiMonitor)
	}

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
						if (d.name == "%s" || d.id == "%s") { // Target specific desktop
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
				if commandExists("swww") {
					cmd = exec.Command("swww", "img", "-o", monitor.Name, path)
				} else {
					return fmt.Errorf("wallpaper setting for Niri requires 'swww'. Please install it")
				}

			case "sway":
				if monitor.Name != "" && monitor.Name != "default" {
					cmd = exec.Command("swaymsg", "output", monitor.Name, "bg", path, "fill")
				} else {
					cmd = exec.Command("swaybg", "-i", path, "-m", "fill")
				}

			case "feh":
				display := fmt.Sprintf(":%d.%d", 0, 0) // Default to :0.0 for single target
				if monitor.ID != "default" {
					// Attempt to map monitor ID to X display if possible, otherwise use default
					// This is a simplification; a robust solution would parse xrandr output more thoroughly.
					// For now, we assume the first detected monitor is :0.0, second :0.1, etc.
					allMonitors, err := wc.DetectMonitors()
					if err == nil {
						for idx, m := range allMonitors {
							if m.ID == monitor.ID {
								display = fmt.Sprintf(":%d.%d", 0, idx)
								break
							}
						}
					}
				}
				cmd = exec.Command("feh", "--bg-fill", "--no-fehbg", "--display", display, path)

			case "nitrogen":
				cmd = exec.Command("nitrogen", "--set-auto", "--save", path)

			case "dms":
				var ipcCmd *exec.Cmd
				if monitor.Name != "" && monitor.Name != "default" {
					ipcCmd = exec.Command("dms", "ipc", "call", "wallpaper", "setFor", monitor.Name, path)
				} else {
					ipcCmd = exec.Command("dms", "ipc", "call", "wallpaper", "set", path)
				}

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

			case "windows":
				// En Windows, la llamada al sistema aplica el fondo a todos los monitores
				// según la configuración del usuario (expandir, rellenar, etc.).
				if err := setWallpaperWindows(path); err != nil {
					allErrs = append(allErrs, fmt.Errorf("failed to set wallpaper on Windows: %w", err))
				}
				continue // Continuar al siguiente monitor es irrelevante en Windows para el modo clon.

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
		if len(paths) == 0 {
			return fmt.Errorf("no wallpapers provided for distinct mode")
		}
		// If fewer wallpapers than monitors, we cycle through them.

		var allErrs []error
	MonitorLoop:
		for i, monitor := range monitors {
			path := paths[i%len(paths)] // Cycle through wallpapers if fewer than monitors
			utils.Log.Info("Setting wallpaper for monitor %s: %s", monitor.Name, path)
			utils.Log.Debug("Distinct Mode: Assigning '%s' to monitor '%s' (ID: %s)", path, monitor.Name, monitor.ID)

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
				if commandExists("swww") {
					cmd = exec.Command("swww", "img", "-o", monitor.Name, path)
				} else {
					return fmt.Errorf("wallpaper setting for Niri requires 'swww'. Please install it")
				}

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
				utils.Log.Debug("DMS: Executing IPC call for monitor %s...", monitor.Name)
				ipcCmd := exec.Command("dms", "ipc", "call", "wallpaper", "setFor", monitor.Name, path)
				err := ipcCmd.Run()
				if err == nil {
					utils.Log.Debug("DMS: IPC call successful for monitor %s", monitor.Name)
					continue // Successfully set for this monitor
				}
				utils.Log.Error("Warning: DMS IPC call failed for monitor %s (error: %v).", monitor.Name, err)
				// Do not fallback to global quickshell -w in distinct mode as it overwrites other monitors
				if len(monitors) == 1 {
					utils.Log.Debug("DMS: Falling back to quickshell -w for single monitor")
					cmd = exec.Command("quickshell", "-w", path)
				} else {
					utils.Log.Debug("DMS: Skipping fallback in multi-monitor distinct mode to avoid overwrite")
					continue
				}

			case "test":
				utils.Log.Info("Test environment: Would set wallpaper %s for monitor %s", path, monitor.Name)
				continue

			case "windows":
				// El modo "distinct" es complejo en Windows y requiere APIs más avanzadas.
				// Por ahora, nos comportamos como "clone" y establecemos el mismo fondo en todos lados.
				utils.Log.Info("Warning: 'distinct' mode is not fully supported on Windows. Setting wallpaper for all monitors.")
				if err := setWallpaperWindows(path); err != nil {
					allErrs = append(allErrs, fmt.Errorf("failed to set wallpaper on Windows: %w", err))
				}
				break MonitorLoop // Salimos del bucle de monitores, ya que Windows lo aplica a todos.
			default:
				err := fmt.Errorf("unsupported or undetected desktop environment '%s' for distinct multi-monitor mode", wc.Env)
				allErrs = append(allErrs, err)
				utils.Log.Error("Error: %v", err)
				continue
			}

			if cmd != nil {
				utils.Log.Debug("Executing command: %s", cmd.String())
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
var isProcessRunning = func(processName string) bool {
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
	if runtime.GOOS == "windows" {
		return "windows"
	}
	// 1. Check for running processes (most reliable indicator of an active session)
	// Give priority to dedicated wallpaper managers like dms (Dank Material Shell) if they are running.
	if isProcessRunning("swww-daemon") && commandExists("swww") {
		return "swww"
	}
	if (isProcessRunning("dms") || isProcessRunning("quickshell")) && (commandExists("dms") || commandExists("quickshell")) {
		return "dms"
	}
	if (isProcessRunning("niri") || isProcessRunning("niri-session")) && commandExists("niri") {
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
	if strings.Contains(desktop, "hyprland") {
		if commandExists("dms") {
			return "dms"
		}
		if commandExists("swww") {
			return "swww"
		}
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
	if wc.DetectMonitorsFunc != nil {
		return wc.DetectMonitorsFunc()
	}

	if runtime.GOOS == "windows" {
		// La detección de monitores en Windows es compleja y requiere llamadas a la API de Windows.
		// Por ahora, para simplificar, asumimos un solo monitor.
		utils.Log.Info("Windows environment: Assuming single monitor. Full multi-monitor detection requires platform-specific API calls.")
		return []Monitor{{ID: "default", Name: "default", Primary: true}}, nil
	}

	var monitors []Monitor

	// 1. Try Hyprland (hyprctl) - Common for DMS users
	if commandExists("hyprctl") {
		out, err := exec.Command("hyprctl", "monitors", "-j").Output()
		if err == nil {
			var hyprMonitors []struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Width   int    `json:"width"`
				Height  int    `json:"height"`
				X       int    `json:"x"`
				Y       int    `json:"y"`
				Focused bool   `json:"focused"`
			}
			if err := json.Unmarshal(out, &hyprMonitors); err == nil && len(hyprMonitors) > 0 {
				for _, m := range hyprMonitors {
					monitors = append(monitors, Monitor{
						ID:      m.Name,
						Name:    m.Name,
						Width:   m.Width,
						Height:  m.Height,
						X:       m.X,
						Y:       m.Y,
						Primary: m.Focused,
					})
				}
				utils.Log.Debug("Monitors detected via hyprctl: %d found", len(monitors))
				return monitors, nil
			}
		}
	}

	// 2. Try Sway (swaymsg)
	if commandExists("swaymsg") {
		out, err := exec.Command("swaymsg", "-t", "get_outputs").Output()
		if err == nil {
			var swayMonitors []struct {
				Name string `json:"name"`
				Rect struct {
					Width  int `json:"width"`
					Height int `json:"height"`
					X      int `json:"x"`
					Y      int `json:"y"`
				} `json:"rect"`
				Focused bool `json:"focused"`
				Active  bool `json:"active"`
			}
			if err := json.Unmarshal(out, &swayMonitors); err == nil && len(swayMonitors) > 0 {
				for _, m := range swayMonitors {
					if !m.Active {
						continue
					}
					monitors = append(monitors, Monitor{
						ID:      m.Name,
						Name:    m.Name,
						Width:   m.Rect.Width,
						Height:  m.Rect.Height,
						X:       m.Rect.X,
						Y:       m.Rect.Y,
						Primary: m.Focused,
					})
				}
				utils.Log.Debug("Monitors detected via swaymsg: %d found", len(monitors))
				return monitors, nil
			}
		}
	}

	switch wc.Env {
	case "gnome", "kde", "feh", "nitrogen", "swww", "awww", "dms", "sway", "niri", "unknown": // X11-based or compatible
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

			utils.Log.Debug("Monitors detected via xrandr: %d found (before filtering)", len(monitors))
			monitors = filterXWaylandMonitors(monitors)
		} else {
			utils.Log.Info("Warning: xrandr not found. Cannot detect monitors accurately for X11 environment.")
			// Fallback to a single monitor if xrandr is not available
			monitors = append(monitors, Monitor{ID: "default", Name: "default", Primary: true})
		}

	default:
		utils.Log.Info("Warning: Multi-monitor detection for environment '%s' is not supported. Assuming single monitor.", wc.Env)
		monitors = append(monitors, Monitor{ID: "default", Name: "default", Primary: true})
	}

	if len(monitors) == 0 {
		return nil, fmt.Errorf("no monitors detected")
	}
	return monitors, nil
}

// filterXWaylandMonitors removes virtual XWAYLAND monitors if real hardware monitors are present.
func filterXWaylandMonitors(monitors []Monitor) []Monitor {
	var realMonitors []Monitor
	for _, m := range monitors {
		if !strings.HasPrefix(m.Name, "XWAYLAND") {
			realMonitors = append(realMonitors, m)
		}
	}
	if len(realMonitors) > 0 {
		return realMonitors
	}
	return monitors
}

func IsSystemInDarkMode() bool {
	// 1. XDG Desktop Portal (Estándar moderno para Wayland/Flatpak/Sandboxed)
	// Devuelve uint32 1 para oscuro, 0 para sin preferencia, 2 para claro.
	if commandExists("dbus-send") {
		out, err := exec.Command("dbus-send", "--session", "--print-reply=literal", "--dest=org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop", "org.freedesktop.portal.Settings.Read", "string:org.freedesktop.appearance", "string:color-scheme").Output()
		if err == nil {
			if strings.Contains(string(out), "uint32 1") {
				return true
			}
		}
	}

	// 2. GNOME / GTK (gsettings)
	if commandExists("gsettings") {
		out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output()
		if err == nil {
			s := strings.TrimSpace(string(out))
			s = strings.Trim(s, "'")
			if s == "prefer-dark" {
				return true
			}
		}
	}

	// 3. KDE Plasma (kreadconfig5)
	if commandExists("kreadconfig5") {
		out, err := exec.Command("kreadconfig5", "--file", "kdeglobals", "--group", "General", "--key", "ColorScheme").Output()
		if err == nil {
			s := strings.ToLower(strings.TrimSpace(string(out)))
			// Los esquemas oscuros suelen tener "Dark" en el nombre (ej. BreezeDark)
			if strings.Contains(s, "dark") {
				return true
			}
		}
	}

	// 4. Variables de entorno (Fallback)
	if strings.Contains(strings.ToLower(os.Getenv("GTK_THEME")), "dark") {
		return true
	}
	if strings.ToLower(os.Getenv("QT_STYLE_OVERRIDE")) == "breeze-dark" {
		return true
	}

	return false
}
