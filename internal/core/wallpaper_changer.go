package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gower/internal/utils"
	"gower/pkg/models"
)

type WallpaperChanger struct {
	Env                string
	RespectDarkMode    bool
	Transition         models.TransitionConfig
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

	if len(paths) == 0 {
		return fmt.Errorf("no wallpapers provided")
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

		return wc.applyToMonitors(targetMonitors, path)
	} else if multiMonitor == "distinct" {
		var allErrs []error
		for i, monitor := range monitors {
			path := paths[i%len(paths)]
			err := wc.applyToMonitor(monitor, path, i)
			if err != nil {
				allErrs = append(allErrs, err)
			}
		}
		if len(allErrs) > 0 {
			return fmt.Errorf("errors occurred while setting distinct wallpapers: %w", errors.Join(allErrs...))
		}
		return nil
	}

	return fmt.Errorf("invalid multi-monitor mode: %s", multiMonitor)
}

func (wc *WallpaperChanger) applyToMonitors(monitors []Monitor, path string) error {
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
			err := runCommand(ipcCmd)
			if err == nil {
				continue
			}
			wc.log().Info("Warning: DMS IPC call failed (error: %v). Falling back to quickshell.", err)
			cmd := exec.Command("quickshell", "-w", path)
			if err := runCommand(cmd); err != nil {
				allErrs = append(allErrs, fmt.Errorf("failed to set wallpaper for monitor %s: %w", monitor.Name, err))
			}
			continue
		}

		if err := wc.execCommand(monitor, path, 0); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	if len(allErrs) > 0 {
		return fmt.Errorf("errors occurred while setting wallpapers: %w", errors.Join(allErrs...))
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
		err := runCommand(ipcCmd)
		if err == nil {
			wc.log().Debug("DMS: IPC call successful for monitor %s", monitor.Name)
			return nil
		}
		wc.log().Info("Warning: DMS IPC call failed for monitor %s (error: %v). Falling back to quickshell.", monitor.Name, err)
		if err := runCommand(exec.Command("quickshell", "-w", path)); err != nil {
			return fmt.Errorf("failed to set wallpaper for monitor %s: %w", monitor.Name, err)
		}
		return nil
	}

	return wc.execCommand(monitor, path, index)
}

func (wc *WallpaperChanger) execCommand(monitor Monitor, path string, index int) error {
	// GNOME distingue picture-uri y picture-uri-dark. Cuando no se respeta el
	// tema hay que fijar ambas; antes se lanzaba una de ellas como efecto
	// secundario dentro de buildCommand.
	if wc.Env == "gnome" && !wc.RespectDarkMode {
		uri := "file://" + path
		if err := runCommand(exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri)); err != nil {
			return fmt.Errorf("failed to set wallpaper for monitor %s: %w", monitor.Name, err)
		}
		if err := runCommand(exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", uri)); err != nil {
			return fmt.Errorf("failed to set wallpaper for monitor %s: %w", monitor.Name, err)
		}
		return nil
	}

	cmd, err := wc.buildCommand(monitor, path, index)
	if err != nil {
		return err
	}
	if cmd == nil {
		return nil
	}

	// swaybg es un demonio en primer plano: cmd.Run() no retornaría nunca.
	if isBackgroundCommand(cmd) {
		return wc.runBackgroundCommand(cmd)
	}

	if isRoserlayCommand(cmd) {
		if err := runCommand(cmd); err != nil {
			wc.log().Error("roserlay failed (%v); falling back to native transitions", err)
			previousEngine := wc.Transition.Engine
			wc.Transition.Engine = "awww"
			native, nativeErr := wc.buildCommand(monitor, path, index)
			wc.Transition.Engine = previousEngine
			if nativeErr != nil {
				return nativeErr
			}
			if native == nil {
				return fmt.Errorf("roserlay failed and no native fallback is available: %w", err)
			}
			if isBackgroundCommand(native) {
				return wc.runBackgroundCommand(native)
			}
			if nativeErr := runCommand(native); nativeErr != nil {
				return fmt.Errorf("failed to set wallpaper for monitor %s: %w", monitor.Name, nativeErr)
			}
			return nil
		}
		return nil
	}

	if err := runCommand(cmd); err != nil {
		return fmt.Errorf("failed to set wallpaper for monitor %s: %w", monitor.Name, err)
	}
	return nil
}

func isRoserlayCommand(cmd *exec.Cmd) bool {
	return cmd != nil && filepath.Base(cmd.Path) == "roserlay"
}

func isBackgroundCommand(cmd *exec.Cmd) bool {
	if cmd == nil || len(cmd.Args) == 0 {
		return false
	}
	return filepath.Base(cmd.Args[0]) == "swaybg"
}

// SupportsTransitions indica si el backend activo acepta animación.
func (wc *WallpaperChanger) SupportsTransitions() bool {
	switch wc.Env {
	case "awww", "swww":
		return commandExists(wc.Env)
	case "niri":
		return commandExists("awww") || commandExists("swww")
	default:
		return false
	}
}

// transitionArgs traduce la configuración de transición a flags de awww/swww.
// Los valores a cero se omiten para dejar que el backend use su default.
func (wc *WallpaperChanger) transitionArgs() []string {
	t := wc.Transition
	if t.Type == "" {
		return nil
	}

	args := []string{"--transition-type", t.Type}
	if t.Duration > 0 {
		args = append(args, "--transition-duration", strconv.FormatFloat(t.Duration, 'f', -1, 64))
	}
	if t.FPS > 0 {
		args = append(args, "--transition-fps", strconv.Itoa(t.FPS))
	}
	if t.Step > 0 {
		args = append(args, "--transition-step", strconv.Itoa(t.Step))
	}
	if t.Angle > 0 {
		args = append(args, "--transition-angle", strconv.Itoa(t.Angle))
	}
	if t.Pos != "" {
		args = append(args, "--transition-pos", t.Pos)
	}
	if t.Bezier != "" {
		args = append(args, "--transition-bezier", t.Bezier)
	}
	if t.Wave != "" {
		args = append(args, "--transition-wave", t.Wave)
	}
	return args
}

// swwwCommand construye el comando compartido por swww y awww.
func (wc *WallpaperChanger) swwwCommand(binary string, monitor Monitor, path string) *exec.Cmd {
	args := []string{"img", "-o", monitor.Name}
	args = append(args, wc.transitionArgs()...)
	args = append(args, path)
	return exec.Command(binary, args...)
}

// swwwBinary devuelve qué ejecutable awww/swww está disponible para fijar el
// fondo definitivo. Es también el que usa roserlay en su --on-complete.
func (wc *WallpaperChanger) swwwBinary() (string, bool) {
	switch wc.Env {
	case "awww":
		if commandExists("awww") {
			return "awww", true
		}
	case "swww":
		if commandExists("swww") {
			return "swww", true
		}
	case "niri":
		if IsProcessRunning("awww-daemon") && commandExists("awww") {
			return "awww", true
		}
		if IsProcessRunning("swww-daemon") && commandExists("swww") {
			return "swww", true
		}
		if commandExists("awww") {
			return "awww", true
		}
		if commandExists("swww") {
			return "swww", true
		}
	}
	return "", false
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// roserlayMask traduce type/mask de la configuración a la máscara de roserlay.
// Devuelve también el ángulo y si hay que invertir el orden de revelado.
func roserlayMask(t models.TransitionConfig) (mask string, angle int, invert bool) {
	mask = t.Mask
	angle = t.Angle

	if mask == "" {
		switch t.Type {
		case "", "simple", "fade":
			mask = "crossfade"
		case "left":
			mask, angle = "wipe", 0
		case "right":
			mask, angle = "wipe", 180
		case "top":
			mask, angle = "wipe", 90
		case "bottom":
			mask, angle = "wipe", 270
		case "wipe", "wave":
			mask = "wipe"
		case "grow", "center", "any":
			mask = "radial"
		case "outer":
			mask = "radial"
			invert = true
		case "random":
			mask = "random"
		default:
			mask = "crossfade"
		}
	}

	if t.Type == "outer" {
		invert = true
	}
	return mask, angle, invert
}

// CurrentWallpaperPath consulta al backend (awww/swww) qué imagen se está
// mostrando en el monitor indicado.
func CurrentWallpaperPath(setterPath, monitorName string) (string, error) {
	out, err := outputWithTimeout(2*time.Second, setterPath, "query")
	if err != nil {
		return "", err
	}

	first := ""
	for _, line := range strings.Split(string(out), "\n") {
		idx := strings.Index(line, "image: ")
		if idx < 0 {
			continue
		}
		path := strings.TrimSpace(line[idx+len("image: "):])
		if path == "" {
			continue
		}
		if first == "" {
			first = path
		}
		if monitorName == "" || monitorName == "default" || strings.Contains(line, monitorName) {
			return path, nil
		}
	}

	if first != "" {
		return first, nil
	}
	return "", fmt.Errorf("current wallpaper not found")
}

// roserlayCommand construye el overlay propio de transición. Devuelve false si
// el motor no es roserlay, si no está instalado o si el backend no lo soporta;
// en ese caso se usa la transición nativa de awww.
func (wc *WallpaperChanger) roserlayCommand(monitor Monitor, path string) (*exec.Cmd, bool) {
	engine := wc.Transition.Engine
	if engine == "" {
		// Valor cero (tests / config antigua): comportamiento clásico.
		engine = "awww"
	}
	if engine == "awww" || wc.Transition.Type == "none" {
		return nil, false
	}

	roserlayPath, err := exec.LookPath("roserlay")
	if err != nil {
		if engine == "auto" {
			wc.log().Debug("transition engine auto: roserlay not found, using awww")
		}
		return nil, false
	}
	setter, ok := wc.swwwBinary()
	if !ok {
		if engine == "auto" {
			wc.log().Debug("transition engine auto: no awww/swww backend, using native transitions")
		}
		return nil, false
	}
	setterPath, err := exec.LookPath(setter)
	if err != nil {
		return nil, false
	}
	if engine == "auto" {
		wc.log().Debug("transition engine auto: using roserlay")
	}

	args := []string{"--new", path}
	if wc.Transition.Duration > 0 {
		args = append(args, "--duration", strconv.FormatFloat(wc.Transition.Duration, 'f', -1, 64))
	}
	if wc.Transition.Linger > 0 {
		args = append(args, "--linger", strconv.Itoa(wc.Transition.Linger))
	}

	mask, angle, invert := roserlayMask(wc.Transition)
	if wc.Transition.Invert {
		invert = true
	}
	maskFile := ""
	oldPath := ""
	if mask == "diff" {
		var err error
		oldPath, err = CurrentWallpaperPath(setterPath, monitor.Name)
		if err != nil {
			wc.log().Error("diff mask: cannot read current wallpaper for %s: %v; falling back to color", monitor.Name, err)
			mask = "color"
		}
	}
	if IsContentMask(mask) {
		generated, err := GenerateTransitionMaskFile(path, mask, oldPath)
		if err != nil {
			wc.log().Error("failed to generate %s mask for %s: %v; falling back to crossfade", mask, path, err)
			mask = "crossfade"
		} else {
			maskFile = generated
			mask = ""
		}
	}
	if maskFile != "" {
		args = append(args, "--mask-file", maskFile)
	} else if mask != "" {
		args = append(args, "--mask", mask)
	}
	if angle > 0 {
		args = append(args, "--angle", strconv.Itoa(angle))
	}
	if wc.Transition.Pos != "" {
		args = append(args, "--pos", wc.Transition.Pos)
	}
	if wc.Transition.Feather > 0 {
		args = append(args, "--feather", strconv.FormatFloat(wc.Transition.Feather, 'f', -1, 64))
	}
	if invert {
		args = append(args, "--invert")
	}
	if wc.Transition.Layer != "" {
		args = append(args, "--layer", wc.Transition.Layer)
	}
	useOutput := monitor.Name != "" && monitor.Name != "default"
	if useOutput {
		args = append(args, "--output", monitor.Name)
	}

	var onComplete string
	if useOutput {
		onComplete = fmt.Sprintf("%s img -o %s --transition-type none %s",
			shQuote(setterPath), shQuote(monitor.Name), shQuote(path))
	} else {
		onComplete = fmt.Sprintf("%s img --transition-type none %s",
			shQuote(setterPath), shQuote(path))
	}
	args = append(args, "--on-complete", onComplete)

	return exec.Command(roserlayPath, args...), true
}

func (wc *WallpaperChanger) runBackgroundCommand(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", cmd.Path, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// runCommand ejecuta el comando que fija el wallpaper. Es una costura para que
// los tests puedan comprobar la construcción del comando y el manejo de errores
// sin invocarlo: gsettings y dbus-send modificarían el escritorio real del
// usuario y swaymsg, swaybg, awww o swww llegarían a arrancar procesos.
//
// La implementación real aplica un timeout para que un compositor colgado no
// bloquee el daemon indefinidamente.
var runCommand = func(cmd *exec.Cmd) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(commandTimeout(cmd)):
		_ = cmd.Process.Kill()
		<-done
		return fmt.Errorf("command %s timed out", cmd.Path)
	}
}

// commandTimeout da más margen a roserlay porque su duración es configurable;
// el resto de comandos de escritorio deben responder rápido.
func commandTimeout(cmd *exec.Cmd) time.Duration {
	if filepath.Base(cmd.Path) == "roserlay" {
		return 5 * time.Minute
	}
	return 10 * time.Second
}

// startBackgroundProcess lanza un demonio de wallpaper en segundo plano.
var startBackgroundProcess = func(name string) error {
	cmd := exec.Command(name)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func (wc *WallpaperChanger) buildCommand(monitor Monitor, path string, index int) (*exec.Cmd, error) {
	if cmd, ok := wc.roserlayCommand(monitor, path); ok {
		return cmd, nil
	}

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
			return exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", uri), nil
		}

	case "niri":
		// Prioriza el demonio ya activo: tener swww y awww a la vez causa parpadeo.
		if IsProcessRunning("awww-daemon") && commandExists("awww") {
			return wc.swwwCommand("awww", monitor, path), nil
		}
		if IsProcessRunning("swww-daemon") && commandExists("swww") {
			return wc.swwwCommand("swww", monitor, path), nil
		}
		// Ninguno activo: intenta iniciar uno de los instalados.
		if commandExists("awww") {
			startDaemon("awww-daemon")
			return wc.swwwCommand("awww", monitor, path), nil
		}
		if commandExists("swww") {
			startDaemon("swww-daemon")
			return wc.swwwCommand("swww", monitor, path), nil
		}
		if commandExists("swaybg") {
			return exec.Command("swaybg", "-o", monitor.Name, "-i", path, "-m", "fill"), nil
		}
		return nil, fmt.Errorf("wallpaper setting for Niri requires 'swww', 'awww' or 'swaybg'. Please install one of them")

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
		return wc.swwwCommand("swww", monitor, path), nil

	case "awww":
		return wc.swwwCommand("awww", monitor, path), nil

	default:
		return nil, fmt.Errorf("unsupported or undetected desktop environment '%s'", wc.Env)
	}
}

// commandExists indica si hay un ejecutable en el PATH. Es una costura porque
// DetectDesktopEnv decide con ella, y los tests necesitan fijar que ajustadores
// hay disponibles sin depender de los instalados en la máquina.
var commandExists = func(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// outputWithTimeout ejecuta un comando de consulta con un timeout corto para
// que un compositor congelado no bloquee el daemon.
func outputWithTimeout(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

// IsProcessRunning checks if a process with the exact given name is running.
// Exported so other packages (p. ej. cmd/status) can detect daemon conflicts.
var IsProcessRunning = func(processName string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pgrep", "-x", processName)
	if err := cmd.Run(); err == nil {
		return true
	}
	return false
}

// isProcessRunning mantiene la costura que usan los tests. Es una función (no
// una copia del valor de IsProcessRunning) para que sustituir IsProcessRunning
// afecte también a la detección interna.
var isProcessRunning = func(processName string) bool {
	return IsProcessRunning(processName)
}

// startDaemon lanza un demonio de fondo y espera a que el proceso aparezca.
func startDaemon(name string) {
	if isProcessRunning(name) {
		return
	}
	if err := startBackgroundProcess(name); err != nil {
		utils.Log.Error("Failed to start %s: %v", name, err)
		return
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if isProcessRunning(name) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func DetectDesktopEnv() string {
	if runtime.GOOS == "windows" {
		return "windows"
	}
	// El compositor manda: un niri con swww-daemon activo se detecta como niri,
	// no como swww, para poder usar los nombres de monitor correctos.
	if (isProcessRunning("niri") || isProcessRunning("niri-session")) && commandExists("niri") {
		return "niri"
	}
	if isProcessRunning("sway") && commandExists("swaybg") {
		return "sway"
	}
	if (isProcessRunning("dms") || isProcessRunning("quickshell")) && (commandExists("dms") || commandExists("quickshell")) {
		return "dms"
	}
	if isProcessRunning("swww-daemon") && commandExists("swww") {
		return "swww"
	}
	if isProcessRunning("awww-daemon") && commandExists("awww") {
		return "awww"
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
		out, err := outputWithTimeout(3*time.Second, "hyprctl", "monitors", "-j")
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
		out, err := outputWithTimeout(3*time.Second, "swaymsg", "-t", "get_outputs")
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
	case "gnome", "kde", "feh", "nitrogen", "swww", "awww", "dms", "sway", "niri", "hyprland", "unknown", "":
		if commandExists("xrandr") {
			output, err := outputWithTimeout(3*time.Second, "xrandr", "--query")
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
		out, err := outputWithTimeout(3*time.Second, "dbus-send", "--session", "--print-reply=literal", "--dest=org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop", "org.freedesktop.portal.Settings.Read", "string:org.freedesktop.appearance", "string:color-scheme")
		if err == nil {
			if strings.Contains(string(out), "uint32 1") {
				return true
			}
		}
	}

	if commandExists("gsettings") {
		out, err := outputWithTimeout(3*time.Second, "gsettings", "get", "org.gnome.desktop.interface", "color-scheme")
		if err == nil {
			s := strings.TrimSpace(string(out))
			s = strings.Trim(s, "'")
			if s == "prefer-dark" {
				return true
			}
		}
	}

	if commandExists("kreadconfig5") {
		out, err := outputWithTimeout(3*time.Second, "kreadconfig5", "--file", "kdeglobals", "--group", "General", "--key", "ColorScheme")
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
