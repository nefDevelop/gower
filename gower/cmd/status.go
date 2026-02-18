package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"syscall"
	"text/tabwriter"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var (
	statusJSON      bool
	statusProviders bool
	statusStorage   bool
	statusDaemon    bool
	statusSystem    bool
	statusMonitors  bool
	statusWallpaper bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show application status",
	Long:  `Show status of providers, storage, daemon, and system information.`,
	Run:   runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output in JSON")
	statusCmd.Flags().BoolVar(&statusProviders, "providers", false, "Show providers status")
	statusCmd.Flags().BoolVar(&statusStorage, "storage", false, "Show storage usage")
	statusCmd.Flags().BoolVar(&statusDaemon, "daemon", false, "Show daemon status")
	statusCmd.Flags().BoolVar(&statusSystem, "system", false, "Show system information")
	statusCmd.Flags().BoolVar(&statusMonitors, "monitors", false, "Show monitor information")
	statusCmd.Flags().BoolVar(&statusWallpaper, "wallpapers", false, "Show current wallpaper information")
}

type StatusOutput struct {
	System    *SystemStatus           `json:"system,omitempty"`
	Daemon    *DaemonStatus           `json:"daemon,omitempty"`
	Providers *ProvidersStatus        `json:"providers,omitempty"`
	Storage   *StorageStatus          `json:"storage,omitempty"`
	Monitors  []core.Monitor          `json:"monitors,omitempty"`
	Wallpaper *CurrentWallpaperStatus `json:"wallpaper,omitempty"`
}

type SystemStatus struct {
	OS           string          `json:"os"`
	Arch         string          `json:"arch"`
	HomeDir      string          `json:"home_dir"`
	ConfigDir    string          `json:"config_dir"`
	DesktopEnv   string          `json:"desktop_env,omitempty"`
	Dependencies map[string]bool `json:"dependencies"`
}

type DaemonStatus struct {
	Running bool `json:"running"`
	PID     int  `json:"pid"`
}

type ProvidersStatus struct {
	Wallhaven bool            `json:"wallhaven"`
	Reddit    bool            `json:"reddit"`
	Nasa      bool            `json:"nasa"`
	Generic   map[string]bool `json:"generic"`
}

type StorageStatus struct {
	CacheSize string `json:"cache_size"`
	DataSize  string `json:"data_size"`
	TotalSize string `json:"total_size"`
}

type CurrentWallpaperStatus struct {
	Wallpapers []models.Wallpaper `json:"wallpapers,omitempty"`
}

func runStatus(cmd *cobra.Command, args []string) {
	// If no specific flag is set, show all
	showAll := !statusProviders && !statusStorage && !statusDaemon && !statusSystem && !statusMonitors && !statusWallpaper

	output := StatusOutput{}

	if showAll || statusSystem {
		output.System = getSystemStatus()
	}
	if showAll || statusDaemon {
		output.Daemon = getDaemonStatus()
	}
	if showAll || statusWallpaper {
		output.Wallpaper = getWallpaperStatus()
	}
	if showAll || statusProviders {
		output.Providers = getProvidersStatus()
	}
	if showAll || statusStorage {
		output.Storage = getStorageStatus()
	}
	if showAll || statusMonitors {
		// Need a WallpaperChanger instance to detect monitors
		changer := core.NewWallpaperChanger("", false) // RespectDarkMode doesn't matter for detection
		monitors, err := changer.DetectMonitors()
		if err != nil {
			cmd.Printf("Error detecting monitors: %v\n", err)
		} else {
			output.Monitors = monitors
		}
	}

	if statusJSON {
		data, _ := json.MarshalIndent(output, "", "  ")
		cmd.Println(string(data))
		return
	}

	// Helper to create a new tabwriter
	newTabWriter := func() *tabwriter.Writer {
		return tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	}

	// Text output
	if output.System != nil {
		cmd.Println("--- System ---")
		w := newTabWriter()
		_, _ = fmt.Fprintf(w, "OS:\t%s\n", output.System.OS)
		_, _ = fmt.Fprintf(w, "Arch:\t%s\n", output.System.Arch)
		_, _ = fmt.Fprintf(w, "Desktop Environment:\t%s\n", output.System.DesktopEnv)
		_, _ = fmt.Fprintf(w, "Home:\t%s\n", output.System.HomeDir)
		_ = w.Flush()
		cmd.Println("Dependencies:")
		// Sort keys for consistent output
		depKeys := make([]string, 0, len(output.System.Dependencies))
		for k := range output.System.Dependencies {
			depKeys = append(depKeys, k)
		}
		sort.Strings(depKeys)

		w = newTabWriter()
		for _, dep := range depKeys {
			installed := output.System.Dependencies[dep]
			status := colorize("Not Found", colorRed)
			if installed {
				status = colorize("Installed", colorGreen)
			}
			_, _ = fmt.Fprintf(w, "  %s:\t%s\n", dep, status)
		}
		_ = w.Flush()
		cmd.Println()
	}

	if output.Daemon != nil {
		cmd.Println("--- Daemon ---")
		state := colorize("Stopped", colorRed)
		if output.Daemon.Running {
			state = colorize(fmt.Sprintf("Running (PID: %d)", output.Daemon.PID), colorGreen)
		}
		w := newTabWriter()
		_, _ = fmt.Fprintf(w, "Status:\t%s\n", state)
		_ = w.Flush()
		cmd.Println()
	}

	if output.Wallpaper != nil && len(output.Wallpaper.Wallpapers) > 0 {
		cmd.Println("--- Wallpaper ---")
		w := newTabWriter()
		for i, wp := range output.Wallpaper.Wallpapers {
			_, _ = fmt.Fprintf(w, "Monitor %d ID:\t%s\n", i+1, wp.ID)
			_, _ = fmt.Fprintf(w, "Monitor %d Path:\t%s\n", i+1, wp.Path)
			_, _ = fmt.Fprintf(w, "Monitor %d Source:\t%s\n", i+1, wp.Source)
			_, _ = fmt.Fprintf(w, "Monitor %d URL:\t%s\n", i+1, wp.URL)
			_, _ = fmt.Fprintf(w, "Monitor %d Dimension:\t%s\n", i+1, wp.Dimension)
			_, _ = fmt.Fprintf(w, "Monitor %d Color:\t%s\n", i+1, wp.Color)
			_, _ = fmt.Fprintf(w, "Monitor %d Theme:\t%s\n", i+1, wp.Theme)
		}
		_ = w.Flush()
		cmd.Println()
	} else if output.Wallpaper != nil {
		cmd.Println("--- Wallpaper ---")
		cmd.Println("  No wallpapers currently set.")
		cmd.Println()
	}

	if output.Providers != nil {
		cmd.Println("--- Providers ---")
		w := newTabWriter()
		_, _ = fmt.Fprintf(w, "Wallhaven:\t%v\n", colorizeBool(output.Providers.Wallhaven))
		_, _ = fmt.Fprintf(w, "Reddit:\t%v\n", colorizeBool(output.Providers.Reddit))
		_, _ = fmt.Fprintf(w, "Nasa:\t%v\n", colorizeBool(output.Providers.Nasa))
		_ = w.Flush()
		if len(output.Providers.Generic) > 0 {
			cmd.Println("Manual Providers:")
			keys := make([]string, 0, len(output.Providers.Generic))
			for k := range output.Providers.Generic {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			w = newTabWriter()
			for _, name := range keys {
				_, _ = fmt.Fprintf(w, "  %s:\t%v\n", name, colorizeBool(output.Providers.Generic[name]))
			}
			_ = w.Flush()
		}
		cmd.Println()
	}

	if output.Storage != nil {
		cmd.Println("--- Storage ---")
		w := newTabWriter()
		_, _ = fmt.Fprintf(w, "Cache:\t%s\n", output.Storage.CacheSize)
		_, _ = fmt.Fprintf(w, "Data:\t%s\n", output.Storage.DataSize)
		_, _ = fmt.Fprintf(w, "Total:\t%s\n", output.Storage.TotalSize)
		_ = w.Flush()
		cmd.Println()
	}

	if len(output.Monitors) > 0 {
		cmd.Println("--- Monitors ---")
		for i, mon := range output.Monitors {
			primary := ""
			if mon.Primary {
				primary = " (Primary)"
			}
			cmd.Printf("  Monitor %d: %s%s\n", i+1, mon.Name, primary)
			w := newTabWriter()
			_, _ = fmt.Fprintf(w, "    Resolution:\t%dx%d\n", mon.Width, mon.Height)
			_, _ = fmt.Fprintf(w, "    Position:\t%d,%d\n", mon.X, mon.Y)
			_ = w.Flush()
		}
		cmd.Println()
	}
}

func colorizeBool(b bool) string {
	if b {
		return colorize("true", colorGreen)
	}
	return colorize("false", colorRed)
}

func getSystemStatus() *SystemStatus {
	home, _ := os.UserHomeDir()
	configDir, _ := core.GetAppDir()

	deps := make(map[string]bool)
	deps["feh"] = checkCommand("feh")
	deps["nitrogen"] = checkCommand("nitrogen")
	deps["gsettings"] = checkCommand("gsettings")
	deps["dbus-send"] = checkCommand("dbus-send")
	deps["swaybg"] = checkCommand("swaybg")
	deps["quickshell"] = checkCommand("quickshell") // For Dank Material Shell (dms)
	deps["niri"] = checkCommand("niri")
	deps["matugen"] = checkCommand("matugen")
	deps["swww"] = checkCommand("swww")

	return &SystemStatus{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		HomeDir:      home,
		ConfigDir:    configDir,
		DesktopEnv:   core.DetectDesktopEnv(),
		Dependencies: deps,
	}
}

func checkCommand(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func getDaemonStatus() *DaemonStatus {
	pidFile := getPidFilePath() // Defined in daemon.go
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
	return &DaemonStatus{Running: running, PID: pid}
}

func getWallpaperStatus() *CurrentWallpaperStatus {
	state, err := loadState()
	if err != nil {
		// If state can't be loaded, we can't determine wallpaper status.
		_, _ = fmt.Fprintf(os.Stderr, "Error loading state for wallpaper status: %v\n", err)
		return &CurrentWallpaperStatus{Wallpapers: []models.Wallpaper{}}
	}

	var wallpapers []models.Wallpaper
	cfg, err := loadConfig()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error loading config for wallpaper status: %v\n", err)
		return nil
	}
	controller := core.NewController(cfg)

	// If multiple wallpapers are set (for multiple monitors)
	if len(state.CurrentWallpapers) > 0 {
		for _, id := range state.CurrentWallpapers {
			wp, err := controller.GetWallpaper(id)
			if err != nil {
				// Log error but continue with other wallpapers
				_, _ = fmt.Fprintf(os.Stderr, "Error retrieving wallpaper %s: %v\n", id, err)
				continue
			}
			if wp != nil {
				wallpapers = append(wallpapers, *wp)
			}
		}
	} else if state.CurrentWallpaperID != "" {
		// Fallback for single wallpaper (older state format or single monitor setup)
		wp, err := controller.GetWallpaper(state.CurrentWallpaperID)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error retrieving wallpaper %s: %v\n", state.CurrentWallpaperID, err)
		}
		if wp != nil {
			wallpapers = append(wallpapers, *wp)
		}
	}

	return &CurrentWallpaperStatus{
		Wallpapers: wallpapers,
	}
}

func getProvidersStatus() *ProvidersStatus {
	cfg, err := loadConfig()
	if err != nil {
		return nil
	}

	generic := make(map[string]bool)
	for _, p := range cfg.GenericProviders {
		generic[p.Name] = p.Enabled
	}

	return &ProvidersStatus{
		Wallhaven: cfg.Providers.Wallhaven.Enabled,
		Reddit:    cfg.Providers.Reddit.Enabled,
		Nasa:      cfg.Providers.Nasa.Enabled,
		Generic:   generic,
	}
}

func getStorageStatus() *StorageStatus {
	baseDir, _ := core.GetAppDir()

	cacheSize := getDirSize(filepath.Join(baseDir, "cache"))
	dataSize := getDirSize(filepath.Join(baseDir, "data"))

	return &StorageStatus{
		CacheSize: formatBytes(cacheSize),
		DataSize:  formatBytes(dataSize),
		TotalSize: formatBytes(cacheSize + dataSize),
	}
}

func getDirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
