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
)

var exploreCmd = &cobra.Command{
	Use:   "explore [término]",
	Short: "Buscar wallpapers",
	Long:  `Busca wallpapers en los proveedores configurados.`,
	RunE:  runExplore, // Changed from Run to RunE
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
}

func runExplore(cmd *cobra.Command, args []string) error {
	ensureConfig()

	term := ""
	if len(args) > 0 {
		term = args[0]
	}

	cfg, err := loadConfig()
	if err != nil {
		cmd.Printf("Error loading config: %v\n", err)
		return err // Return the error
	}

	controller := core.NewController(cfg)
	allProviders := controller.ProviderManager.GetProviders()

	var selectedProviders []providers.Provider

	if exploreAll {
		selectedProviders = allProviders
	} else if exploreProvider != "" {
		p, err := controller.ProviderManager.GetProvider(exploreProvider) // Changed to RunE to allow error return
		if err != nil {
			cmd.Printf("Error: %v\n", err)
			return err
		}
		selectedProviders = append(selectedProviders, p)
	} else {
		// Default provider // Changed to RunE to allow error return
		if len(allProviders) > 0 {
			selectedProviders = append(selectedProviders, allProviders[0])
		}
	}

	if len(selectedProviders) == 0 {
		cmd.Println("No enabled providers found or selected.")
		return nil // No error, but explicitly return nil
	}

	if !config.Quiet && !config.JSONOutput {
		displayTerm := term
		if displayTerm == "" {
			displayTerm = "random/latest"
		}
		cmd.Printf("Exploring: '%s'\n", displayTerm)
		if exploreMinWidth > 0 {
			cmd.Printf("Filter Min-Width: %dpx\n", exploreMinWidth)
		}
		if exploreMinHeight > 0 {
			cmd.Printf("Filter Min-Height: %dpx\n", exploreMinHeight)
		}
		if exploreAspectRatio != "" {
			cmd.Printf("Filter Aspect-Ratio: %s\n", exploreAspectRatio)
		}
		if exploreColor != "" {
			cmd.Printf("Filter Color: %s\n", exploreColor)
		}
	}

	// Prepare ExcludeIDs from feed and blacklist
	feed, _ := controller.GetFeedWallpapers()
	blacklist, _ := controller.GetBlacklist()
	excludeMap := make(map[string]bool)
	for _, wp := range feed {
		excludeMap[wp.ID] = true
	}
	for _, id := range blacklist {
		excludeMap[id] = true
	}

	searchOpts := providers.SearchOptions{
		MinWidth:    exploreMinWidth,
		MinHeight:   exploreMinHeight,
		AspectRatio: exploreAspectRatio,
		Color:       exploreColor,
		Page:        explorePage,
		ForceUpdate: exploreForceUpdate,
		ExcludeIDs:  excludeMap,
	}

	var allWallpapers []models.Wallpaper
	var encounteredError error // Para almacenar el primer error encontrado

	for _, p := range selectedProviders {
		if !config.Quiet && !config.JSONOutput {
			cmd.Printf("Querying provider: %s...\n", p.GetName())
		}
		results, err := p.Search(term, searchOpts)
		if err != nil {
			if !config.Quiet {
				cmd.Printf("Warning: Error searching %s: %v\n", p.GetName(), err) // Cambiado a Warning
			}
			if encounteredError == nil { // Almacenar el primer error
				encounteredError = fmt.Errorf("provider %s: %w", p.GetName(), err)
			}
			continue
		}

		// Save to parser cache
		if err := controller.SaveParserSearch(p.GetName(), term, results); err != nil {
			if !config.Quiet {
				cmd.Printf("Warning: Failed to save parser cache for %s: %v\n", p.GetName(), err)
			}
		}

		allWallpapers = append(allWallpapers, results...)
	}

	if len(allWallpapers) == 0 { // Si no se encontraron wallpapers en absoluto
		if !config.Quiet && !config.JSONOutput {
			cmd.Println("No results found.")
		}
		if encounteredError != nil { // Y hubo al menos un error durante la búsqueda
			// Devolver el primer error encontrado
			return fmt.Errorf("explore failed: %w", encounteredError)
		}
		return nil // No error, but explicitly return nil
	}

	if config.JSONOutput {
		data, _ := json.MarshalIndent(allWallpapers, "", "  ")
		cmd.Println(string(data))
	} else {
		for _, w := range allWallpapers {
			cmd.Printf("  - ID: %s | Res: %s | Source: %s | URL: %s\n", w.ID, w.Dimension, w.Source, w.URL)
		}
	}
	return nil
}
