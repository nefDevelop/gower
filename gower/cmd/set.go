// cmd/set.go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	setID            string
	setURL           string
	setRandom        bool
	setTheme         string
	setFromFavorites bool
	setMultiMonitor  string
	setCommand       string
	setNoDownload    bool
)

var setCmd = &cobra.Command{
	Use:   "set [id|url|random]",
	Short: "Set wallpaper",
	Args:  cobra.MaximumNArgs(1),
	Run:   runSet,
}

func init() {
	setCmd.Flags().StringVar(&setID, "id", "",
		"wallpaper ID (e.g., wh_123456)")
	setCmd.Flags().StringVar(&setURL, "url", "",
		"direct wallpaper URL")
	setCmd.Flags().BoolVarP(&setRandom, "random", "r", false,
		"set random wallpaper")
	setCmd.Flags().StringVar(&setTheme, "theme", "",
		"theme filter [dark|light|auto]")
	setCmd.Flags().BoolVar(&setFromFavorites, "from-favorites", false,
		"random from favorites only")
	setCmd.Flags().StringVar(&setMultiMonitor, "multi-monitor", "",
		"multi-monitor mode [clone|distinct]")
	setCmd.Flags().StringVar(&setCommand, "command", "",
		"custom wallpaper command")
	setCmd.Flags().BoolVar(&setNoDownload, "no-download", false,
		"don't download, use existing file")

	// Subcomandos
	setCmd.AddCommand(&cobra.Command{
		Use:   "random",
		Short: "Set random wallpaper",
		Run:   runSetRandom,
	})

	rootCmd.AddCommand(setCmd)
}

func runSet(cmd *cobra.Command, args []string) {
	fmt.Println("Ejecutando el comando 'set'...")
	// Aquí iría la lógica real para establecer el wallpaper
}

func runSetRandom(cmd *cobra.Command, args []string) {
	fmt.Println("Ejecutando el comando 'set random'...")
}
