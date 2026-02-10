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

	"gower/internal/core"
	"gower/internal/utils"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Mostrar configuración",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureConfig(); err != nil {
			cmd.Println(err)
			return
		}
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error cargando configuración: %v\n", err)
			return
		}
		if !config.Quiet {
			data, _ := json.MarshalIndent(cfg, "", "  ")
			cmd.Println(string(data))
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <clave=valor>",
	Short: "Cambiar configuración",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureConfig(); err != nil {
			cmd.Println(err)
			return
		}
		parts := strings.SplitN(args[0], "=", 2)
		if len(parts) != 2 {
			cmd.Println("Formato requerido: clave=valor")
			return
		}
		key, val := parts[0], parts[1]

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error cargando configuración: %v\n", err)
			return
		}

		if err := setConfigValue(cfg, key, val); err != nil {
			cmd.Printf("Error estableciendo valor: %v\n", err)
			return
		}

		if err := saveConfig(cfg); err != nil {
			cmd.Printf("Error guardando configuración: %v\n", err)
			return
		}
		if !config.Quiet {
			cmd.Printf("%s Configuración actualizada: %s = %s\n", colorize(symbolCheck, colorGreen), key, val)
		}
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <clave>",
	Short: "Obtener valor",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureConfig(); err != nil {
			cmd.Println(err)
			return
		}
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error cargando configuración: %v\n", err)
			return
		}
		val, err := getConfigValue(cfg, args[0])
		if err != nil {
			cmd.Printf("Error obteniendo valor: %v\n", err)
			return
		}
		if !config.Quiet {
			cmd.Println(val)
		}
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Restablecer configuración",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureConfig(); err != nil {
			cmd.Println(err)
			return
		}
		defaultCfg := getDefaultConfig()
		if err := saveConfig(&defaultCfg); err != nil {
			cmd.Printf("Error restableciendo configuración: %v\n", err)
			return
		}
		if !config.Quiet {
			cmd.Println(colorize(symbolCheck+" Configuración restablecida a los valores por defecto.", colorGreen))
		}
	},
}

var configUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Actualizar estructura del archivo de configuración",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureConfig(); err != nil {
			cmd.Println(err)
			return
		}
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error cargando configuración: %v\n", err)
			return
		}
		if err := saveConfig(cfg); err != nil {
			cmd.Printf("Error guardando configuración: %v\n", err)
			return
		}
		if !config.Quiet {
			cmd.Println(colorize(symbolCheck+" Configuración actualizada con nuevos campos.", colorGreen))
		}
	},
}

func ensureConfig() error {
	configFile, err := getConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("Configuración no encontrada. Ejecuta 'gower config init' primero.")
	}
	return nil
}

func getConfigPath() (string, error) {
	if config.ConfigFile != "" {
		return config.ConfigFile, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// 1. Check primary location: ~/.config/gower/config.json
	xdgPath := filepath.Join(homeDir, ".config", "gower", "config.json")
	if _, err := os.Stat(xdgPath); err == nil {
		return xdgPath, nil
	}

	// 2. Check legacy location: ~/.gower/config.json
	legacyPath := filepath.Join(homeDir, ".gower", "config.json")
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath, nil
	}

	// Default to primary for new configurations
	return xdgPath, nil
}

var loadConfig = func() (*models.Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	var cfg models.Config
	manager := utils.NewSecureJSONManager()
	if err := manager.ReadJSON(path, &cfg); err != nil {
		return nil, err
	}

	if strings.HasPrefix(cfg.Paths.Wallpapers, "~/") || cfg.Paths.Wallpapers == "~" {
		homeDir, _ := os.UserHomeDir()
		if cfg.Paths.Wallpapers == "~" {
			cfg.Paths.Wallpapers = homeDir
		} else {
			cfg.Paths.Wallpapers = filepath.Join(homeDir, cfg.Paths.Wallpapers[2:])
		}
	}

	return &cfg, nil
}

func saveConfig(cfg *models.Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}
	manager := utils.NewSecureJSONManager()
	return manager.WriteJSON(path, cfg)
}

func getDefaultConfig() models.Config {
	return models.Config{
		Providers: models.ProvidersConfig{
			Wallhaven: models.WallhavenConfig{
				Enabled:   true,
				APIKey:    "",
				RateLimit: models.RateLimitConfig{Requests: 45, PerSeconds: 60},
			},
			Reddit: models.RedditConfig{
				Enabled: true, Subreddit: "wallpapers", Sort: "mix", Limit: 100,
			},
			Nasa: models.NasaConfig{
				Enabled: false, APIKey: "DEMO_KEY",
			},
			Bing: models.BingConfig{
				Enabled: true, Market: "en-US",
			},
		},
		GenericProviders: []models.GenericProviderConfig{},
		Search: models.SearchConfig{
			MinWidth: 1920, MinHeight: 1080, AspectRatio: "16:9", Tolerance: 0.05,
		},
		Behavior: models.BehaviorConfig{
			Theme: "", ChangeInterval: 30, MultiMonitor: "clone",
			AutoDownload: true, RespectDarkMode: true, SaveFavoritesToFolder: false, FromFavorites: false,
		},
		Power: models.PowerConfig{
			BatteryMultiplier: 4, PauseOnLowBattery: true, LowBatteryThreshold: 20,
		},
		Paths: models.PathsConfig{
			Wallpapers: "", UseSystemDir: true, IndexWallpapers: false,
		},
		UI: models.UIConfig{
			ShowColors: true, ItemsPerPage: 10, ImagePreview: true,
		},
		Limits: models.LimitsConfig{
			FeedSoftLimit: 400, FeedHardLimit: 2000, RateLimitRequests: 45, RateLimitPeriod: 60, LogRetentionDays: 7,
		},
	}
}

func setConfigValue(cfg *models.Config, path string, value string) error {
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
			tagVal := strings.Split(tag, ",")[0]
			if strings.EqualFold(tagVal, part) {
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

func getConfigValue(cfg *models.Config, path string) (string, error) {
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
			tagVal := strings.Split(tag, ",")[0]
			if strings.EqualFold(tagVal, part) {
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

func createConfigStructure(cmd *cobra.Command) error {
	baseDir, err := core.GetAppDir()
	if err != nil {
		return err
	}

	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "data"),
		filepath.Join(baseDir, "data", "parser"),
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
	emptyFiles := []string{"feed.json", "favorites.json", "blacklist.json"}
	for _, f := range emptyFiles {
		path := filepath.Join(baseDir, "data", f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			ioutil.WriteFile(path, []byte("[]"), 0644)
		}
	}

	colorsPath := filepath.Join(baseDir, "data", "colors.json")
	if _, err := os.Stat(colorsPath); os.IsNotExist(err) {
		ioutil.WriteFile(colorsPath, []byte(`{"feed_palette":[],"favorites_palette":[]}`), 0644)
	}

	configFile := filepath.Join(baseDir, "config.json")

	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		return nil // Already exists
	}

	defaultConfig := getDefaultConfig()

	if err := saveConfig(&defaultConfig); err != nil {
		return fmt.Errorf("error guardando configuración inicial: %v", err)
	}

	if !config.Quiet {
		cmd.Printf("Estructura de configuración creada en: %s\n", baseDir)
	}
	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error obteniendo directorio home: %v\n", err)
		return
	}
	configFile := filepath.Join(homeDir, ".config", "gower", "config.json")

	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		if !config.Quiet {
			cmd.Printf("El archivo de configuración ya existe en: %s\n", configFile)
		}
		return
	}

	if config.DryRun {
		if !config.Quiet {
			cmd.Printf("[DRY-RUN] Se crearía el directorio base: %s\n", filepath.Dir(configFile))
			cmd.Println("[DRY-RUN] Se crearían los directorios de datos, caché y logs.")
			cmd.Println("[DRY-RUN] Se inicializarían los archivos JSON vacíos.")
			cmd.Printf("[DRY-RUN] Se generaría el archivo de configuración en: %s\n", configFile)
		}
		return
	}

	if err := createConfigStructure(cmd); err != nil {
		cmd.Printf("Error: %v\n", err)
	}
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Configuración inicial",
	Run:   runConfigInit,
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configUpdateCmd)

	configInitCmd.Flags().String("wallhaven-api-key", "", "Wallhaven API key")
	configInitCmd.Flags().StringSlice("providers", []string{"wallhaven", "reddit"},
		"enabled providers")
	configInitCmd.Flags().String("min-resolution", "1920x1080", "minimum resolution")
	configInitCmd.Flags().Int("change-interval", 30, "auto-change interval (minutes)")
	configInitCmd.Flags().String("wallpapers-dir", "", "wallpapers directory")
	configInitCmd.Flags().String("multi-monitor", "clone", "multi-monitor mode")
}
