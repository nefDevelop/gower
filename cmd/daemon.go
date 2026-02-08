package cmd

import (
	"encoding/json"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
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
	Run:   func(cmd *cobra.Command, args []string) { sendSignal(cmd, syscall.SIGUSR1, "paused") },
}

var daemonResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume the daemon",
	Run:   func(cmd *cobra.Command, args []string) { sendSignal(cmd, syscall.SIGUSR2, "resumed") },
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

// init initializes the random seed once
func init() {
	rand.Seed(time.Now().UnixNano())
}

func getPidFilePath() string {
	appDir, _ := core.GetAppDir()
	return filepath.Join(appDir, "gower.pid")
}

func runDaemonStart(cmd *cobra.Command, args []string) {
	if !daemonForeground {
		pidFile := getPidFilePath()
		if _, err := os.Stat(pidFile); err == nil {
			if !config.Quiet {
				cmd.Println("Daemon appears to be running (pid file exists). Use stop or force.")
			}
			return
		}

		exe, err := os.Executable()
		if err != nil {
			cmd.Printf("Error getting executable: %v\n", err)
			return
		}

		procArgs := append(os.Args[1:], "--foreground")
		command := exec.Command(exe, procArgs...)

		if err := command.Start(); err != nil {
			cmd.Printf("Error starting daemon: %v\n", err)
			return
		}
		if !config.Quiet {
			cmd.Printf("Daemon started in background with PID %d\n", command.Process.Pid)
		}
		return
	}

	pidFile := getPidFilePath()
	if _, err := os.Stat(pidFile); err == nil {
		if !config.Quiet {
			cmd.Println("Daemon appears to be running (pid file exists). Use stop or force.")
		}
		return
	}

	pid := os.Getpid()
	os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
	defer os.Remove(pidFile)

	if !config.Quiet {
		cmd.Printf("Daemon started with PID %d\n", pid)
	}
	utils.Log.Info("Daemon started with PID %d", pid)

	cleanupLogs()

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

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

	ticker := time.NewTicker(time.Duration(daemonInterval) * time.Minute)
	defer ticker.Stop()

	paused := false

	// Initial run
	changeWallpaper(cmd)

	for {
		select {
		case sig := <-sigs:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				if !config.Quiet {
					cmd.Println("Stopping daemon...")
				}
				utils.Log.Info("Stopping daemon...")
				return
			case syscall.SIGUSR1:
				if !config.Quiet {
					cmd.Println("Daemon paused.")
				}
				utils.Log.Info("Daemon paused.")
				paused = true
			case syscall.SIGUSR2:
				if !config.Quiet {
					cmd.Println("Daemon resumed.")
				}
				utils.Log.Info("Daemon resumed.")
				paused = false
				changeWallpaper(cmd)
			}
		case <-ticker.C:
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

	// 2. Get Cached Wallpapers
	cached, err := controller.GetCachedWallpapers(daemonFromFavorites, targetTheme)
	if err != nil || len(cached) == 0 {
		utils.Log.Info("Daemon: No cached wallpapers found. Run 'gower set random' or 'gower download' first.")
		return
	}

	// 3. Select Random Wallpapers
	rand.Shuffle(len(cached), func(i, j int) { cached[i], cached[j] = cached[j], cached[i] })

	var selectedPaths []string
	var selectedWallpapers []models.Wallpaper
	count := needed
	if len(cached) < count {
		count = len(cached)
	}
	for i := 0; i < count; i++ {
		path, _ := controller.GetWallpaperLocalPath(cached[i])
		selectedPaths = append(selectedPaths, path)
		selectedWallpapers = append(selectedWallpapers, cached[i])
		if config.Debug {
			lum := controller.ColorManager.GetLuminance(cached[i].Color)
			utils.Log.Info("Daemon Debug: Selected %s | Color: %s | Luminance: %.2f", cached[i].ID, cached[i].Color, lum)
		}
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
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if !config.Quiet {
			cmd.Println("Daemon not running (no pid file).")
		}
		return
	}
	pid, _ := strconv.Atoi(string(data))

	proc, err := os.FindProcess(pid)
	if err == nil {
		proc.Signal(syscall.SIGTERM)
		if !config.Quiet {
			cmd.Println("Stop signal sent.")
		}
	}
	if daemonForce {
		os.Remove(pidFile)
	}
}

func runDaemonStatus(cmd *cobra.Command, args []string) {
	pidFile := getPidFilePath()
	data, err := os.ReadFile(pidFile)
	running := false
	pid := 0

	if err == nil {
		pid, _ = strconv.Atoi(string(data))
		proc, err := os.FindProcess(pid)
		if err == nil {
			if err := proc.Signal(syscall.Signal(0)); err == nil {
				running = true
			}
		}
	}

	if daemonJSON {
		status := map[string]interface{}{
			"running": running,
			"pid":     pid,
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

func sendSignal(cmd *cobra.Command, sig syscall.Signal, action string) {
	pidFile := getPidFilePath()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if !config.Quiet {
			cmd.Println("Daemon not running.")
		}
		return
	}
	pid, _ := strconv.Atoi(string(data))
	proc, err := os.FindProcess(pid)
	if err == nil {
		proc.Signal(sig)
		if !config.Quiet {
			cmd.Printf("Daemon %s.\n", action)
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
				os.Remove(filepath.Join(logsDir, file.Name()))
			}
		}
	}
}
