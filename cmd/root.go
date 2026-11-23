package cmd

import (
	"fmt"
	"os"

	"gower/internal/utils"

	"github.com/spf13/cobra"
)

// CLIConfig holds all global command line flags
type CLIConfig struct {
	Debug      bool
	Quiet      bool
	JSONOutput bool
	NoColor    bool
	ConfigFile string
	DryRun     bool
}

var config CLIConfig

// commandErr permite que los comandos basados en Run (que no devuelven error)
// propaguen un fallo hasta Execute a través de PersistentPostRunE, de modo que
// el proceso termine con código distinto de cero.
var commandErr error

func setCommandError(err error) {
	if err != nil && commandErr == nil {
		commandErr = err
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gower",
	Short: "A powerful wallpaper manager CLI.",
	Long: `gower is a command-line tool to manage and change your desktop wallpapers
from various online sources.`,
	// Cobra no debe imprimir el uso ni el error: Execute los maneja en un único
	// punto, por stderr, para no contaminar stdout (salida --json).
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		commandErr = nil
		if err := utils.InitLogger(config.Debug); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to initialize logger: %v\n", err)
		} else {
			utils.Log.Info("Execution started: %s %v", cmd.CommandPath(), args)
		}
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		_ = utils.CloseLogger()
		return commandErr
	},
	// Version se establece vía SetVersionInfo (ldflags); este es el fallback.
	Version: "0.1.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().BoolVar(&config.Debug, "debug", false, "Habilita la salida de depuración.")
	rootCmd.PersistentFlags().BoolVarP(&config.Quiet, "quiet", "q", false, "Suprime toda la salida excepto los errores.")
	rootCmd.PersistentFlags().BoolVar(&config.JSONOutput, "json", false, "Formatea la salida como JSON.")
	rootCmd.PersistentFlags().BoolVar(&config.NoColor, "no-color", false, "Desactivar colores en output.")
	rootCmd.PersistentFlags().StringVar(&config.ConfigFile, "config", "", "Ruta al archivo de configuración.")
	rootCmd.PersistentFlags().BoolVar(&config.DryRun, "dry-run", false, "Simula la ejecución sin realizar cambios.")
}
