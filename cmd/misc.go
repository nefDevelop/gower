package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var miscCmd = &cobra.Command{
	Use:   "misc",
	Short: "Miscellaneous utilities",
	Long:  `This command provides various utility functions for gower.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Comando 'misc' ejecutado. La funcionalidad aún no está implementada.")
	},
}

func init() {
	rootCmd.AddCommand(miscCmd)
}