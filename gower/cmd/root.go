package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// CLIConfig holds all global command line flags
type CLIConfig struct {
	Verbose    bool
	Debug      bool
	Quiet      bool
	JSONOutput bool
	ConfigFile string
	DryRun     bool
}

var config CLIConfig

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gower",
	Short: "A powerful wallpaper manager CLI.",
	Long: `gower is a command-line tool to manage and change your desktop wallpapers
from various online sources.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().BoolVarP(&config.Verbose, "verbose", "v", false, "Habilita la salida detallada.")
	rootCmd.PersistentFlags().BoolVar(&config.Debug, "debug", false, "Habilita la salida de depuración.")
	rootCmd.PersistentFlags().BoolVarP(&config.Quiet, "quiet", "q", false, "Suprime toda la salida excepto los errores.")
	rootCmd.PersistentFlags().BoolVar(&config.JSONOutput, "json", false, "Formatea la salida como JSON.")
	rootCmd.PersistentFlags().StringVar(&config.ConfigFile, "config", "", "Ruta al archivo de configuración.")
	rootCmd.PersistentFlags().BoolVar(&config.DryRun, "dry-run", false, "Simula la ejecución sin realizar cambios.")
}
