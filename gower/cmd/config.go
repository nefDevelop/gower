package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gower/internal/utils"

	"github.com/spf13/cobra"
)

// Config structures for config.json
type Config struct {
	Providers ProvidersConfig `json:"providers"`
	Search    SearchConfig    `json:"search"`
	Behavior  BehaviorConfig  `json:"behavior"`
	Power     PowerConfig     `json:"power"`
	Paths     PathsConfig     `json:"paths"`
	UI        UIConfig        `json:"ui"`
	Limits    LimitsConfig    `json:"limits"`
}

type ProvidersConfig struct {
	Wallhaven WallhavenConfig `json:"wallhaven"`
	Reddit    RedditConfig    `json:"reddit"`
	Nasa      NasaConfig      `json:"nasa"`
}

type WallhavenConfig struct {
	Enabled   bool            `json:"enabled"`
	APIKey    string          `json:"api_key"`
	RateLimit RateLimitConfig `json:"ratelimit"`
}

type RateLimitConfig struct {
	Requests   int `json:"requests"`
	PerSeconds int `json:"per_seconds"`
}

type RedditConfig struct {
	Enabled   bool   `json:"enabled"`
	Subreddit string `json:"subreddit"`
	Sort      string `json:"sort"`
	Limit     int    `json:"limit"`
}

type NasaConfig struct {
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"api_key"`
}

type SearchConfig struct {
	MinWidth    int     `json:"min_width"`
	MinHeight   int     `json:"min_height"`
	AspectRatio string  `json:"aspect_ratio"`
	Tolerance   float64 `json:"tolerance"`
}

type BehaviorConfig struct {
	Theme            string `json:"theme"`
	ChangeInterval   int    `json:"change_interval"`
	MultiMonitor     string `json:"multi_monitor"`
	WallpaperCommand string `json:"wallpaper_command"`
	AutoDownload     bool   `json:"auto_download"`
	RespectDarkMode  bool   `json:"respect_dark_mode"`
}

type PowerConfig struct {
	BatteryMultiplier   int  `json:"battery_multiplier"`
	PauseOnLowBattery   bool `json:"pause_on_low_battery"`
	LowBatteryThreshold int  `json:"low_battery_threshold"`
}

type PathsConfig struct {
	Wallpapers   string `json:"wallpapers"`
	UseSystemDir bool   `json:"use_system_dir"`
}

type UIConfig struct {
	ShowColors   bool `json:"show_colors"`
	ItemsPerPage int  `json:"items_per_page"`
	ImagePreview bool `json:"image_preview"`
}

type LimitsConfig struct {
	FeedSoftLimit     int `json:"feed_soft_limit"`
	FeedHardLimit     int `json:"feed_hard_limit"`
	RateLimitRequests int `json:"rate_limit_requests"`
	RateLimitPeriod   int `json:"rate_limit_period"`
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Mostrar configuración",
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error cargando configuración: %v\n", err)
			return
		}
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(data))
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <clave=valor>",
	Short: "Cambiar configuración",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		parts := strings.SplitN(args[0], "=", 2)
		if len(parts) != 2 {
			fmt.Println("Formato requerido: clave=valor")
			return
		}
		key, val := parts[0], parts[1]

		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error cargando configuración: %v\n", err)
			return
		}

		if err := setConfigValue(cfg, key, val); err != nil {
			fmt.Printf("Error estableciendo valor: %v\n", err)
			return
		}

		if err := saveConfig(cfg); err != nil {
			fmt.Printf("Error guardando configuración: %v\n", err)
			return
		}
		fmt.Printf("Configuración actualizada: %s = %s\n", key, val)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <clave>",
	Short: "Obtener valor",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error cargando configuración: %v\n", err)
			return
		}
		val, err := getConfigValue(cfg, args[0])
		if err != nil {
			fmt.Printf("Error obteniendo valor: %v\n", err)
			return
		}
		fmt.Println(val)
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Restablecer configuración",
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		defaultCfg := getDefaultConfig()
		if err := saveConfig(&defaultCfg); err != nil {
			fmt.Printf("Error restableciendo configuración: %v\n", err)
			return
		}
		fmt.Println("Configuración restablecida a los valores por defecto.")
	},
}

var configExportCmd = &cobra.Command{
	Use:   "export [archivo]",
	Short: "Exportar configuración",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		path, _ := getConfigPath()
		data, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("Error leyendo configuración: %v\n", err)
			return
		}

		if len(args) > 0 {
			if err := ioutil.WriteFile(args[0], data, 0644); err != nil {
				fmt.Printf("Error exportando: %v\n", err)
				return
			}
			fmt.Printf("Configuración exportada a: %s\n", args[0])
		} else {
			fmt.Println(string(data))
		}
	},
}

var configImportCmd = &cobra.Command{
	Use:   "import <archivo>",
	Short: "Importar configuración",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		data, err := ioutil.ReadFile(args[0])
		if err != nil {
			fmt.Printf("Error leyendo archivo: %v\n", err)
			return
		}

		var tmp Config
		if err := json.Unmarshal(data, &tmp); err != nil {
			fmt.Printf("Archivo de configuración inválido: %v\n", err)
			return
		}

		path, _ := getConfigPath()
		if err := ioutil.WriteFile(path, data, 0644); err != nil {
			fmt.Printf("Error guardando configuración: %v\n", err)
			return
		}
		fmt.Println("Configuración importada exitosamente.")
	},
}

func ensureConfig() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error obteniendo directorio home: %v\n", err)
		return
	}
	configFile := filepath.Join(homeDir, ".gower", "config.json")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("Configuración no encontrada. Inicializando estructura...")
		if err := createConfigStructure(); err != nil {
			fmt.Printf("Error inicializando configuración: %v\n", err)
		}
	}
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gower", "config.json"), nil
}

func loadConfig() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	var cfg Config
	manager := utils.NewSecureJSONManager()
	if err := manager.ReadJSON(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}
	manager := utils.NewSecureJSONManager()
	return manager.WriteJSON(path, cfg)
}

func getDefaultConfig() Config {
	return Config{
		Providers: ProvidersConfig{
			Wallhaven: WallhavenConfig{
				Enabled:   true,
				APIKey:    "",
				RateLimit: RateLimitConfig{Requests: 45, PerSeconds: 60},
			},
			Reddit: RedditConfig{
				Enabled: true, Subreddit: "wallpapers", Sort: "top", Limit: 100,
			},
			Nasa: NasaConfig{
				Enabled: false, APIKey: "DEMO_KEY",
			},
		},
		Search: SearchConfig{
			MinWidth: 1920, MinHeight: 1080, AspectRatio: "16:9", Tolerance: 0.05,
		},
		Behavior: BehaviorConfig{
			Theme: "dark", ChangeInterval: 30, MultiMonitor: "clone",
			WallpaperCommand: "", AutoDownload: true, RespectDarkMode: true,
		},
		Power: PowerConfig{
			BatteryMultiplier: 4, PauseOnLowBattery: true, LowBatteryThreshold: 20,
		},
		Paths: PathsConfig{
			Wallpapers: "", UseSystemDir: true,
		},
		UI: UIConfig{
			ShowColors: true, ItemsPerPage: 10, ImagePreview: true,
		},
		Limits: LimitsConfig{
			FeedSoftLimit: 400, FeedHardLimit: 2000, RateLimitRequests: 45, RateLimitPeriod: 60,
		},
	}
}

func setConfigValue(cfg *Config, path string, value string) error {
	v := reflect.ValueOf(cfg).Elem()
	parts := strings.Split(path, ".")

	for _, part := range parts {
		if v.Kind() != reflect.Struct {
			return fmt.Errorf("ruta inválida: %s", path)
		}

		found := false
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := typ.Field(i)
			tag := field.Tag.Get("json")
			if tag == part || strings.Split(tag, ",")[0] == part {
				v = v.Field(i)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("campo no encontrado: %s", part)
		}
	}

	if !v.CanSet() {
		return fmt.Errorf("no se puede establecer el valor para %s", path)
	}

	switch v.Kind() {
	case reflect.String:
		v.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("valor entero inválido: %s", value)
		}
		v.SetInt(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("valor booleano inválido: %s", value)
		}
		v.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("valor flotante inválido: %s", value)
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("tipo no soportado: %s", v.Kind())
	}
	return nil
}

func getConfigValue(cfg *Config, path string) (string, error) {
	v := reflect.ValueOf(cfg).Elem()
	parts := strings.Split(path, ".")

	for _, part := range parts {
		if v.Kind() != reflect.Struct {
			return "", fmt.Errorf("ruta inválida: %s", path)
		}

		found := false
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := typ.Field(i)
			tag := field.Tag.Get("json")
			if tag == part || strings.Split(tag, ",")[0] == part {
				v = v.Field(i)
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("campo no encontrado: %s", part)
		}
	}

	return fmt.Sprintf("%v", v.Interface()), nil
}

func createConfigStructure() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(homeDir, ".gower")
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "data"),
		filepath.Join(baseDir, "cache", "thumbs"),
		filepath.Join(baseDir, "cache", "wallpapers"),
		filepath.Join(baseDir, "logs"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("error creando directorio %s: %v", d, err)
		}
	}

	// Create empty json files in data/
	emptyFiles := []string{"feed.json", "favorites.json", "blacklist.json", "colors.json"}
	for _, f := range emptyFiles {
		path := filepath.Join(baseDir, "data", f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			ioutil.WriteFile(path, []byte("[]"), 0644)
		}
	}

	configFile := filepath.Join(baseDir, "config.json")

	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		return nil // Already exists
	}

	defaultConfig := getDefaultConfig()

	if err := saveConfig(&defaultConfig); err != nil {
		return fmt.Errorf("error guardando configuración inicial: %v", err)
	}

	fmt.Printf("Estructura de configuración creada en: %s\n", baseDir)
	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error obteniendo directorio home: %v\n", err)
		return
	}
	configFile := filepath.Join(homeDir, ".gower", "config.json")

	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		fmt.Printf("El archivo de configuración ya existe en: %s\n", configFile)
		return
	}

	if err := createConfigStructure(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Configuración inicial",
	Run:   runConfigInit,
}

func init() {
	rootCmd.AddCommand(configCmd) // <-- Adjuntar al comando raíz

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)

	configInitCmd.Flags().String("wallhaven-api-key", "", "Wallhaven API key")
	configInitCmd.Flags().StringSlice("providers", []string{"wallhaven", "reddit"},
		"enabled providers")
	configInitCmd.Flags().String("min-resolution", "1920x1080", "minimum resolution")
	configInitCmd.Flags().Int("change-interval", 30, "auto-change interval (minutes)")
	configInitCmd.Flags().String("wallpaper-command", "", "custom wallpaper command")
	configInitCmd.Flags().String("wallpapers-dir", "", "wallpapers directory")
	configInitCmd.Flags().String("theme", "dark", "default theme [dark|light|auto]")
	configInitCmd.Flags().String("multi-monitor", "clone", "multi-monitor mode")
}
