package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start and manage the gower daemon",
	Long: `This command starts the gower daemon, which automatically changes
wallpapers based on your configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Comando 'daemon' ejecutado. La funcionalidad aún no está implementada.")
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
