package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var favoritesCmd = &cobra.Command{
	Use:   "favorites",
	Short: "Muestra tus wallpapers favoritos",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ejecutando el comando 'favorites'...")
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Inicia el demonio en segundo plano",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ejecutando el comando 'daemon'...")
	},
}

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Inicia el modo interactivo",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ejecutando el comando 'interactive'...")
	},
}

func init() {
	rootCmd.AddCommand(favoritesCmd)
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(interactiveCmd)
}
