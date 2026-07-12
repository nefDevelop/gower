package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"gower/internal/utils"
)

type WallpaperChanger struct {
	Env                string
	RespectDarkMode    bool
	DetectMonitorsFunc func() ([]Monitor, error)
	SetWallpapersFunc  func([]string, []Monitor, string) error
	Log                *utils.Logger
}

func (wc *WallpaperChanger) log() *utils.Logger {
	if wc.Log != nil {
		return wc.Log
	}
	return utils.Log
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
	wc.log().Info("Setting wallpapers for %d monitors (Env: %s, Mode: %s)", len(monitors), wc.Env, multiMonitor)
	wc.log().Debug("Monitors detected: %+v", monitors)
	wc.log().Debug("Wallpapers provided: %v", paths)

	if wc.SetWallpapersFunc != nil {
		return wc.SetWallpapersFunc(paths, monitors, multiMonitor)
	}

	if (len(monitors) == 1 && multiMonitor != "distinct") || multiMonitor == "clone" || len(monitors) == 0 {
		path := paths[0]
		var targetMonitors []Monitor
		if len(monitors) == 1 {
			targetMonitors = monitors
		} else {
			var err error
			targetMonitors, err = wc.DetectMonitors()
			if err != nil {
				return fmt.Errorf("failed to detect monitors for clone mode: %w", err)
			}
		}

		return wc.applyToMonitors(targetMonitors, path, "")
	} else if multiMonitor == "distinct" {
		if len(paths) == 0 {
			return fmt.Errorf("no wallpapers provided for distinct mode")
		}

		var allErrs []error
		for i, monitor := range monitors {
			path := paths[i%len(paths)]
			err := wc.applyToMonitor(monitor, path, i)
			if err != nil {
				allErrs = append(allErrs, err)
			}
		}
		if len(allErrs) > 0 {
			return fmt.Errorf("errors occurred while setting distinct wallpapers: %v", allErrs)
		}
		return nil
	}

	return fmt.Errorf("invalid multi-monitor mode: %s", multiMonitor)
}

func (wc *WallpaperChanger) applyToMonitors(monitors []Monitor, path string, _ string) error {
	var allErrs []error
	for _, monitor := range monitors {
		wc.log().Info("Setting wallpaper for monitor %s: %s", monitor.Name, path)
		if wc.Env == "windows" {
			if err := setWallpaperWindows(path); err != nil {
				allErrs = append(allErrs, fmt.Errorf("failed to set wallpaper on Windows: %w", err))
			}
			continue
		}

		if wc.Env == "test" {
			wc.log().Info("Test environment: Would set wallpaper %s for monitor %s", path, monitor.Name)
			continue
		}

		if wc.Env == "dms" {
			ipcCmd := exec.Command("dms", "ipc", "call", "wallpaper", "setFor", monitor.Name, path)
			err := ipcCmd.Run()
			if err == nil {
				continue
			}
			wc.log().Info("Warning: DMS IPC call failed (error: %v). Falling back to quickshell.", err)
			cmd := exec.Command("quickshell", "-w", path)
			if err := cmd.Run(); err != nil {
				allErrs = append(allErrs, fmt.Errorf("failed to set wallpaper for monitor %s: %w", monitor.Name, err))
			}
			continue
		}

		if err := wc.execCommand(monitor, path, 0); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	if len(allErrs) > 0 {
		return fmt.Errorf("errors occurred while setting wallpapers: %v", allErrs)
	}
	return nil
}

func (wc *WallpaperChanger) applyToMonitor(monitor Monitor, path string, index int) error {
	wc.log().Info("Setting wallpaper for monitor %s: %s", monitor.Name, path)
	wc.log().Debug("Distinct Mode: Assigning '%s' to monitor '%s' (ID: %s)", path, monitor.Name, monitor.ID)

	if wc.Env == "windows" {
		wc.log().Info("Warning: 'distinct' mode is not fully supported on Windows. Setting wallpaper for all monitors.")
		if err := setWallpaperWindows(path); err != nil {
			return fmt.Errorf("failed to set wallpaper on Windows: %w", err)
		}
		return nil
	}

	if wc.Env == "test" {
		wc.log().Info("Test environment: Would set wallpaper %s for monitor %s", path, monitor.Name)
		return nil
	}

	if wc.Env == "dms" {
		ipcCmd := exec.Command("dms", "ipc", "call", "wallpaper", "setFor", monitor.Name, path)
		err := ipcCmd.Run()
		if err == nil {
			wc.log().Debug("DMS: IPC call successful for monitor %s", monitor.Name)
			return nil
		}
		wc.log().Error("Warning: DMS IPC call failed for monitor %s (error: %v).", monitor.Name, err)
		return nil
	}

	return wc.execCommand(monitor, path, index)
}

func (wc *WallpaperChanger) execCommand(monitor Monitor, path string, index int) error {
	cmd, err := wc.buildCommand(monitor, path, index)
	if err != nil {
		return err
	}
	if cmd == nil {
		return nil
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set wallpaper for monitor %s: %w", monitor.Name, err)
	}
	return nil
}

func (wc *WallpaperChanger) buildCommand(monitor Monitor, path string, index int) (*exec.Cmd, error) {
	switch wc.Env {
	case "kde":
		script := fmt.Sprintf(`
			var allDesktops = desktops();
			for (i=0;i<allDesktops.length;i++) {
				d = allDesktops[i];
				if (d.name == %s || d.id == parseInt(%s)) {
					d.wallpaperPlugin = "org.kde.image";
					d.currentConfigGroup = Array("Wallpaper", "org.kde.image", "General");
					d.writeConfig("Image", "file://%s");
				}
			}
		`, strconv.Quote(monitor.Name), strconv.Quote(monitor.ID), strconv.Quote(path))
		return exec.Command("dbus-send", "--session", "--dest=org.kde.plasmashell",
			"--type=method_call", "/PlasmaShell",
			"org.kde.PlasmaShell.evaluateScript",
			"string:"+script), nil

	case "gnome":
		uri := "file://" + path
		if wc.RespectDarkMode {
			if IsSystemInDarkMode() {
				return exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri), nil
			} else {
				return exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri), nil
			}
		} else {
			_ = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri).Run()
			return exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri), nil
		}

	case "niri":
		if commandExists("swww") {
			return exec.Command("swww", "img", "-o", monitor.Name, path), nil
		}
		return nil, fmt.Errorf("wallpaper setting for Niri requires 'swww'. Please install it")

	case "sway":
		if monitor.Name != "" && monitor.Name != "default" {
			return exec.Command("swaymsg", "output", monitor.Name, "bg", path, "fill"), nil
		}
		return exec.Command("swaybg", "-i", path, "-m", "fill"), nil

	case "feh":
		display := fmt.Sprintf(":%d.%d", 0, index)
		return exec.Command("feh", "--bg-fill", "--no-fehbg", "--display", display, path), nil

	case "nitrogen":
		return exec.Command("nitrogen", "--set-auto", "--save", path), nil

	case "swww":
		return exec.Command("swww", "img", "-o", monitor.Name, path), nil

	case "awww":
		return exec.Command("awww", "-o", monitor.Name, path), nil

	default:
		return nil, fmt.Errorf("unsupported or undetected desktop environment '%s'", wc.Env)
	}
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

var isProcessRunning = func(processName string) bool {
	cmd := exec.Command("pgrep", "-x", processName)
	if err := cmd.Run(); err == nil {
		return true
	}
	return false
}

func DetectDesktopEnv() string {
	if runtime.GOOS == "windows" {
		return "windows"
	}
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

	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	if desktop == "test" {
		return "test"
	}
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

	if commandExists("swww") {
		return "swww"
	}
	if commandExists("awww") {
		return "awww"
	}
	if commandExists("quickshell") {
		return "dms"
	}
	if commandExists("feh") {
		return "feh"
	}
	if commandExists("nitrogen") {
		return "nitrogen"
	}
	if commandExists("swaybg") {
		return "sway"
	}
	if commandExists("gsettings") {
		return "gnome"
	}
	return ""
}

type Monitor struct {
	ID      string
	Name    string
	Width   int
	Height  int
	X       int
	Y       int
	Primary bool
}

func (wc *WallpaperChanger) DetectMonitors() ([]Monitor, error) {
	wc.log().Info("Detecting monitors for environment: %s", wc.Env)
	if wc.DetectMonitorsFunc != nil {
		return wc.DetectMonitorsFunc()
	}

	if runtime.GOOS == "windows" {
		wc.log().Info("Windows environment: Assuming single monitor.")
		return []Monitor{{ID: "default", Name: "default", Primary: true}}, nil
	}

	var monitors []Monitor

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
						ID: m.Name, Name: m.Name, Width: m.Width, Height: m.Height,
						X: m.X, Y: m.Y, Primary: m.Focused,
					})
				}
				wc.log().Debug("Monitors detected via hyprctl: %d found", len(monitors))
				return monitors, nil
			}
		}
	}

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
						ID: m.Name, Name: m.Name, Width: m.Rect.Width, Height: m.Rect.Height,
						X: m.Rect.X, Y: m.Rect.Y, Primary: m.Focused,
					})
				}
				wc.log().Debug("Monitors detected via swaymsg: %d found", len(monitors))
				return monitors, nil
			}
		}
	}

	switch wc.Env {
	case "gnome", "kde", "feh", "nitrogen", "swww", "awww", "dms", "sway", "niri", "unknown":
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
						if len(resPosParts) == 3 {
							dim := strings.Split(resPosParts[0], "x")
							if len(dim) == 2 {
								_, _ = fmt.Sscanf(dim[0], "%d", &width)
								_, _ = fmt.Sscanf(dim[1], "%d", &height)
							}
							_, _ = fmt.Sscanf(resPosParts[1], "%d", &x)
							_, _ = fmt.Sscanf(resPosParts[2], "%d", &y)
						} else if len(resPosParts) == 2 {
							dim := strings.Split(resPosParts[0], "x")
							if len(dim) == 2 {
								_, _ = fmt.Sscanf(dim[0], "%d", &width)
								_, _ = fmt.Sscanf(dim[1], "%d", &height)
							}
							_, _ = fmt.Sscanf(resPosParts[1], "%d", &x)
						}
					}

					monitors = append(monitors, Monitor{
						ID: name, Name: name, Width: width, Height: height,
						X: x, Y: y, Primary: primary,
					})
				}
			}
			wc.log().Debug("Monitors detected via xrandr: %d found (before filtering)", len(monitors))
			monitors = filterXWaylandMonitors(monitors)
		} else {
			wc.log().Info("Warning: xrandr not found. Cannot detect monitors accurately for X11 environment.")
			monitors = append(monitors, Monitor{ID: "default", Name: "default", Primary: true})
		}

	default:
		wc.log().Info("Warning: Multi-monitor detection for environment '%s' is not supported. Assuming single monitor.", wc.Env)
		monitors = append(monitors, Monitor{ID: "default", Name: "default", Primary: true})
	}

	if len(monitors) == 0 {
		return nil, fmt.Errorf("no monitors detected")
	}
	return monitors, nil
}

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
	if commandExists("dbus-send") {
		out, err := exec.Command("dbus-send", "--session", "--print-reply=literal", "--dest=org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop", "org.freedesktop.portal.Settings.Read", "string:org.freedesktop.appearance", "string:color-scheme").Output()
		if err == nil {
			if strings.Contains(string(out), "uint32 1") {
				return true
			}
		}
	}

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

	if commandExists("kreadconfig5") {
		out, err := exec.Command("kreadconfig5", "--file", "kdeglobals", "--group", "General", "--key", "ColorScheme").Output()
		if err == nil {
			s := strings.ToLower(strings.TrimSpace(string(out)))
			if strings.Contains(s, "dark") {
				return true
			}
		}
	}

	if strings.Contains(strings.ToLower(os.Getenv("GTK_THEME")), "dark") {
		return true
	}
	if strings.ToLower(os.Getenv("QT_STYLE_OVERRIDE")) == "breeze-dark" {
		return true
	}

	return false
}
