package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var blacklistCmd = &cobra.Command{
	Use:   "blacklist",
	Short: "Add a wallpaper to the blacklist",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ejecutando el comando 'blacklist'...")
	},
}

func init() {
	rootCmd.AddCommand(blacklistCmd)
}
