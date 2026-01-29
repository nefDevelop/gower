package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// cmd/config.go
var (
	configShow   bool
	configSet    []string
	configGet    string
	configReset  bool
	configPath   bool
	configInit   bool
	configImport string
	configExport string
)

// PersistentConfig representa la estructura del archivo config.yaml
type PersistentConfig struct {
	Provider     string `yaml:"provider"`
	DefaultTheme string `yaml:"default_theme"`
	MinWidth     int    `yaml:"min_width"`
	MinHeight    int    `yaml:"min_height"`
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Run:   runConfig,
}

func runConfig(cmd *cobra.Command, args []string) {
	// Esta es la función principal para el comando 'config'
	// Aquí se añadiría la lógica para mostrar, setear, obtener, etc.
	fmt.Println("Ejecutando el comando 'config'...")
	fmt.Println("Flags:")
	fmt.Printf("  Show: %v\n", configShow)
	fmt.Printf("  Set: %v\n", configSet)
	fmt.Printf("  Get: %s\n", configGet)
}

func runConfigInit(cmd *cobra.Command, args []string) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Printf("Error obteniendo directorio de configuración: %v\n", err)
		return
	}

	configPath := filepath.Join(userConfigDir, "gower")
	configFile := filepath.Join(configPath, "config.yaml")

	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		fmt.Printf("El archivo de configuración ya existe en: %s\n", configFile)
		return
	}

	defaultConfig := PersistentConfig{
		Provider:     "wallhaven",
		DefaultTheme: "dark",
		MinWidth:     1920,
		MinHeight:    1080,
	}

	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		fmt.Printf("Error generando YAML: %v\n", err)
		return
	}

	os.MkdirAll(configPath, os.ModePerm)
	if err := ioutil.WriteFile(configFile, data, 0644); err != nil {
		fmt.Printf("Error escribiendo archivo: %v\n", err)
		return
	}

	fmt.Printf("Archivo de configuración creado en: %s\n", configFile)
}

func init() {
	rootCmd.AddCommand(configCmd) // <-- Adjuntar al comando raíz

	configCmd.Flags().BoolVar(&configShow, "show", false,
		"show current configuration")
	configCmd.Flags().StringSliceVar(&configSet, "set", []string{},
		"set configuration (key=value)")
	configCmd.Flags().StringVar(&configGet, "get", "",
		"get specific configuration value")
	configCmd.Flags().BoolVar(&configReset, "reset", false,
		"reset to default configuration")
	configCmd.Flags().BoolVar(&configPath, "path", false,
		"show configuration file path")
	configCmd.Flags().BoolVar(&configInit, "init", false,
		"configuration initialization")
	configCmd.Flags().StringVar(&configImport, "import", "",
		"import configuration from file")
	configCmd.Flags().StringVar(&configExport, "export", "",
		"export configuration to file")

	// Subcomando init con flags específicos
	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration",
		Run:   runConfigInit,
	}

	initCmd.Flags().String("wallhaven-api-key", "", "Wallhaven API key")
	initCmd.Flags().StringSlice("providers", []string{"wallhaven", "reddit"},
		"enabled providers")
	initCmd.Flags().String("min-resolution", "1920x1080", "minimum resolution")
	initCmd.Flags().Int("change-interval", 30, "auto-change interval (minutes)")
	initCmd.Flags().String("wallpaper-command", "", "custom wallpaper command")
	initCmd.Flags().String("wallpapers-dir", "", "wallpapers directory")
	initCmd.Flags().String("theme", "dark", "default theme [dark|light|auto]")
	initCmd.Flags().String("multi-monitor", "clone", "multi-monitor mode")

	configCmd.AddCommand(initCmd)
}
