package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context

	UserHome   string
	ConfigPath string
	DataPath   string
	CachePath  string
	IsLaptop   bool
}

// Structs for gower CLI output (simplified for now)
type Monitor struct {
	Name    string `json:"Name"`
	Primary bool   `json:"Primary"`
}

type Wallpaper struct {
	ID        string `json:"id"`
	Ext       string `json:"ext"`
	Thumbnail string `json:"thumbnail"` // frontend will construct this URL
	Permalink string `json:"permalink"`
	PostURL   string `json:"post_url"`
	URL       string `json:"url"`
	Link      string `json:"link"`
	Seen      bool   `json:"seen"`
	// Add other fields as needed
}

type ConfigPaths struct {
	Wallpapers      string `json:"wallpapers"`
	UseSystemDir    bool   `json:"use_system_dir"`
	IndexWallpapers bool   `json:"index_wallpapers"`
}

type ConfigBehavior struct {
	ChangeInterval  int    `json:"change_interval"`
	AutoDownload    bool   `json:"auto_download"`
	RespectDarkMode bool   `json:"respect_dark_mode"`
	MultiMonitor    string `json:"multi_monitor"`
	FromFavorites   bool   `json:"from_favorites"`
	DaemonEnabled   bool   `json:"daemon_enabled"`
}

type ConfigPower struct {
	PauseOnLowBattery   bool `json:"pause_on_low_battery"`
	LowBatteryThreshold int  `json:"low_battery_threshold"`
}

type Provider struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	APIKey    string `json:"api_key,omitempty"`
	HasAPIKey bool   `json:"hasApiKey"` // Custom field for frontend
	SearchURL string `json:"search_url,omitempty"`
	IsCustom  bool   `json:"isCustom,omitempty"` // Custom field for frontend
}

type GowerConfig struct {
	Paths          ConfigPaths         `json:"paths"`
	Behavior       ConfigBehavior      `json:"behavior"`
	Power          ConfigPower         `json:"power"`
	Providers      map[string]Provider `json:"providers"`
	GenericProviders map[string]Provider `json:"generic_providers"` // Renamed from generic_providers to match JSON output
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{}
	// Determine if it's a laptop (simple heuristic for now)
	if runtime.GOOS == "linux" {
		if _, err := os.Stat("/sys/class/power_supply/BAT0"); err == nil {
			app.IsLaptop = true
		}
	}
	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// execGower runs a gower command and returns its stdout.
func (a *App) execGower(args ...string) (string, error) {
	cmd := exec.Command("gower", args...)
	output, err := cmd.CombinedOutput() // CombinedOutput captures both stdout and stderr
	if err != nil {
		// Attempt to parse JSON error if available, otherwise return raw output
		var jsonErr struct {
			Error string `json:"error"`
		}
		if e := json.Unmarshal(output, &jsonErr); e == nil && jsonErr.Error != "" {
			return "", fmt.Errorf("gower error: %s", jsonErr.Error)
		}
		return "", fmt.Errorf("gower command failed: %v, output: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// Initialize performs initial setup like determining paths
func (a *App) Initialize() error {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	a.UserHome = userHome

	// Detect config paths - similar to QML's detectPaths
	// First, check XDG standard path
	xdgConfig := filepath.Join(userHome, ".config", "gower")
	legacyConfig := filepath.Join(userHome, ".gower")

	var configPathCandidates []string
	if _, err := os.Stat(filepath.Join(xdgConfig, "config.json")); err == nil {
		configPathCandidates = append(configPathCandidates, xdgConfig)
	}
	if _, err := os.Stat(filepath.Join(legacyConfig, "config.json")); err == nil {
		configPathCandidates = append(configPathCandidates, legacyConfig)
	}
	if len(configPathCandidates) == 0 {
		// If no config found, try initializing gower config
		fmt.Println("No gower config found. Attempting to initialize...")
		_, initErr := a.execGower("config", "init")
		if initErr != nil {
			return fmt.Errorf("failed to initialize gower config: %w", initErr)
		}
		// Re-check after init, it should create XDG path by default
		if _, err := os.Stat(filepath.Join(xdgConfig, "config.json")); err == nil {
			configPathCandidates = append(configPathCandidates, xdgConfig)
		}
	}

	if len(configPathCandidates) > 0 {
		a.ConfigPath = configPathCandidates[0] // Use the first found config path
	} else {
		// Fallback if init didn't create it or still not found
		a.ConfigPath = xdgConfig
	}

	// Data and cache paths derive from config path by default in gower,
	// or are XDG standard if config path is XDG.
	if strings.Contains(a.ConfigPath, ".config/gower") { // Assuming XDG standard
		a.DataPath = filepath.Join(userHome, ".local", "share", "gower")
		a.CachePath = filepath.Join(userHome, ".cache", "gower")
	} else { // Legacy .gower structure
		a.DataPath = filepath.Join(a.ConfigPath, "data")
		a.CachePath = filepath.Join(a.ConfigPath, "cache")
	}

	fmt.Printf("Initialized paths: UserHome=%s, ConfigPath=%s, DataPath=%s, CachePath=%s, IsLaptop=%t\n",
		a.UserHome, a.ConfigPath, a.DataPath, a.CachePath, a.IsLaptop)

	return nil
}

// CheckDaemonStatus checks if the gower daemon is running
func (a *App) CheckDaemonStatus() (bool, error) {
	output, err := a.execGower("status", "--daemon", "--json") // Assuming gower status --daemon --json exists
	if err != nil {
		// If gower status --daemon --json is not available, try pgrep
		if strings.Contains(err.Error(), "unknown flag: --daemon") || strings.Contains(err.Error(), "unknown command") {
			// Fallback to pgrep logic from QML Backend
			cmd := exec.Command("sh", "-c", "pgrep -f \"gower [d]aemon\" > /dev/null && echo 1 || echo 0")
			pgrepOutput, pgrepErr := cmd.CombinedOutput()
			if pgrepErr != nil {
				return false, fmt.Errorf("failed to execute pgrep: %w, output: %s", pgrepErr, string(pgrepOutput))
			}
			return strings.TrimSpace(string(pgrepOutput)) == "1", nil
		}
		return false, err
	}

	// Try to parse JSON output if --json was used and worked
	var daemonStatus struct {
		Daemon struct {
			Running bool `json:"running"`
		} `json:"daemon"`
	}
	if err := json.Unmarshal([]byte(output), &daemonStatus); err == nil {
		return daemonStatus.Daemon.Running, nil
	}

	// If JSON parsing fails (e.g., older gower version), check for "Running" string
	return strings.Contains(output, "Running"), nil
}

// ToggleDaemon starts or stops the gower daemon
func (a *App) ToggleDaemon(enable bool) error {
	var action string
	if enable {
		action = "start"
	} else {
		action = "stop"
	}
	_, err := a.execGower("daemon", action)
	if err != nil {
		return fmt.Errorf("failed to %s daemon: %w", action, err)
	}
	return nil
}

// GetMonitors retrieves monitor information
func (a *App) GetMonitors() ([]Monitor, error) {
	output, err := a.execGower("status", "--monitors", "--json")
	if err != nil {
		return nil, fmt.Errorf("failed to get monitors: %w", err)
	}

	var status struct {
		Monitors []Monitor `json:"monitors"`
	}
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return nil, fmt.Errorf("failed to parse monitor JSON: %w, output: %s", err, output)
	}
	return status.Monitors, nil
}

// LoadConfig loads the gower configuration
func (a *App) LoadConfig() (GowerConfig, error) {
	var config GowerConfig
	if a.ConfigPath == "" {
		return config, fmt.Errorf("config path is not initialized")
	}

	configFilePath := filepath.Join(a.ConfigPath, "config.json")
	content, err := os.ReadFile(configFilePath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file %s: %w", configFilePath, err)
	}

	// The QML example did some string manipulation to find JSON start/end.
	// Assume Go's json.Unmarshal can handle leading/trailing non-JSON output if any.
	// If it fails, we might need a more robust parser.
	if err := json.Unmarshal(content, &config); err != nil {
		return config, fmt.Errorf("failed to parse config JSON from %s: %w, content: %s", configFilePath, err, string(content))
	}

	return config, nil
}

// SetConfig sets a specific configuration key-value pair
func (a *App) SetConfig(key, value string) error {
	_, err := a.execGower("config", "set", fmt.Sprintf("%s=%s", key, value))
	if err != nil {
		return fmt.Errorf("failed to set config '%s=%s': %w", key, value, err)
	}
	return nil
}

// GetConfigPath returns the current config path
func (a *App) GetConfigPath() string {
	return a.ConfigPath
}


// LoadFeed loads the wallpaper feed
func (a *App) LoadFeed(color string, page, limit int, refresh bool) ([]Wallpaper, error) {
	args := []string{"feed", "show", "--quiet", "--page", fmt.Sprintf("%d", page), "--limit", fmt.Sprintf("%d", limit), "--json"}
	if color != "" {
		args = append(args, "--color", strings.TrimPrefix(color, "#"))
	}
	if refresh {
		args = append(args, "--refresh")
	}

	output, err := a.execGower(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load feed: %w", err)
	}

	var wallpapers []Wallpaper
	if err := json.Unmarshal([]byte(output), &wallpapers); err != nil {
		return nil, fmt.Errorf("failed to parse feed JSON: %w, output: %s", err, output)
	}

	// Construct thumbnail paths for frontend
	for i := range wallpapers {
		wallpapers[i].Thumbnail = fmt.Sprintf("file://%s/thumbs/%s%s", a.CachePath, wallpapers[i].ID, wallpapers[i].Ext)
	}

	return wallpapers, nil
}

// LoadFavorites loads favorite wallpapers
func (a *App) LoadFavorites(color string) ([]Wallpaper, error) {
	args := []string{"favorites", "list", "--json"}
	if color != "" {
		args = append(args, "--color", strings.TrimPrefix(color, "#"))
	}

	output, err := a.execGower(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load favorites: %w", err)
	}

	var favs []Wallpaper
	// The QML example had some complex JSON parsing for favorites. Let's simplify and assume direct array.
	if err := json.Unmarshal([]byte(output), &favs); err != nil {
		return nil, fmt.Errorf("failed to parse favorites JSON: %w, output: %s", err, output)
	}

	for i := range favs {
		favs[i].Thumbnail = fmt.Sprintf("file://%s/thumbs/%s%s", a.CachePath, favs[i].ID, favs[i].Ext)
	}

	return favs, nil
}

// LoadCurrentWallpapers loads currently set wallpapers
func (a *App) LoadCurrentWallpapers() ([]Wallpaper, error) {
	output, err := a.execGower("status", "--wallpapers", "--json")
	if err != nil {
		return nil, fmt.Errorf("failed to load current wallpapers: %w", err)
	}

	var status struct {
		Wallpaper struct {
			Wallpapers []Wallpaper `json:"wallpapers"`
		} `json:"wallpaper"`
	}
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return nil, fmt.Errorf("failed to parse current wallpapers JSON: %w, output: %s", err, output)
	}

	for i := range status.Wallpaper.Wallpapers {
		status.Wallpaper.Wallpapers[i].Thumbnail = fmt.Sprintf("file://%s/thumbs/%s%s", a.CachePath, status.Wallpaper.Wallpapers[i].ID, status.Wallpaper.Wallpapers[i].Ext)
	}

	return status.Wallpaper.Wallpapers, nil
}

// SetWallpaper sets a wallpaper
func (a *App) SetWallpaper(id, monitor string) error {
	args := []string{"set", id}
	if monitor != "" {
		args = append(args, "--target-monitor", monitor)
	}
	_, err := a.execGower(args...)
	if err != nil {
		return fmt.Errorf("failed to set wallpaper %s: %w", id, err)
	}
	return nil
}

// Blacklist adds a wallpaper to the blacklist
func (a *App) Blacklist(id string) error {
	_, err := a.execGower("blacklist", "add", id)
	if err != nil {
		return fmt.Errorf("failed to blacklist wallpaper %s: %w", id, err)
	}
	return nil
}

// Download downloads a wallpaper
func (a *App) Download(id string) error {
	_, err := a.execGower("download", id)
	if err != nil {
		return fmt.Errorf("failed to download wallpaper %s: %w", id, err)
	}
	return nil
}

// AddFavorite adds a wallpaper to favorites
func (a *App) AddFavorite(id string) error {
	_, err := a.execGower("favorites", "add", id)
	if err != nil {
		return fmt.Errorf("failed to add wallpaper %s to favorites: %w", id, err)
	}
	return nil
}

// RemoveFavorite removes a wallpaper from favorites
func (a *App) RemoveFavorite(id string) error {
	_, err := a.execGower("favorites", "remove", id)
	if err != nil {
		return fmt.Errorf("failed to remove wallpaper %s from favorites: %w", id, err)
	}
	return nil
}

// UpdateFeed forces a feed update
func (a *App) UpdateFeed() error {
	_, err := a.execGower("feed", "update")
	if err != nil {
		return fmt.Errorf("failed to update feed: %w", err)
	}
	return nil
}

// UndoWallpaper undoes the last wallpaper change
func (a *App) UndoWallpaper() error {
	_, err := a.execGower("set", "undo")
	if err != nil {
		return fmt.Errorf("failed to undo wallpaper: %w", err)
	}
	return nil
}

// DeleteWallpaper deletes a specific wallpaper
func (a *App) DeleteWallpaper(id string) error {
	_, err := a.execGower("wallpaper", id, "--delete", "--file", "--force")
	if err != nil {
		return fmt.Errorf("failed to delete wallpaper %s: %w", id, err)
	}
	return nil
}

// Search searches for wallpapers
func (a *App) Search(query, provider string) ([]Wallpaper, error) {
	args := []string{"explore", query, "--json"}
	if provider != "" {
		args = append(args, "--provider", provider)
	}

	output, err := a.execGower(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search for wallpapers: %w", err)
	}

	var wallpapers []Wallpaper
	if err := json.Unmarshal([]byte(output), &wallpapers); err != nil {
		return nil, fmt.Errorf("failed to parse search JSON: %w, output: %s", err, output)
	}
	return wallpapers, nil
}


// OpenFolder opens a folder in the native file explorer
func (a *App) OpenFolder(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		return fmt.Errorf("unsupported platform to open folder: %s", runtime.GOOS)
	}
	return cmd.Run()
}

// OpenImageExternally opens an image file with the default application
func (a *App) OpenImageExternally(id, ext string) error {
	if a.ConfigPath == "" {
		return fmt.Errorf("config path is not initialized")
	}
	// Need to load config first to get the wallpaper path
	config, err := a.LoadConfig() 
	if err != nil {
		return fmt.Errorf("failed to load config for image path: %w", err)
	}
	imagePath := filepath.Join(config.Paths.Wallpapers, id+ext)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("start", imagePath)
	case "darwin":
		cmd = exec.Command("open", imagePath)
	case "linux":
		cmd = exec.Command("xdg-open", imagePath)
	default:
		return fmt.Errorf("unsupported platform to open image externally: %s", runtime.GOOS)
	}
	return cmd.Run()
}

// OpenURL opens a URL in the default web browser
func (a *App) OpenURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform to open URL: %s", runtime.GOOS)
	}
	return cmd.Run()
}

// OpenFolderPicker opens a native folder selection dialog and returns the selected path
func (a *App) OpenFolderPicker() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("context is not initialized")
	}
	// Wails runtime provides cross-platform dialogs
	selectedPath, err := wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select Wallpaper Folder",
	})
	if err != nil {
		return "", fmt.Errorf("failed to open folder picker: %w", err)
	}
	return selectedPath, nil
}


// SetNextWallpaper sets the next wallpaper
func (a *App) SetNextWallpaper() error {
	_, err := a.execGower("set", "next")
	if err != nil {
		return fmt.Errorf("failed to set next wallpaper: %w", err)
	}
	return nil
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}


