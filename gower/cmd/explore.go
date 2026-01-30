package cmd

import (
	"encoding/json"
	"fmt"
	"gower/internal/core"
	"gower/internal/providers"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var (
	exploreProvider    string
	exploreAll         bool
	exploreMinWidth    int
	exploreMinHeight   int
	exploreAspectRatio string
	exploreColor       string
	explorePage        int
	exploreForceUpdate bool
	exploreSave        bool
)

var exploreCmd = &cobra.Command{
	Use:   "explore [término]",
	Short: "Buscar wallpapers",
	Long:  `Busca wallpapers en los proveedores configurados.`,
	Run:   runExplore,
}

func init() {
	rootCmd.AddCommand(exploreCmd)

	exploreCmd.Flags().StringVar(&exploreProvider, "provider", "", "Proveedor específico")
	exploreCmd.Flags().BoolVar(&exploreAll, "all", false, "Búsqueda en todos los proveedores habilitados")
	exploreCmd.Flags().IntVar(&exploreMinWidth, "min-width", 0, "Filtro de resolución mínima")
	exploreCmd.Flags().IntVar(&exploreMinHeight, "min-height", 0, "Altura mínima")
	exploreCmd.Flags().StringVar(&exploreAspectRatio, "aspect-ratio", "", "Proporción (16:9, 21:9, etc.)")
	exploreCmd.Flags().StringVar(&exploreColor, "color", "", "Buscar por color (hex)")
	exploreCmd.Flags().IntVarP(&explorePage, "page", "p", 1, "Paginación")
	exploreCmd.Flags().BoolVar(&exploreForceUpdate, "force-update", false, "Forzar actualización")
	exploreCmd.Flags().BoolVar(&exploreSave, "save", false, "Guardar resultados en feed.json")
}

func runExplore(cmd *cobra.Command, args []string) {
	ensureConfig()

	term := ""
	if len(args) > 0 {
		term = args[0]
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	controller := core.NewController(cfg)
	allProviders := controller.ProviderManager.GetProviders()

	var selectedProviders []providers.Provider

	if exploreAll {
		selectedProviders = allProviders
	} else if exploreProvider != "" {
		p, err := controller.ProviderManager.GetProvider(exploreProvider)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		selectedProviders = append(selectedProviders, p)
	} else {
		// Default provider
		if len(allProviders) > 0 {
			selectedProviders = append(selectedProviders, allProviders[0])
		}
	}

	if len(selectedProviders) == 0 {
		fmt.Println("No enabled providers found or selected.")
		return
	}

	if !config.Quiet && !config.JSONOutput {
		fmt.Printf("Exploring: '%s'\n", term)
		if exploreMinWidth > 0 {
			fmt.Printf("Filter Min-Width: %dpx\n", exploreMinWidth)
		}
		if exploreMinHeight > 0 {
			fmt.Printf("Filter Min-Height: %dpx\n", exploreMinHeight)
		}
		if exploreAspectRatio != "" {
			fmt.Printf("Filter Aspect-Ratio: %s\n", exploreAspectRatio)
		}
		if exploreColor != "" {
			fmt.Printf("Filter Color: %s\n", exploreColor)
		}
	}

	searchOpts := providers.SearchOptions{
		MinWidth:    exploreMinWidth,
		MinHeight:   exploreMinHeight,
		AspectRatio: exploreAspectRatio,
		Color:       exploreColor,
		Page:        explorePage,
		ForceUpdate: exploreForceUpdate,
	}

	var allWallpapers []models.Wallpaper

	for _, p := range selectedProviders {
		if !config.Quiet && !config.JSONOutput {
			fmt.Printf("Querying provider: %s...\n", p.GetName())
		}
		results, err := p.Search(term, searchOpts)
		if err != nil {
			if !config.Quiet {
				fmt.Printf("Error searching %s: %v\n", p.GetName(), err)
			}
			continue
		}

		// Save to parser cache
		if err := controller.SaveParserSearch(p.GetName(), term, results); err != nil {
			if !config.Quiet {
				fmt.Printf("Warning: Failed to save parser cache for %s: %v\n", p.GetName(), err)
			}
		}

		allWallpapers = append(allWallpapers, results...)
	}

	if len(allWallpapers) == 0 {
		if !config.Quiet && !config.JSONOutput {
			fmt.Println("No results found.")
		}
		return
	}

	if config.JSONOutput {
		data, _ := json.MarshalIndent(allWallpapers, "", "  ")
		fmt.Println(string(data))
	} else {
		for _, w := range allWallpapers {
			fmt.Printf("  - ID: %s | Res: %s | Source: %s | URL: %s\n", w.ID, w.Dimension, w.Source, w.URL)
		}
	}

	if exploreSave {
		count, err := controller.AddWallpapersToFeed(allWallpapers)
		if err != nil {
			fmt.Printf("Error saving to feed: %v\n", err)
		} else if !config.Quiet && !config.JSONOutput {
			fmt.Printf("Saved %d new wallpapers to feed.\n", count)
		}
	}
}
