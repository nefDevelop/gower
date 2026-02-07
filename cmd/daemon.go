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

	wallpapers, err := controller.GetCachedWallpapers(daemonFromFavorites, targetTheme)
	if err != nil || len(wallpapers) == 0 {
		utils.Log.Debug("Daemon: No cached wallpapers found (err: %v)", err)
		return
	}

	wp := wallpapers[rand.Intn(len(wallpapers))]

	path, _ := controller.GetWallpaperLocalPath(wp)
	changer := core.NewWallpaperChanger("", cfg.Behavior.RespectDarkMode)
	if err := changer.SetWallpapers([]string{path}, []core.Monitor{}, cfg.Behavior.MultiMonitor); err != nil {
		utils.Log.Error("Daemon failed to set wallpaper %s: %v", wp.ID, err)
	} else {
		utils.Log.Info("Daemon set wallpaper: %s", wp.ID)
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
