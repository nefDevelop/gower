package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// La versión llega desde main vía ldflags (-X main.Version). Aquí se guarda el
// valor ya resuelto para mostrarlo y para el flag --version de cobra.
var (
	version   = "dev"
	buildTime = "unknown"
)

// SetVersionInfo establece la versión y tiempo de build desde main.
func SetVersionInfo(v, bt string) {
	version = v
	buildTime = bt
	rootCmd.Version = v
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show gower version",
	Run: func(cmd *cobra.Command, args []string) {
		if config.JSONOutput {
			out := map[string]string{"version": version, "build_time": buildTime}
			data, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				cmd.PrintErrf("Error marshalling version: %v\n", err)
				return
			}
			cmd.Println(string(data))
			return
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Gower Version: %s\n", version)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Build Time:    %s\n", buildTime)
	},
}
