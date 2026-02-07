package cmd

import (
	"encoding/json"
	"math/rand"
	"os"
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

	daemonStopCmd.Flags().BoolVar(&daemonForce, "force", false, "Force stop")
	daemonStatusCmd.Flags().BoolVar(&daemonJSON, "json", false, "Output in JSON")
}

// init initializes the random seed once
func init() {
	rand.Seed(time.Now().UnixNano())
}

func getPidFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gower", "gower.pid")
}

func runDaemonStart(cmd *cobra.Command, args []string) {
	pidFile := getPidFilePath()
	if _, err := os.Stat(pidFile); err == nil {
		cmd.Println("Daemon appears to be running (pid file exists). Use stop or force.")
		return
	}

	pid := os.Getpid()
	os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
	defer os.Remove(pidFile)

	cmd.Printf("Daemon started with PID %d\n", pid)
	utils.Log.Info("Daemon started with PID %d", pid)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

	ticker := time.NewTicker(time.Duration(daemonInterval) * time.Minute)
	defer ticker.Stop()

	paused := false

	// Initial run
	changeWallpaper()

	for {
		select {
		case sig := <-sigs:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				cmd.Println("Stopping daemon...")
				utils.Log.Info("Stopping daemon...")
				return
			case syscall.SIGUSR1:
				cmd.Println("Daemon paused.")
				utils.Log.Info("Daemon paused.")
				paused = true
			case syscall.SIGUSR2:
				cmd.Println("Daemon resumed.")
				utils.Log.Info("Daemon resumed.")
				paused = false
				changeWallpaper()
			}
		case <-ticker.C:
			if !paused {
				changeWallpaper()
			}
		}
	}
}

func changeWallpaper() {
	cfg, err := loadConfig()
	if err != nil {
		utils.Log.Error("Daemon failed to load config: %v", err)
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
		monitors, err = changer.DetectMonitors()
		if err != nil {
			utils.Log.Error("Daemon: Failed to detect monitors for distinct mode: %v. Falling back to clone.", err)
			mmMode = "clone"
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
	} else {
		utils.Log.Info("Daemon set %d wallpaper(s)", len(selectedPaths))

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
		cmd.Println("Daemon not running (no pid file).")
		return
	}
	pid, _ := strconv.Atoi(string(data))

	proc, err := os.FindProcess(pid)
	if err == nil {
		proc.Signal(syscall.SIGTERM)
		cmd.Println("Stop signal sent.")
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
	} else {
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
		cmd.Println("Daemon not running.")
		return
	}
	pid, _ := strconv.Atoi(string(data))
	proc, err := os.FindProcess(pid)
	if err == nil {
		proc.Signal(sig)
		cmd.Printf("Daemon %s.\n", action)
	}
}
