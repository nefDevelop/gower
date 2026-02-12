package cmd

import (
	"fmt"
	"os"
	"os/exec" // Added for executing the GUI app
	"path/filepath" // Added for path manipulation
	"runtime"  // Added for checking OS

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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gower",
	Short: "A powerful wallpaper manager CLI.",
	Long: `gower is a command-line tool to manage and change your desktop wallpapers
from various online sources.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := utils.InitLogger(config.Debug); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to initialize logger: %v\n", err)
		} else {
			utils.Log.Info("Execution started: %s %v", cmd.CommandPath(), args)
		}
	},
	Version: "0.1.0",
}

// guiCmd represents the gui command
var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Launches the Gower graphical user interface (GUI).",
	Long: `This command starts the Gower GUI application, providing a visual
interface to manage your wallpapers.`,
	Run: func(cmd *cobra.Command, args []string) {
		guiAppPath := ""
		switch runtime.GOOS {
		case "windows":
			guiAppPath = "gower-gui/windows/windows.exe" // Assuming build output is 'windows.exe' relative to gower executable
		case "linux":
			guiAppPath = "gower-gui/linux/windows" // Wails default app name is 'windows' for linux target
		case "darwin":
			// For macOS, Wails builds an app bundle. We'll need to specify the executable inside.
			// This might need adjustment based on final Wails build output for macOS.
			guiAppPath = "gower-gui/darwin/windows.app/Contents/MacOS/windows" // Wails default app name is 'windows' for darwin target
		default:
			fmt.Println("Unsupported operating system for GUI:", runtime.GOOS)
			os.Exit(1)
		}

		// Resolve the absolute path to the GUI application
		exePath, err := os.Executable()
		if err != nil {
			fmt.Printf("Error getting current executable path: %v\n", err)
			os.Exit(1)
		}
		
		// The Wails app executable is expected to be in a sibling directory to the gower executable.
		// e.g., if gower is in /usr/local/bin/gower, Wails app is in /usr/local/bin/gower-gui/windows/windows.exe
		// So we go one level up from gower's directory, then into gower-gui/<os>/
		baseDir := filepath.Dir(exePath)
		absGuiAppPath := filepath.Join(baseDir, guiAppPath)


		// Check if the GUI executable exists
		if _, err := os.Stat(absGuiAppPath); os.IsNotExist(err) {
			fmt.Printf("Error: Gower GUI application not found at %s. Please ensure it is built and placed correctly.\n", absGuiAppPath)
			os.Exit(1)
		}

		// Execute the GUI application
		guiCmd := exec.Command(absGuiAppPath)
		guiCmd.Stdout = os.Stdout
		guiCmd.Stderr = os.Stderr
		guiCmd.Stdin = os.Stdin

		err = guiCmd.Start() // Use Start() for non-blocking execution
		if err != nil {
			fmt.Printf("Error launching Gower GUI: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Gower GUI launched successfully.")
	},
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
	rootCmd.PersistentFlags().BoolVar(&config.Debug, "debug", false, "Habilita la salida de depuración.")
	rootCmd.PersistentFlags().BoolVarP(&config.Quiet, "quiet", "q", false, "Suprime toda la salida excepto los errores.")
	rootCmd.PersistentFlags().BoolVar(&config.JSONOutput, "json", false, "Formatea la salida como JSON.")
	rootCmd.PersistentFlags().BoolVar(&config.NoColor, "no-color", false, "Desactivar colores en output.")
	rootCmd.PersistentFlags().StringVar(&config.ConfigFile, "config", "", "Ruta al archivo de configuración.")
	rootCmd.PersistentFlags().BoolVar(&config.DryRun, "dry-run", false, "Simula la ejecución sin realizar cambios.")

	// Add the gui command to the root command
	rootCmd.AddCommand(guiCmd)
}
