// cmd/feed.go
package cmd

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"text/tabwriter"
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
	feedSort          string
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"

	symbolCheck = "✔"
	symbolCross = "✘"
)

func colorize(text, color string) string {
	if config.NoColor {
		return text
	}
	return color + text + colorReset
}

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Manage wallpaper feed/history",
	Long:  `View, search and manage your wallpaper history feed`,
	Run: func(cmd *cobra.Command, args []string) {
		if !config.JSONOutput {
			cmd.Println("Ejecutando el comando 'feed'...")
		}
		_ = cmd.Help()
	},
}

var feedShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show feed history",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		if feedRefresh && !config.JSONOutput {
			cmd.Println("Refreshing feed view...")
		}

		// Mostrar feed normal
		limit := feedLimit
		if limit <= 0 {
			limit = 1000000 // A very large number to mean "all" for the controller
		}
		wallpapers, err := controller.GetFeed(feedPage, limit, "", feedTheme, feedColor, feedSort, feedRefresh)
		if err != nil {
			cmd.Printf("Error getting feed: %v\n", err)
			return
		}

		if len(wallpapers) == 0 {
			cmd.Println("No wallpapers found in feed.")
			return
		}

		// Manejar salida JSON/CSV/Table (se asumen flags globales)
		if config.JSONOutput {
			displayJSON(cmd, wallpapers)
		} else {
			displayTable(cmd, wallpapers)
			
			// Add summary
			stats, _ := controller.GetFeedStats()
			actualLimit := feedLimit
			if actualLimit <= 0 {
				actualLimit = stats.Total
			}
			start := (feedPage - 1) * actualLimit
			end := start + len(wallpapers)
			if len(wallpapers) > 0 {
				if stats.Total > actualLimit && feedLimit > 0 {
					cmd.Printf("\nShowing %d-%d of %d feed items. Use --page or --limit to see more.\n", start+1, end, stats.Total)
				} else {
					cmd.Printf("\nTotal feed items: %d\n", stats.Total)
				}
			}
		}
	},
}

var feedPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge feed history",
	Run: func(cmd *cobra.Command, args []string) {
		if !feedForce {
			cmd.Println("Are you sure you want to purge the feed? Use --force to confirm.")
			return
		}

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		if err := controller.PurgeFeed(); err != nil {
			cmd.Printf("Error purging feed: %v\n", err)
			return
		}
		cmd.Println(colorize(symbolCheck+" Feed purged successfully", colorGreen))
	},
}

var feedStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show feed statistics",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)
		displayStats(cmd, controller)
	},
}

var feedAnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze feed items to extract metadata and colors",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		cmd.Println("Analyzing feed items...")

		// Define progress function as a closure to capture `cmd`
		progressFunc := func(msg string) {
			if !config.Quiet {
				if strings.Contains(msg, "Error") {
					cmd.Printf("  %s %s\n", colorize(symbolCross, colorRed), msg)
				} else if strings.Contains(msg, "Downloading") {
					cmd.Printf("  %s %s\n", colorize("⬇", colorCyan), msg)
				} else if strings.Contains(msg, "Deleting") || strings.Contains(msg, "Removing") {
					cmd.Printf("  %s %s\n", colorize("🗑", colorRed), msg)
				} else if strings.Contains(msg, "Skipping") {
					cmd.Printf("  %s %s\n", colorize("⏭", colorYellow), msg)
				} else {
					cmd.Printf("%s %s\n", colorize("::", colorBlue), msg)
				}
			}
		}

		if err := controller.AnalyzeFeed(feedAll, feedForce, progressFunc); err != nil {
			cmd.Printf("Error analyzing feed: %v\n", err)
			return
		}

		// If --all is used, also analyze favorites to ensure full consistency
		if feedAll {
			cmd.Println("Analyzing favorites (implied by --all)...")
			if err := controller.AnalyzeFavorites(true, feedForce, progressFunc); err != nil {
				cmd.Printf("Error analyzing favorites: %v\n", err)
			}
		}

		cmd.Println(colorize(symbolCheck+" Analysis complete.", colorGreen))
	},
}

var feedRandomCmd = &cobra.Command{
	Use:   "random",
	Short: "Get a random wallpaper from feed or favorites",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		var wallpaper models.Wallpaper

		if feedFromFavorites {
			favorites, err := loadFavorites() // Using function from favorites.go
			if err != nil {
				cmd.Printf("Error loading favorites: %v\n", err)
				return
			}
			if len(favorites) == 0 {
				cmd.Println("No favorites found.")
				return
			}
			// Simple random pick from favorites
			fav := favorites[rand.Intn(len(favorites))] //nolint:gosec // Not security-critical, seeding is done in root.
			wallpaper = fav.Wallpaper
		} else {
			var err error
			wallpaper, err = controller.GetRandomFromFeed(feedTheme)
			if err != nil {
				cmd.Printf("Error getting random wallpaper: %v\n", err)
				return
			}
		}

		displayWallpaper(cmd, wallpaper, config.NoColor)
	},
}

var feedUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Sync feed from provider caches",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		// Define progress function as a closure to capture `cmd`
		progressFunc := func(msg string) {
			if !config.Quiet {
				if strings.Contains(msg, "Error") {
					cmd.Printf("  %s %s\n", colorize(symbolCross, colorRed), msg)
				} else if strings.Contains(msg, "Downloading") {
					cmd.Printf("  %s %s\n", colorize("⬇", colorCyan), msg)
				} else if strings.Contains(msg, "Deleting") || strings.Contains(msg, "Removing") {
					cmd.Printf("  %s %s\n", colorize("🗑", colorRed), msg)
				} else if strings.Contains(msg, "Skipping") {
					cmd.Printf("  %s %s\n", colorize("⏭", colorYellow), msg)
				} else {
					cmd.Printf("%s %s\n", colorize("::", colorBlue), msg)
				}
			}
		}

		cmd.Println("Syncing feed from parser caches...")
		count, repaired, err := controller.SyncFeed()
		if err != nil {
			cmd.Printf("Error syncing feed: %v\n", err) // This uses the local `cmd`
			return
		}

		if count == 0 && repaired == 0 {
			// Verificar si ya tenemos suficientes wallpapers (Soft Limit)
			stats, err := controller.GetFeedStats()
			if err == nil && cfg.Limits.FeedSoftLimit > 0 && stats.Total >= cfg.Limits.FeedSoftLimit {
				cmd.Printf("%s Feed saludable (%d items, límite suave: %d). Saltando búsqueda automática.\n", colorize(symbolCheck, colorGreen), stats.Total, cfg.Limits.FeedSoftLimit)
				return
			}

			// Verificar Rate Limit usando la fecha de los archivos de caché
			lastUpdate, err := controller.GetLastProviderUpdateTime()
			if err == nil && !lastUpdate.IsZero() {
				elapsed := time.Since(lastUpdate)
				limitPeriod := time.Duration(cfg.Limits.RateLimitPeriod) * time.Minute
				if elapsed < limitPeriod && !feedForce {
					cmd.Printf("%s Límite de frecuencia activo. Última búsqueda hace %v (Límite: %v). Saltando búsqueda en proveedores.\nUse --force para ignorar.\n", colorize("!", colorYellow), elapsed.Round(time.Minute), limitPeriod)
					return
				}
			}

			// If nothing added, maybe caches are empty. Run explore all.
			cmd.Println(colorize("No new wallpapers found in cache. Running 'explore --all'...", colorYellow))
			// We call the explore command logic directly or via subprocess // This uses the local `cmd`
			// For simplicity, we can just invoke the runExplore function if we exported it or use executeCommand logic
			// But since runExplore is in same package:
			exploreAll = true
			runExplore(exploreCmd, []string{"random"}) // Search for "random" or generic

			// Sync again
			cmd.Println("Syncing feed again...")
			count, repaired, _ = controller.SyncFeed()
		}

		// Después de sincronizar nuevos elementos, analiza todo el feed para asegurar que todos los elementos
		cmd.Println("Analyzing entire feed for validity and consistency...")
		if err := controller.AnalyzeFeed(true, false, progressFunc); err != nil { // 'true' para todos, 'false' para forzar regeneración de miniaturas
			cmd.Printf("Warning: Error during post-sync feed analysis: %v\n", err)
		}

		cmd.Printf("%s Feed updated. Added %d new wallpapers, repaired %d. Feed analyzed for consistency.\n", colorize(symbolCheck, colorGreen), count, repaired)
	},
}

var feedGetColorsCmd = &cobra.Command{
	Use:   "get colors",
	Short: "Get the color palette of feed wallpapers from colors.json",
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig() // Ensure config is loaded or initialized

		palettes, err := loadColorPalettes()
		if err != nil {
			cmd.Printf("Error loading color palettes: %v\n", err)
			return
		}

		if config.JSONOutput {
			data, _ := json.MarshalIndent(palettes.FeedPalette, "", "  ")
			cmd.Println(string(data))
		} else {
			if len(palettes.FeedPalette) == 0 {
				cmd.Println("No feed colors found in palette.")
				return
			}
			for _, color := range palettes.FeedPalette {
				cmd.Println(color)
			}
		}
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
	feedCmd.AddCommand(feedGetColorsCmd)

	feedShowCmd.Flags().IntVarP(&feedPage, "page", "p", 1, "Page number")
	feedShowCmd.Flags().IntVarP(&feedLimit, "limit", "l", 0, "Items per page (default: all)")
	feedShowCmd.Flags().StringVar(&feedTheme, "theme", "", "Filter by theme [dark|light]")
	feedShowCmd.Flags().StringVar(&feedColor, "color", "", "Filter by color (hex)")
	feedShowCmd.Flags().BoolVar(&feedRefresh, "refresh", false, "Refresh feed view")
	feedShowCmd.Flags().StringVar(&feedSort, "sort", "smart", "Sort order [smart|newest|oldest|source]")

	feedPurgeCmd.Flags().BoolVar(&feedForce, "force", false, "Force purge without confirmation")
	feedUpdateCmd.Flags().BoolVar(&feedForce, "force", false, "Force update ignoring limits")

	feedStatsCmd.Flags().BoolVar(&feedDetailed, "detailed", false, "Show detailed statistics")

	feedAnalyzeCmd.Flags().BoolVar(&feedAll, "all", false, "Analyze all items, not just new ones")
	feedAnalyzeCmd.Flags().BoolVar(&feedForce, "force", false, "Force regeneration of thumbnails")

	feedRandomCmd.Flags().StringVar(&feedTheme, "theme", "", "Filter by theme [dark|light]")
	feedRandomCmd.Flags().BoolVar(&feedFromFavorites, "from-favorites", false, "Pick from favorites instead of feed")
}

// Funciones helper para display
func displayStats(cmd *cobra.Command, controller *core.Controller) {
	stats, err := controller.GetFeedStats()
	if err != nil {
		cmd.Printf("Error getting stats: %v\n", err)
		return
	}

	if config.JSONOutput {
		displayJSON(cmd, stats)
	} else {
		cmd.Printf("Wallpaper Statistics:\n")
		cmd.Printf("  Feed size (history): %d\n", stats.Total)
		cmd.Printf("  Favorites:           %d\n", stats.FavoritesCount)
		cmd.Printf("  -------------------------\n")
		cmd.Printf("  Total Managed:       %d\n", stats.Total+stats.FavoritesCount)
		cmd.Printf("\nTheme distribution (Feed only):\n")
		cmd.Printf("  Dark theme:          %d\n", stats.DarkCount)
		cmd.Printf("  Light theme:         %d\n", stats.LightCount)
		
		if feedDetailed {
			cmd.Println("\nDetailed statistics:")
			cmd.Println("  (Detailed breakdown by source not implemented yet)")
		}
	}
}

func displayWallpaper(cmd *cobra.Command, wallpaper interface{}, noColor bool) {
	// Placeholder: Implementar lógica de visualización del wallpaper
	cmd.Printf("Displaying wallpaper: %+v (Color disabled: %t)\n", wallpaper, noColor)
}

func displayJSON(cmd *cobra.Command, data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		cmd.PrintErrf("Error marshalling JSON: %v\n", err)
		return
	}
	cmd.Println(string(jsonData))
}

func displayTable(cmd *cobra.Command, wallpapers interface{}) {
	wps, ok := wallpapers.([]models.Wallpaper)
	if !ok {
		return
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tRES\tTHEME\tSOURCE\tSEEN") // Error check is not critical for CLI output

	for _, wp := range wps {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%t\n", wp.ID, wp.Dimension, wp.Theme, wp.Source, wp.Seen)
	}
	_ = w.Flush()
}
