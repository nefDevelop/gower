// cmd/feed.go
package cmd

import (
	"fmt"
	// "os"

	// "gower/internal/core"

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
	},
}

func init() {
	/*
		// Variables locales si se prefiere, o usar las globales definidas arriba
		page    int
		limit   int
		search  string
		theme   string
		purge   bool
		stats   bool
		random  bool
		noColor bool
	*/

	// Subcomando: feed show
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show feed history",
		Run: func(cmd *cobra.Command, args []string) {
			/* controller := core.NewController()

			// Mostrar estadísticas si se solicita
			if stats {
				displayStats(controller)
				return
			}

			// Purgar si se solicita
			if purge {
				if err := controller.PurgeFeed(); err != nil {
					fmt.Printf("Error purging feed: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Feed purged successfully")
				return
			}

			// Obtener aleatorio si se solicita
			if random {
				wallpaper, err := controller.GetRandomFromFeed(theme)
				if err != nil {
					fmt.Printf("Error getting random wallpaper: %v\n", err)
					os.Exit(1)
				}
				displayWallpaper(wallpaper, noColor)
				return
			}

			// Mostrar feed normal
			wallpapers, err := controller.GetFeed(page, limit, search, theme)
			if err != nil {
				fmt.Printf("Error getting feed: %v\n", err)
				os.Exit(1)
			}

			// Mostrar según formato
			if jsonOutput {
				displayJSON(wallpapers)
			} else if csvOutput {
				displayCSV(wallpapers)
			} else {
				displayTable(wallpapers, noColor)
			}
			*/
			fmt.Println("Ejecutando 'feed show' (Lógica pendiente de internal/core)...")
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
			/* controller := core.NewController()

			wallpapers, err := controller.SearchFeed(args[0], page, limit, theme)
			if err != nil {
				fmt.Printf("Error searching feed: %v\n", err)
				os.Exit(1)
			}

			if jsonOutput {
				displayJSON(wallpapers)
			} else {
				displayTable(wallpapers, noColor)
			}
			*/
			fmt.Printf("Buscando en feed: %s\n", args[0])
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
/* func displayStats(controller *core.Controller) {
	stats, err := controller.GetFeedStats()
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		displayJSON(stats)
	} else {
		fmt.Printf("Feed Statistics:\n")
		fmt.Printf("  Total wallpapers: %d\n", stats.Total)
		fmt.Printf("  Dark theme: %d\n", stats.DarkCount)
		fmt.Printf("  Light theme: %d\n", stats.LightCount)
		fmt.Printf("  Favorites: %d\n", stats.FavoritesCount)
		fmt.Printf("  Last added: %s\n", stats.LastAdded.Format("2006-01-02 15:04:05"))
	}
} */
