// cmd/feed.go
package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var (
	feedPage          int
	feedLimit         int
	feedTheme         string
	feedColor         string
	feedRefresh       bool
	feedForce         bool
	feedDetailed      bool
	feedAll           bool
	feedFromFavorites bool
)

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Manage wallpaper feed/history",
	Long:  `View, search and manage your wallpaper history feed`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Ejecutando el comando 'feed'...")
		cmd.Help()
	},
}

var feedShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show feed history",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		controller := core.NewController(cfg)

		if feedRefresh {
			fmt.Println("Refreshing feed view...")
		}

		// Mostrar feed normal
		wallpapers, err := controller.GetFeed(feedPage, feedLimit, "", feedTheme, feedColor)
		if err != nil {
			fmt.Printf("Error getting feed: %v\n", err)
			os.Exit(1)
		}

		if len(wallpapers) == 0 {
			fmt.Println("No wallpapers found in feed.")
			return
		}

		// Manejar salida JSON/CSV/Table (se asumen flags globales)
		if config.JSONOutput {
			displayJSON(wallpapers)
		} else {
			displayTable(wallpapers, config.NoColor)
		}
	},
}

var feedPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge feed history",
	Run: func(cmd *cobra.Command, args []string) {
		if !feedForce {
			fmt.Println("Are you sure you want to purge the feed? Use --force to confirm.")
			return
		}

		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		controller := core.NewController(cfg)

		if err := controller.PurgeFeed(); err != nil {
			fmt.Printf("Error purging feed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Feed purged successfully")
	},
}

var feedStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show feed statistics",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		controller := core.NewController(cfg)
		displayStats(controller)
	},
}

var feedAnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze feed items",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		controller := core.NewController(cfg)

		fmt.Println("Analyzing feed items...")
		if err := controller.AnalyzeFeed(feedAll); err != nil {
			fmt.Printf("Error analyzing feed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Analysis complete.")
	},
}

var feedRandomCmd = &cobra.Command{
	Use:   "random",
	Short: "Get a random wallpaper from feed or favorites",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		controller := core.NewController(cfg)

		var wallpaper models.Wallpaper

		if feedFromFavorites {
			favorites, err := loadFavorites() // Using function from favorites.go
			if err != nil {
				fmt.Printf("Error loading favorites: %v\n", err)
				os.Exit(1)
			}
			if len(favorites) == 0 {
				fmt.Println("No favorites found.")
				os.Exit(1)
			}
			// Simple random pick from favorites
			rand.Seed(time.Now().UnixNano())
			fav := favorites[rand.Intn(len(favorites))]
			wallpaper = fav.Wallpaper
		} else {
			var err error
			wallpaper, err = controller.GetRandomFromFeed(feedTheme)
			if err != nil {
				fmt.Printf("Error getting random wallpaper: %v\n", err)
				os.Exit(1)
			}
		}

		displayWallpaper(wallpaper, config.NoColor)
	},
}

var feedUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Sync feed from provider caches",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		controller := core.NewController(cfg)

		fmt.Println("Syncing feed from parser caches...")
		count, err := controller.SyncFeed()
		if err != nil {
			fmt.Printf("Error syncing feed: %v\n", err)
			os.Exit(1)
		}

		if count == 0 {
			// If nothing added, maybe caches are empty. Run explore all.
			fmt.Println("No new wallpapers found in cache. Running 'explore --all'...")
			// We call the explore command logic directly or via subprocess
			// For simplicity, we can just invoke the runExplore function if we exported it or use executeCommand logic
			// But since runExplore is in same package:
			exploreAll = true
			exploreSave = true // Ensure it saves to parser cache
			runExplore(exploreCmd, []string{"random"}) // Search for "random" or generic
			
			// Sync again
			fmt.Println("Syncing feed again...")
			count, _ = controller.SyncFeed()
		}

		fmt.Printf("Feed updated. Added %d new wallpapers.\n", count)
	},
}

func init() {
	rootCmd.AddCommand(feedCmd)

	// Agregar subcomandos
	feedCmd.AddCommand(feedShowCmd)
	feedCmd.AddCommand(feedUpdateCmd)
	feedCmd.AddCommand(feedPurgeCmd)
	feedCmd.AddCommand(feedStatsCmd)
	feedCmd.AddCommand(feedAnalyzeCmd)
	feedCmd.AddCommand(feedRandomCmd)

	feedShowCmd.Flags().IntVarP(&feedPage, "page", "p", 1, "Page number")
	feedShowCmd.Flags().IntVarP(&feedLimit, "limit", "l", 20, "Items per page")
	feedShowCmd.Flags().StringVar(&feedTheme, "theme", "", "Filter by theme [dark|light]")
	feedShowCmd.Flags().StringVar(&feedColor, "color", "", "Filter by color (hex)")
	feedShowCmd.Flags().BoolVar(&feedRefresh, "refresh", false, "Refresh feed view")

	feedPurgeCmd.Flags().BoolVar(&feedForce, "force", false, "Force purge without confirmation")

	feedStatsCmd.Flags().BoolVar(&feedDetailed, "detailed", false, "Show detailed statistics")

	feedAnalyzeCmd.Flags().BoolVar(&feedAll, "all", false, "Analyze all items, not just new ones")

	feedRandomCmd.Flags().StringVar(&feedTheme, "theme", "", "Filter by theme [dark|light]")
	feedRandomCmd.Flags().BoolVar(&feedFromFavorites, "from-favorites", false, "Pick from favorites instead of feed")
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
		if feedDetailed {
			fmt.Println("  (Detailed stats not implemented yet)")
		}
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
