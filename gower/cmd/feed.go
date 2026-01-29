// cmd/feed.go
package cmd

import (
	"fmt"
	"os"

	"gower/internal/core"

	"github.com/spf13/cobra"
)

var (
	feedPage    int
	feedLimit   int
	feedSearch  string
	feedTheme   string
	feedPurge   bool
	feedStats   bool
	feedRandom  bool
	feedNoColor bool
)

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Manage wallpaper feed/history",
	Long:  `View, search and manage your wallpaper history feed`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ejecutando el comando 'feed'...")
		fmt.Println("Considera usar 'gower feed show --help' o 'gower feed search --help'")
	},
}

func init() {
	// Subcomando: feed show
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show feed history",
		Run: func(cmd *cobra.Command, args []string) {
			controller := core.NewController()

			// Mostrar estadísticas si se solicita
			if feedStats {
				displayStats(controller)
				return
			}

			// Purgar si se solicita
			if feedPurge {
				if err := controller.PurgeFeed(); err != nil {
					fmt.Printf("Error purging feed: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Feed purged successfully")
				return
			}

			// Obtener aleatorio si se solicita
			if feedRandom {
				wallpaper, err := controller.GetRandomFromFeed(feedTheme)
				if err != nil {
					fmt.Printf("Error getting random wallpaper: %v\n", err)
					os.Exit(1)
				}
				displayWallpaper(wallpaper, feedNoColor)
				return
			}

			// Mostrar feed normal
			wallpapers, err := controller.GetFeed(feedPage, feedLimit, feedSearch, feedTheme)
			if err != nil {
				fmt.Printf("Error getting feed: %v\n", err)
				os.Exit(1)
			}

			// Manejar salida JSON/CSV/Table (se asumen flags globales)
			if config.JSONOutput { // Usando config.JSONOutput de root.go
				displayJSON(wallpapers)
			} else { // Por ahora solo displayTable, se pueden añadir más tarde
				displayTable(wallpapers, feedNoColor)
			}
		},
	}

	showCmd.Flags().IntVarP(&feedPage, "page", "p", 1, "Page number")
	showCmd.Flags().IntVarP(&feedLimit, "limit", "l", 20, "Items per page")
	showCmd.Flags().StringVarP(&feedSearch, "search", "s", "", "Search term")
	showCmd.Flags().StringVar(&feedTheme, "theme", "", "Filter by theme [dark|light]")
	showCmd.Flags().BoolVar(&feedPurge, "purge", false, "Purge old entries")
	showCmd.Flags().BoolVar(&feedStats, "stats", false, "Show feed statistics")
	showCmd.Flags().BoolVarP(&feedRandom, "random", "r", false, "Get random wallpaper")
	showCmd.Flags().BoolVar(&feedNoColor, "no-color", false, "Disable color output")

	// Subcomando: feed search
	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search in feed",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			controller := core.NewController()

			wallpapers, err := controller.SearchFeed(args[0], feedPage, feedLimit, feedTheme)
			if err != nil {
				fmt.Printf("Error searching feed: %v\n", err)
				os.Exit(1)
			}

			if config.JSONOutput { // Usando config.JSONOutput de root.go
				displayJSON(wallpapers)
			} else {
				displayTable(wallpapers, feedNoColor)
			}
		},
	}

	searchCmd.Flags().IntVarP(&feedPage, "page", "p", 1, "Page number")
	searchCmd.Flags().IntVarP(&feedLimit, "limit", "l", 20, "Items per page")
	searchCmd.Flags().StringVar(&feedTheme, "theme", "", "Filter by theme [dark|light]")
	searchCmd.Flags().BoolVar(&feedNoColor, "no-color", false, "Disable color output")

	// Agregar subcomandos
	feedCmd.AddCommand(showCmd)
	feedCmd.AddCommand(searchCmd)

	rootCmd.AddCommand(feedCmd)
}

// Funciones helper para display
func displayStats(controller *core.Controller) {
	stats, err := controller.GetFeedStats()
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		os.Exit(1)
	}

	if config.JSONOutput {
		// Asumiendo que Stats también puede ser serializado a JSON
		// y que displayJSON puede manejar diferentes tipos
		displayJSON(stats)
	} else {
		fmt.Printf("Feed Statistics:\n")
		fmt.Printf("  Total wallpapers: %d\n", stats.Total)
		fmt.Printf("  Dark theme: %d\n", stats.DarkCount)
		fmt.Printf("  Light theme: %d\n", stats.LightCount)
		fmt.Printf("  Favorites: %d\n", stats.FavoritesCount)
		// Necesitaríamos saber el formato exacto de LastAdded para mostrarlo
		// fmt.Printf("  Last added: %s\n", stats.LastAdded.Format("2006-01-02 15:04:05"))
	}
}

func displayWallpaper(wallpaper interface{}, noColor bool) {
	// Placeholder: Implementar lógica de visualización del wallpaper
	fmt.Printf("Displaying wallpaper: %+v (Color disabled: %t)\n", wallpaper, noColor)
}

func displayJSON(data interface{}) {
	// Placeholder: Implementar lógica de visualización JSON
	fmt.Printf("Displaying JSON: %+v\n", data)
}

func displayTable(wallpapers interface{}, noColor bool) {
	// Placeholder: Implementar lógica de visualización de tabla
	fmt.Printf("Displaying table: %+v (Color disabled: %t)\n", wallpapers, noColor)
}

