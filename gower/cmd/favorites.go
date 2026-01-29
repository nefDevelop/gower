package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var favoritesCmd = &cobra.Command{
	Use:   "favorites",
	Short: "Manage favorite wallpapers",
	Long: `This command allows you to add, remove, and list your favorite wallpapers.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Comando 'favorites' ejecutado. La funcionalidad aún no está implementada.")
	},
}

func init() {
	rootCmd.AddCommand(favoritesCmd)
}
