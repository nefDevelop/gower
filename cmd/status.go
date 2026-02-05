package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"syscall"

	"gower/internal/core"

	"github.com/spf13/cobra"
)

var (
	statusJSON      bool
	statusProviders bool
	statusStorage   bool
	statusDaemon    bool
	statusSystem    bool
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
}

type StatusOutput struct {
	System    *SystemStatus    `json:"system,omitempty"`
	Daemon    *DaemonStatus    `json:"daemon,omitempty"`
	Providers *ProvidersStatus `json:"providers,omitempty"`
	Storage   *StorageStatus   `json:"storage,omitempty"`
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

func runStatus(cmd *cobra.Command, args []string) {
	// If no specific flag is set, show all
	showAll := !statusProviders && !statusStorage && !statusDaemon && !statusSystem

	output := StatusOutput{}

	if showAll || statusSystem {
		output.System = getSystemStatus()
	}
	if showAll || statusDaemon {
		output.Daemon = getDaemonStatus()
	}
	if showAll || statusProviders {
		output.Providers = getProvidersStatus()
	}
	if showAll || statusStorage {
		output.Storage = getStorageStatus()
	}

	if statusJSON {
		data, _ := json.MarshalIndent(output, "", "  ")
		cmd.Println(string(data))
		return
	}

	// Text output
	if output.System != nil {
		cmd.Println("--- System ---")
		cmd.Printf("OS: %s\n", output.System.OS)
		cmd.Printf("Arch: %s\n", output.System.Arch)
		cmd.Printf("Desktop Environment: %s\n", output.System.DesktopEnv)
		cmd.Printf("Home: %s\n", output.System.HomeDir)
		cmd.Println("Dependencies:")
		// Sort keys for consistent output
		depKeys := make([]string, 0, len(output.System.Dependencies))
		for k := range output.System.Dependencies {
			depKeys = append(depKeys, k)
		}
		sort.Strings(depKeys)

		for _, dep := range depKeys {
			installed := output.System.Dependencies[dep]
			status := "Not Found"
			if installed {
				status = "Installed"
			}
			cmd.Printf("  %s: %s\n", dep, status)
		}
		cmd.Println()
	}

	if output.Daemon != nil {
		cmd.Println("--- Daemon ---")
		state := "Stopped"
		if output.Daemon.Running {
			state = fmt.Sprintf("Running (PID: %d)", output.Daemon.PID)
		}
		cmd.Printf("Status: %s\n", state)
		cmd.Println()
	}

	if output.Providers != nil {
		cmd.Println("--- Providers ---")
		cmd.Printf("Wallhaven: %v\n", output.Providers.Wallhaven)
		cmd.Printf("Reddit: %v\n", output.Providers.Reddit)
		cmd.Printf("Nasa: %v\n", output.Providers.Nasa)
		if len(output.Providers.Generic) > 0 {
			cmd.Println("Manual Providers:")
			keys := make([]string, 0, len(output.Providers.Generic))
			for k := range output.Providers.Generic {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, name := range keys {
				cmd.Printf("  %s: %v\n", name, output.Providers.Generic[name])
			}
		}
		cmd.Println()
	}

	if output.Storage != nil {
		cmd.Println("--- Storage ---")
		cmd.Printf("Cache: %s\n", output.Storage.CacheSize)
		cmd.Printf("Data: %s\n", output.Storage.DataSize)
		cmd.Printf("Total: %s\n", output.Storage.TotalSize)
		cmd.Println()
	}
}

func getSystemStatus() *SystemStatus {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".gower")

	deps := make(map[string]bool)
	deps["feh"] = checkCommand("feh")
	deps["nitrogen"] = checkCommand("nitrogen")
	deps["gsettings"] = checkCommand("gsettings")
	deps["dbus-send"] = checkCommand("dbus-send")
	deps["swaybg"] = checkCommand("swaybg")
	deps["quickshell"] = checkCommand("quickshell") // For Dank Material Shell (dms)
	deps["niri"] = checkCommand("niri")
	deps["matugen"] = checkCommand("matugen")

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
	data, err := ioutil.ReadFile(pidFile)
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
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".gower")

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
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
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
