package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Estas variables son inyectadas durante la compilación usando LDFLAGS
var (
	Version   = "dev"
	BuildTime = "unknown"
)

// SetVersionInfo establece la versión y tiempo de build desde main
func SetVersionInfo(v, bt string) {
	Version = v
	BuildTime = bt
	rootCmd.Version = v
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Muestra la versión de gower",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Gower Version: %s\n", Version)
		fmt.Printf("Build Time:    %s\n", BuildTime)
	},
}
