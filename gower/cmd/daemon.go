package cmd

import (
	"encoding/json"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv" // Import syscall for process signal check
	"time"

	"gower/internal/core"
	"gower/internal/utils"
	"gower/pkg/models"

	"syscall"

	"github.com/spf13/cobra"
)

var (
	daemonInterval      int
	daemonFromFavorites bool
	daemonTheme         string
	daemonForce         bool
	daemonJSON          bool
	daemonForeground    bool
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
		pidFile := getPidFilePath()
		if _, err := os.Stat(pidFile); os.IsNotExist(err) {
			if !config.Quiet {
				cmd.Println("Daemon not running.")
			}
			return
		}
		pauseFile := getPauseFilePath()
		if f, err := os.Create(pauseFile); err != nil {
			cmd.Printf("Error creating pause signal: %v\n", err)
		} else {
			_ = f.Close()
			if !config.Quiet {
				cmd.Println("Daemon pause signal sent.")
			}
		}
	},
}

var daemonResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume the daemon",
	Run: func(cmd *cobra.Command, args []string) {
		pidFile := getPidFilePath()
		if _, err := os.Stat(pidFile); os.IsNotExist(err) {
			if !config.Quiet {
				cmd.Println("Daemon not running.")
			}
			return
		}
		pauseFile := getPauseFilePath()
		if err := os.Remove(pauseFile); err != nil {
			if !os.IsNotExist(err) {
				cmd.Printf("Error removing pause signal: %v\n", err)
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

	daemonStartCmd.Flags().IntVar(&daemonInterval, "interval", 30, "Interval in minutes")
	daemonStartCmd.Flags().BoolVar(&daemonFromFavorites, "from-favorites", false, "Include favorites")
	daemonStartCmd.Flags().StringVar(&daemonTheme, "theme", "", "Filter by theme")
	daemonStartCmd.Flags().BoolVar(&daemonForeground, "foreground", false, "Run in foreground (blocking)")

	daemonStopCmd.Flags().BoolVar(&daemonForce, "force", false, "Force stop")
	daemonStatusCmd.Flags().BoolVar(&daemonJSON, "json", false, "Output in JSON")
}

// init initializes randomness
func init() {
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
		// and doesn't try to fork again.
		procArgs := append(os.Args[1:], "--foreground")
		command := exec.Command(exe, procArgs...) //nolint:gosec // Starting self, not a security issue.

		if err := command.Start(); err != nil {
			cmd.Printf("Error starting daemon: %v\n", err)
			return
		}
		if !config.Quiet {
			cmd.Printf("Daemon started in background with PID %d\n", command.Process.Pid)
		}
		return
	}

	// This block runs when --foreground is true (i.e., the background process itself)
	pidFile := getPidFilePath()
	// Double check if PID file exists and process is running, in case of race condition
	if _, err := os.Stat(pidFile); err == nil { // PID file exists
		if !config.Quiet {
			cmd.Println("Checking for existing daemon process in foreground mode...")
		}
		pid := os.Getpid()
		pidData, readErr := os.ReadFile(pidFile)
		if readErr == nil {
			existingPid, _ := strconv.Atoi(string(pidData))
			if existingPid > 0 && existingPid != pid { // If PID file points to a different running process
				proc, findErr := os.FindProcess(existingPid)
				if findErr == nil {
					if err := proc.Signal(syscall.Signal(0)); err == nil {
						// Another daemon is already running, this is a duplicate foreground start
						if !config.Quiet {
							cmd.Printf("Another daemon is already running with PID %d. Exiting foreground start.\n", existingPid)
						}
						return
					}
				}
			}
		}
		// If PID file exists but process is not running, or it was our own stale PID, remove it.
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			utils.Log.Error("Failed to remove stale PID file in foreground: %v", err)
		}
	}

	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		cmd.Printf("Error writing PID file: %v\n", err)
		return
	}
	defer func() {
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			utils.Log.Error("Failed to remove PID file on exit: %v", err)
		}
		utils.Log.Info("Daemon stopped, PID file removed.")
	}()

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
			cmd.Println("----------------------------")
		}
	}

	changeIntervalTicker := time.NewTicker(time.Duration(daemonInterval) * time.Minute)
	defer changeIntervalTicker.Stop()

	// Ticker to check for control signals (pause, stop)
	controlTicker := time.NewTicker(2 * time.Second)
	defer controlTicker.Stop()

	paused := false

	// Initial run
	changeWallpaper(cmd)

	for {
		select {
		case <-controlTicker.C:
			// Check for stop signal
			if _, err := os.Stat(getStopFilePath()); err == nil {
				if !config.Quiet {
					cmd.Println("Stopping daemon...")
				}
				utils.Log.Info("Stopping daemon...")
				os.Remove(getStopFilePath()) // Clean up
				return
			}

			// Check for pause signal
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
	if targetTheme == "" && cfg.Behavior.RespectDarkMode {
		if core.IsSystemInDarkMode() {
			targetTheme = "dark"
		} else {
			targetTheme = "light"
		}
	}

	// 1. Prepare Monitor Configuration
	changer := core.NewWallpaperChanger("", cfg.Behavior.RespectDarkMode)
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
				var ids []string
				for _, wp := range selectedWallpapers {
					ids = append(ids, wp.ID)
				}
				state.CurrentWallpapers = ids

				if state.CurrentWallpaperID != selectedWallpapers[0].ID {
					state.PreviousWallpaperID = state.CurrentWallpaperID
					state.CurrentWallpaperID = selectedWallpapers[0].ID
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
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		if !config.Quiet {
			cmd.Println("Daemon not running (no pid file).")
		}
		// If forcing, try to remove control files anyway
		if daemonForce {
			if err := os.Remove(getStopFilePath()); err != nil && !os.IsNotExist(err) {
				utils.Log.Error("Forced stop: failed to remove stop file: %v", err)
			}
			if err := os.Remove(getPauseFilePath()); err != nil && !os.IsNotExist(err) {
				utils.Log.Error("Forced stop: failed to remove pause file: %v", err)
			}
		}
		return
	}

	// Create stop file
	stopFile := getStopFilePath()
	if f, err := os.Create(stopFile); err != nil {
		cmd.Printf("Error creating stop signal: %v\n", err)
	} else {
		_ = f.Close() // Error on close is not critical here.
		if !config.Quiet {
			cmd.Println("Stop signal sent.")
		}
	}

	if daemonForce {
		// Also remove pid file to allow immediate restart
		if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
			utils.Log.Error("Forced stop: failed to remove pid file: %v", err)
		}
		if !config.Quiet {
			cmd.Println("Forcing stop: removed pid file.")
		}
	}
}

func runDaemonStatus(cmd *cobra.Command, args []string) {
	pidFile := getPidFilePath()
	data, err := os.ReadFile(pidFile)
	running := false
	pid := 0

	// The presence of the PID file is our primary indicator.
	// The daemon is responsible for cleaning it up on exit.
	if err == nil {
		running = true
		pid, _ = strconv.Atoi(string(data))
	}

	if daemonJSON {
		status := map[string]interface{}{
			"running": running,
			"pid":     pid, // Will be 0 if not running
		}
		jsonOut, _ := json.Marshal(status)
		cmd.Println(string(jsonOut))
	} else if !config.Quiet {
		if running {
			cmd.Printf("Daemon is running (PID: %d)\n", pid)
		} else {
			cmd.Println("Daemon is stopped.")
		}
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
