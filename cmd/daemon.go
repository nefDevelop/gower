package cmd

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gower/internal/core"
	"gower/internal/utils"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var (
	daemonInterval      int
	daemonFromFavorites bool
	daemonTheme         string
	daemonForce         bool
	daemonJSON          bool
	daemonForeground    bool

	daemonTransitionType     string
	daemonTransitionDuration float64
	daemonTransitionFPS      int
	daemonTransitionStep     int
	daemonTransitionAngle    int
	daemonTransitionPos      string
	daemonTransitionBezier   string
	daemonTransitionWave     string
	daemonTransitionEngine   string
	daemonTransitionLinger   int
	daemonTransitionMask     string
	daemonTransitionFeather  float64
	daemonTransitionInvert   bool
	daemonTransitionLayer    string
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start and manage the gower daemon",
	Long: `This command starts the gower daemon, which automatically changes
wallpapers based on your configuration.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon",
	Run:   runDaemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the daemon",
	Run:   runDaemonStop,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Run:   runDaemonStatus,
}

var daemonPauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause the daemon",
	Run: func(cmd *cobra.Command, args []string) {
		if _, alive := daemonPID(); !alive {
			if !config.Quiet {
				cmd.Println("Daemon not running.")
			}
			return
		}
		pauseFile := getPauseFilePath()
		if f, err := os.Create(pauseFile); err != nil {
			cmd.PrintErrf("Error creating pause signal: %v\n", err)
		} else {
			if err := f.Close(); err != nil {
				utils.Log.Error("Failed to close pause file: %v", err)
			}
			if !config.Quiet {
				cmd.Println("Daemon pause signal sent.")
			}
		}
	},
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install daemon as a systemd user service",
	Long: `Creates a systemd user service unit and enables it to start on boot.

The daemon will run as a user service under systemd, automatically
changing wallpapers at the configured interval.`,
	Run: runDaemonInstall,
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the systemd user service",
	Run:   runDaemonUninstall,
}

var daemonResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume the daemon",
	Run: func(cmd *cobra.Command, args []string) {
		if _, alive := daemonPID(); !alive {
			if !config.Quiet {
				cmd.Println("Daemon not running.")
			}
			return
		}
		pauseFile := getPauseFilePath()
		if err := os.Remove(pauseFile); err != nil {
			if !os.IsNotExist(err) {
				cmd.PrintErrf("Error removing pause signal: %v\n", err)
			}
		}
		if !config.Quiet {
			cmd.Println("Daemon resume signal sent.")
		}
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonPauseCmd)
	daemonCmd.AddCommand(daemonResumeCmd)
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)

	daemonStartCmd.Flags().IntVar(&daemonInterval, "interval", 30, "Interval in minutes")
	daemonStartCmd.Flags().BoolVar(&daemonFromFavorites, "from-favorites", false, "Include favorites")
	daemonStartCmd.Flags().StringVar(&daemonTheme, "theme", "", "Filter by theme")
	daemonStartCmd.Flags().BoolVar(&daemonForeground, "foreground", false, "Run in foreground (blocking)")
	daemonStartCmd.Flags().StringVar(&daemonTransitionType, "transition-type", "", "transition effect for awww/swww")
	daemonStartCmd.Flags().Float64Var(&daemonTransitionDuration, "transition-duration", 0, "transition duration in seconds")
	daemonStartCmd.Flags().IntVar(&daemonTransitionFPS, "transition-fps", 0, "transition frame rate")
	daemonStartCmd.Flags().IntVar(&daemonTransitionStep, "transition-step", 0, "transition step")
	daemonStartCmd.Flags().IntVar(&daemonTransitionAngle, "transition-angle", 0, "wipe/wave angle in degrees")
	daemonStartCmd.Flags().StringVar(&daemonTransitionPos, "transition-pos", "", "grow/outer center")
	daemonStartCmd.Flags().StringVar(&daemonTransitionBezier, "transition-bezier", "", "fade bezier curve")
	daemonStartCmd.Flags().StringVar(&daemonTransitionWave, "transition-wave", "", "wave width and height")
	daemonStartCmd.Flags().StringVar(&daemonTransitionEngine, "transition-engine", "", "transition engine [auto|awww|roserlay]")
	daemonStartCmd.Flags().IntVar(&daemonTransitionLinger, "transition-linger", 0, "roserlay: ms to keep the overlay after committing the image")
	daemonStartCmd.Flags().StringVar(&daemonTransitionMask, "transition-mask", "", "roserlay mask [crossfade|wipe|radial|drops|random|color|edges|luminance|hue|saliency|diff]")
	daemonStartCmd.Flags().Float64Var(&daemonTransitionFeather, "transition-feather", 0, "roserlay mask feather (0..1)")
	daemonStartCmd.Flags().BoolVar(&daemonTransitionInvert, "transition-invert", false, "roserlay: invert reveal order")
	daemonStartCmd.Flags().StringVar(&daemonTransitionLayer, "transition-layer", "", "roserlay layer [background|bottom|top|overlay]")

	daemonStopCmd.Flags().BoolVar(&daemonForce, "force", false, "Force stop")
}

func getPidFilePath() string {
	appDir, _ := core.GetAppDir()
	return filepath.Join(appDir, "gower.pid")
}

func getPauseFilePath() string {
	appDir, _ := core.GetAppDir()
	return filepath.Join(appDir, "gower.pause")
}

func getStopFilePath() string {
	appDir, _ := core.GetAppDir()
	return filepath.Join(appDir, "gower.stop")
}

// processAlive comprueba si un PID sigue vivo con la señal 0.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// daemonPID lee el PID file y comprueba que el proceso siga vivo.
func daemonPID() (int, bool) {
	data, err := os.ReadFile(getPidFilePath())
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, processAlive(pid)
}

// acquirePidFile crea el PID file con O_EXCL para que dos arranques
// simultáneos no puedan pasar ambos la comprobación de existencia.
func acquirePidFile(path string) error {
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			_, writeErr := fmt.Fprintf(f, "%d", os.Getpid())
			if writeErr != nil {
				_ = f.Close()
				_ = os.Remove(path)
				return writeErr
			}
			_ = f.Sync()
			return f.Close()
		}
		if !os.IsExist(err) {
			return err
		}

		if pid, alive := daemonPID(); alive {
			return fmt.Errorf("daemon already running with PID %d", pid)
		}
		// PID file obsoleto: se elimina y se reintenta la creación exclusiva.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return fmt.Errorf("could not acquire pid file %s", path)
}

func runDaemonStart(cmd *cobra.Command, args []string) {
	if !daemonForeground {
		pidFile := getPidFilePath()
		// Check if PID file exists and if the process is actually running
		if _, err := os.Stat(pidFile); err == nil { // PID file exists
			if !config.Quiet {
				cmd.Println("Checking for existing daemon process...")
			}
			// PID file exists, read PID
			pidData, readErr := os.ReadFile(pidFile)
			if readErr == nil {
				pid, _ := strconv.Atoi(string(pidData))
				if pid > 0 {
					proc, findErr := os.FindProcess(pid)
					if findErr == nil {
						// Send signal 0 to check if process is alive
						if err := proc.Signal(syscall.Signal(0)); err == nil {
							// Process is running
							if !config.Quiet {
								cmd.Printf("Daemon is already running with PID %d.\n", pid)
							}
							return // Exit, daemon is already running
						}
					}
				}
			}
			// If we reach here, PID file exists but process is not running or PID is invalid.
			// Clean up the stale PID file.
			if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
				utils.Log.Error("Failed to remove stale PID file: %v", err)
			}
			utils.Log.Info("Removed stale PID file: %s", pidFile)
			if !config.Quiet {
				cmd.Println("Found stale PID file, removed it. Attempting to start daemon.")
			}
		}

		// Start the daemon in the background
		exe, err := os.Executable()
		if err != nil {
			cmd.Printf("Error getting executable: %v\n", err)
			return
		}
		// Pass --foreground to the child process so it runs in foreground mode
		// and doesn't try to fork again. Se copia el slice para no mutar os.Args.
		procArgs := make([]string, 0, len(os.Args))
		for _, arg := range os.Args[1:] {
			if arg != "--foreground" {
				procArgs = append(procArgs, arg)
			}
		}
		procArgs = append(procArgs, "--foreground")
		command := exec.Command(exe, procArgs...) //nolint:gosec // Starting self, not a security issue.
		detachProcess(command)

		if err := command.Start(); err != nil {
			cmd.PrintErrf("Error starting daemon: %v\n", err)
			return
		}
		pid := command.Process.Pid
		_ = command.Process.Release()
		if !config.Quiet {
			cmd.Printf("Daemon started in background with PID %d\n", pid)
		}
		return
	}

	// This block runs when --foreground is true (i.e., the background process itself).
	pidFile := getPidFilePath()
	if err := acquirePidFile(pidFile); err != nil {
		cmd.PrintErrf("Error: %v\n", err)
		return
	}
	defer func() {
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			utils.Log.Error("Failed to remove PID file on exit: %v", err)
		}
		utils.Log.Info("Daemon stopped, PID file removed.")
	}()

	pid := os.Getpid()

	if !config.Quiet { // This message is for the foreground process itself
		cmd.Printf("Daemon started with PID %d\n", pid)
	}
	utils.Log.Info("Daemon started with PID %d", pid)

	cleanupLogs()

	// Cleanup any stale control files on start
	if err := os.Remove(getStopFilePath()); err != nil && !os.IsNotExist(err) {
		utils.Log.Error("Failed to remove stale stop file: %v", err)
	}
	if err := os.Remove(getPauseFilePath()); err != nil && !os.IsNotExist(err) {
		utils.Log.Error("Failed to remove stale pause file: %v", err)
	}

	// Print configuration details
	cfg, err := loadConfig()
	if err != nil {
		cmd.Printf("Warning: Failed to load config: %v\n", err)
	} else {
		// Si no se ha especificado el flag --interval, usar el valor de config.json
		if !cmd.Flags().Changed("interval") && cfg.Behavior.ChangeInterval > 0 {
			daemonInterval = cfg.Behavior.ChangeInterval
		}

		if !cmd.Flags().Changed("from-favorites") {
			daemonFromFavorites = cfg.Behavior.FromFavorites
		}

		changer := core.NewWallpaperChanger("", cfg.Behavior.RespectDarkMode)
		monitors, _ := changer.DetectMonitors()

		themeDisplay := daemonTheme
		if themeDisplay == "" && cfg.Behavior.RespectDarkMode {
			themeDisplay = "Auto (System)"
		} else if themeDisplay == "" {
			themeDisplay = "None"
		}

		if !config.Quiet {
			cmd.Println("--- Daemon Configuration ---")
			cmd.Printf("Interval: %d minutes\n", daemonInterval)
			cmd.Printf("Multi-Monitor Mode: %s\n", cfg.Behavior.MultiMonitor)
			cmd.Printf("Detected Monitors: %d\n", len(monitors))
			cmd.Printf("Theme Filter: %s\n", themeDisplay)
			cmd.Printf("From Favorites: %v\n", daemonFromFavorites)
			cmd.Printf("Transition: %s (engine=%s, linger=%dms)\n", cfg.Behavior.Transition.Type, cfg.Behavior.Transition.Engine, cfg.Behavior.Transition.Linger)
			cmd.Println("----------------------------")
		}
	}

	// time.NewTicker hace panic con un intervalo <= 0. Validarlo aquí evita un
	// crash-loop bajo systemd (Restart=on-failure).
	if daemonInterval <= 0 || daemonInterval > 24*60 {
		cmd.PrintErrf("Error: interval must be between 1 and %d minutes, got %d\n", 24*60, daemonInterval)
		return
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(sigChan)

	changeIntervalTicker := time.NewTicker(time.Duration(daemonInterval) * time.Minute)
	defer changeIntervalTicker.Stop()

	controlTicker := time.NewTicker(2 * time.Second)
	defer controlTicker.Stop()

	paused := false

	changeWallpaper(cmd)

	for {
		select {
		case sig := <-sigChan:
			if !config.Quiet {
				cmd.Printf("Received signal %v, stopping daemon...\n", sig)
			}
			utils.Log.Info("Received signal %v, stopping daemon...", sig)
			return

		case <-controlTicker.C:
			if _, err := os.Stat(getStopFilePath()); err == nil {
				if !config.Quiet {
					cmd.Println("Stopping daemon...")
				}
				utils.Log.Info("Stopping daemon...")
				if err := os.Remove(getStopFilePath()); err != nil {
					utils.Log.Error("Failed to remove stop file: %v", err)
				}
				return
			}

			if _, err := os.Stat(getPauseFilePath()); err == nil {
				if !paused {
					if !config.Quiet {
						cmd.Println("Daemon paused.")
					}
					utils.Log.Info("Daemon paused.")
					paused = true
				}
			} else {
				if paused {
					if !config.Quiet {
						cmd.Println("Daemon resumed.")
					}
					utils.Log.Info("Daemon resumed.")
					paused = false
				}
			}

		case <-changeIntervalTicker.C:
			if !paused {
				changeWallpaper(cmd)
			}
		}
	}
}

// resolveDaemonTransition aplica la precedencia flag de `daemon start` > config.
func resolveDaemonTransition(cfg *models.Config, cmd *cobra.Command) models.TransitionConfig {
	t := cfg.Behavior.Transition
	if cmd == nil {
		return t
	}
	if f := cmd.Flags().Lookup("transition-type"); f != nil && f.Changed {
		t.Type = daemonTransitionType
	}
	if f := cmd.Flags().Lookup("transition-duration"); f != nil && f.Changed {
		t.Duration = daemonTransitionDuration
	}
	if f := cmd.Flags().Lookup("transition-fps"); f != nil && f.Changed {
		t.FPS = daemonTransitionFPS
	}
	if f := cmd.Flags().Lookup("transition-step"); f != nil && f.Changed {
		t.Step = daemonTransitionStep
	}
	if f := cmd.Flags().Lookup("transition-angle"); f != nil && f.Changed {
		t.Angle = daemonTransitionAngle
	}
	if f := cmd.Flags().Lookup("transition-pos"); f != nil && f.Changed {
		t.Pos = daemonTransitionPos
	}
	if f := cmd.Flags().Lookup("transition-bezier"); f != nil && f.Changed {
		t.Bezier = daemonTransitionBezier
	}
	if f := cmd.Flags().Lookup("transition-wave"); f != nil && f.Changed {
		t.Wave = daemonTransitionWave
	}
	if f := cmd.Flags().Lookup("transition-engine"); f != nil && f.Changed {
		t.Engine = daemonTransitionEngine
	}
	if f := cmd.Flags().Lookup("transition-linger"); f != nil && f.Changed {
		t.Linger = daemonTransitionLinger
	}
	if f := cmd.Flags().Lookup("transition-mask"); f != nil && f.Changed {
		t.Mask = daemonTransitionMask
	}
	if f := cmd.Flags().Lookup("transition-feather"); f != nil && f.Changed {
		t.Feather = daemonTransitionFeather
	}
	if f := cmd.Flags().Lookup("transition-invert"); f != nil && f.Changed {
		t.Invert = daemonTransitionInvert
	}
	if f := cmd.Flags().Lookup("transition-layer"); f != nil && f.Changed {
		t.Layer = daemonTransitionLayer
	}
	return t
}

func changeWallpaper(cmd *cobra.Command) {
	cfg, err := loadConfig()
	if err != nil {
		utils.Log.Error("Daemon failed to load config: %v", err)
		if cmd != nil {
			cmd.Printf("Error loading config: %v\n", err)
		}
		return
	}
	controller := core.NewController(cfg)

	targetTheme := daemonTheme
	if targetTheme == "" {
		targetTheme = cfg.Behavior.Theme
	}
	if targetTheme == "auto" {
		targetTheme = ""
	}
	if targetTheme == "" && cfg.Behavior.RespectDarkMode {
		if core.IsSystemInDarkMode() {
			targetTheme = "dark"
		} else {
			targetTheme = "light"
		}
	}

	// 1. Prepare Monitor Configuration
	changer := core.NewWallpaperChanger("", cfg.Behavior.RespectDarkMode)
	changer.Transition = resolveDaemonTransition(cfg, cmd)
	if config.Debug {
		utils.Log.Info("Daemon Debug: Desktop Environment detected: %s", changer.Env)
	}
	var monitors []core.Monitor
	mmMode := cfg.Behavior.MultiMonitor

	if mmMode == "distinct" {
		var err error
		// Retry detection a few times to be robust against transient failures
		for i := 0; i < 3; i++ {
			monitors, err = changer.DetectMonitors()
			if err == nil && len(monitors) > 0 {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}

		if err != nil || len(monitors) == 0 {
			utils.Log.Error("Daemon: Failed to detect monitors for distinct mode (err: %v, count: %d). Falling back to clone.", err, len(monitors))
			mmMode = "clone"
			monitors = []core.Monitor{{ID: "default", Name: "default", Primary: true}}
		} else {
			utils.Log.Debug("Daemon: Detected %d monitors for distinct mode", len(monitors))
		}
	}

	needed := 1
	if mmMode == "distinct" && len(monitors) > 0 {
		needed = len(monitors)
	}

	// 2. Get Wallpapers (from favorites or feed)
	var selectedWallpapers []models.Wallpaper
	for i := 0; i < needed; i++ {
		var wallpaper models.Wallpaper
		var err error

		if daemonFromFavorites {
			favorites, err := loadFavorites()
			if err != nil {
				utils.Log.Error("Daemon: Error loading favorites: %v", err)
				return
			}
			if len(favorites) == 0 {
				utils.Log.Info("Daemon: No favorites found to set.")
				return
			}
			fav := favorites[rand.Intn(len(favorites))] //nolint:gosec // Not security-critical, seeding is done in root.
			wallpaper = fav.Wallpaper
		} else {
			wallpaper, err = controller.GetRandomFromFeed(targetTheme)
			if err != nil {
				utils.Log.Error("Daemon: Error getting random wallpaper from feed: %v", err)
				return
			}
		}
		selectedWallpapers = append(selectedWallpapers, wallpaper)
	}

	// 3. Download wallpapers
	var selectedPaths []string
	for _, wp := range selectedWallpapers {
		path, err := controller.DownloadWallpaper(wp)
		if err != nil {
			utils.Log.Error("Daemon: Failed to download wallpaper %s: %v", wp.ID, err)
			continue // Skip this wallpaper if download fails
		}

		// Update feed with local path so it's persisted in feed.json
		wp.Path = path
		_ = controller.AddWallpaperToFeed(wp)

		// Also update favorites if present
		_ = controller.UpdateWallpaperPath(wp.ID, path)

		selectedPaths = append(selectedPaths, path)
		if config.Debug {
			lum := controller.ColorManager.GetLuminance(wp.Color)
			utils.Log.Info("Daemon Debug: Selected %s | Color: %s | Luminance: %.2f", wp.ID, wp.Color, lum)
		}
	}

	if len(selectedPaths) == 0 {
		utils.Log.Error("Daemon: No wallpapers could be successfully downloaded.")
		return
	}

	// Si fallaron algunas descargas, las listas de paths y monitores ya no
	// cuadran. Se cae a clone para no aplicar el fondo equivocado a un monitor.
	if len(selectedPaths) < needed {
		utils.Log.Info("Daemon: only %d of %d wallpapers downloaded; falling back to clone mode", len(selectedPaths), needed)
		mmMode = "clone"
		selectedPaths = selectedPaths[:1]
	}

	// 4. Apply Wallpapers
	if err := changer.SetWallpapers(selectedPaths, monitors, mmMode); err != nil {
		utils.Log.Error("Daemon failed to set wallpapers: %v", err)
		if cmd != nil {
			cmd.Printf("Error setting wallpapers: %v\n", err)
		}
	} else {
		utils.Log.Info("Daemon set %d wallpaper(s)", len(selectedPaths))
		if cmd != nil && !config.Quiet {
			cmd.Printf("[%s] Wallpaper changed: %d image(s) applied.\n", time.Now().Format("15:04:05"), len(selectedPaths))
			if config.Debug {
				for i, p := range selectedPaths {
					cmd.Printf("  Monitor %d: %s\n", i+1, filepath.Base(p))
				}
			}
		}

		// Update state so 'gower status' knows what's set
		if len(selectedWallpapers) > 0 {
			if state, err := loadState(); err == nil {
				ids := make([]string, 0, len(selectedWallpapers))
				for _, wp := range selectedWallpapers {
					ids = append(ids, wp.ID)
				}

				// Guardar el estado anterior para que 'gower set undo' funcione
				// también tras un cambio hecho por el daemon.
				state.PreviousWallpapers = append([]string(nil), state.CurrentWallpapers...)
				state.PreviousWallpaperID = state.CurrentWallpaperID

				state.CurrentWallpapers = ids
				if len(ids) > 0 {
					state.CurrentWallpaperID = ids[0]
				}

				if err := saveState(state); err != nil {
					utils.Log.Error("Failed to save state: %v", err)
				}
			}
		}
	}
}

func runDaemonStop(cmd *cobra.Command, args []string) {
	pidFile := getPidFilePath()
	pid, alive := daemonPID()

	if !alive {
		if !config.Quiet {
			cmd.Println("Daemon not running (no live pid file).")
		}
		// Limpiar el PID file obsoleto y los ficheros de control.
		_ = os.Remove(pidFile)
		if daemonForce {
			_ = os.Remove(getStopFilePath())
			_ = os.Remove(getPauseFilePath())
		}
		return
	}

	stopFile := getStopFilePath()
	if err := os.WriteFile(stopFile, []byte{}, 0644); err != nil {
		cmd.PrintErrf("Error creating stop signal: %v\n", err)
		return
	}
	if !config.Quiet {
		cmd.Println("Stop signal sent.")
	}

	if !daemonForce {
		return
	}

	// --force no debe dejar un daemon huérfano vivo: se escala a señales.
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Signal(syscall.SIGTERM)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if processAlive(pid) {
		if proc, err := os.FindProcess(pid); err == nil {
			_ = proc.Signal(syscall.SIGKILL)
		}
	}

	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		utils.Log.Error("Forced stop: failed to remove pid file: %v", err)
	}
	if !config.Quiet {
		cmd.Printf("Forced stop: daemon PID %d terminated and pid file removed.\n", pid)
	}
}

func runDaemonStatus(cmd *cobra.Command, args []string) {
	pid, running := daemonPID()
	_, pausedErr := os.Stat(getPauseFilePath())
	paused := pausedErr == nil

	type statusOutput struct {
		Running bool `json:"running"`
		PID     int  `json:"pid"`
		Paused  bool `json:"paused"`
	}
	status := statusOutput{Running: running, PID: pid, Paused: paused}

	if daemonJSON || config.JSONOutput {
		jsonOut, err := json.Marshal(status)
		if err != nil {
			cmd.PrintErrf("Error encoding status: %v\n", err)
			return
		}
		cmd.Println(string(jsonOut))
		return
	}

	if config.Quiet {
		return
	}
	if running {
		suffix := ""
		if paused {
			suffix = " [paused]"
		}
		cmd.Printf("Daemon is running (PID: %d)%s\n", pid, suffix)
	} else {
		cmd.Println("Daemon is stopped.")
	}
}

func runDaemonInstall(cmd *cobra.Command, args []string) {
	if runtime.GOOS != "linux" {
		cmd.PrintErrln("systemd integration is only supported on Linux.")
		return
	}

	if _, err := exec.LookPath("systemctl"); err != nil {
		cmd.PrintErrln("systemctl not found. Make sure systemd is installed.")
		return
	}

	exe, err := os.Executable()
	if err != nil {
		cmd.Printf("Error getting executable path: %v\n", err)
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		cmd.Printf("Error getting home directory: %v\n", err)
		return
	}

	serviceDir := filepath.Join(homeDir, ".config", "systemd", "user")
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		cmd.Printf("Error creating systemd user directory: %v\n", err)
		return
	}

	servicePath := filepath.Join(serviceDir, "gower.service")
	execStart := exe
	if strings.ContainsAny(execStart, " \t\"'\\") {
		execStart = strconv.Quote(execStart)
	}
	unit := fmt.Sprintf(`[Unit]
Description=Gower Wallpaper Daemon
Documentation=https://github.com/nefDevelop/gower
After=network.target graphical-session.target

[Service]
Type=simple
ExecStart=%s daemon start --foreground
Environment=PATH=%%h/.local/bin:/usr/local/bin:/usr/bin:/bin
Restart=on-failure
RestartSec=30
StartLimitIntervalSec=300
StartLimitBurst=5

[Install]
WantedBy=graphical-session.target
`, execStart)

	if err := os.WriteFile(servicePath, []byte(unit), 0644); err != nil {
		cmd.PrintErrf("Error writing service file: %v\n", err)
		return
	}

	utils.Log.Info("systemd service file created: %s", servicePath)

	// Reload systemd and enable+start
	for _, args := range [][]string{
		{"--user", "daemon-reload"},
		{"--user", "enable", "--now", "gower.service"},
	} {
		c := exec.Command("systemctl", args...)
		c.Stderr = os.Stderr
		c.Stdout = os.Stdout
		if err := c.Run(); err != nil {
			cmd.PrintErrf("Error: systemctl %s failed: %v\n", strings.Join(args, " "), err)
			return
		}
	}

	if !config.Quiet {
		cmd.Printf("Daemon installed as systemd user service.\n")
		cmd.Printf("Service file: %s\n", servicePath)
		cmd.Println("Status: systemctl --user status gower.service")
	}
}

func runDaemonUninstall(cmd *cobra.Command, args []string) {
	if runtime.GOOS != "linux" {
		cmd.PrintErrln("systemd integration is only supported on Linux.")
		return
	}

	if _, err := exec.LookPath("systemctl"); err != nil {
		cmd.PrintErrln("systemctl not found.")
		return
	}

	// Stop and disable
	c := exec.Command("systemctl", "--user", "disable", "--now", "gower.service")
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	if err := c.Run(); err != nil {
		cmd.PrintErrf("Warning: systemctl disable --now failed: %v\n", err)
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		servicePath := filepath.Join(homeDir, ".config", "systemd", "user", "gower.service")
		if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
			cmd.PrintErrf("Warning: failed to remove service file: %v\n", err)
		}
	}

	// Reload systemd and clear any failed state.
	for _, args := range [][]string{
		{"--user", "daemon-reload"},
		{"--user", "reset-failed", "gower.service"},
	} {
		if err := exec.Command("systemctl", args...).Run(); err != nil {
			utils.Log.Error("systemctl %s failed: %v", strings.Join(args, " "), err)
		}
	}

	if !config.Quiet {
		cmd.Println("Daemon systemd service removed.")
	}
}

func cleanupLogs() {
	cfg, err := loadConfig()
	if err != nil {
		return
	}
	days := cfg.Limits.LogRetentionDays
	if days <= 0 {
		days = 7
	}

	appDir, err := core.GetAppDir()
	if err != nil {
		return
	}
	logsDir := filepath.Join(appDir, "logs")

	files, err := os.ReadDir(logsDir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info()
			if err == nil && info.ModTime().Before(cutoff) {
				if err := os.Remove(filepath.Join(logsDir, file.Name())); err != nil {
					utils.Log.Error("Failed to remove old log file %s: %v", file.Name(), err)
				}
			}
		}
	}
}
