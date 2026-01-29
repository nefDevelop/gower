package cmd

import (
	"fmt"
	"gower/internal/core"
	"gower/internal/providers"

	"github.com/spf13/cobra"
)

var (
	exploreProvider string
	exploreAll      bool
	exploreMinWidth int
	exploreColor    string
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
	exploreCmd.Flags().IntVar(&exploreMinWidth, "min-width", 0, "Filtro de resolución mínima (ancho)")
	exploreCmd.Flags().StringVar(&exploreColor, "color", "", "Buscar por color (hex)")
}

func runExplore(cmd *cobra.Command, args []string) {
	ensureConfig()

	term := ""
	if len(args) > 0 {
		term = args[0]
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error cargando configuración: %v\n", err)
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
		fmt.Println("No hay proveedores habilitados o seleccionados.")
		return
	}

	fmt.Printf("Explorando: '%s'\n", term)
	if exploreMinWidth > 0 {
		fmt.Printf("Filtro Min-Width: %dpx\n", exploreMinWidth)
	}
	if exploreColor != "" {
		fmt.Printf("Filtro Color: %s\n", exploreColor)
	}

	searchOpts := providers.SearchOptions{
		MinWidth: exploreMinWidth,
		Color:    exploreColor,
	}

	for _, p := range selectedProviders {
		fmt.Printf("Consultando proveedor: %s...\n", p.GetName())
		results, err := p.Search(term, searchOpts)
		if err != nil {
			fmt.Printf("Error buscando en %s: %v\n", p.GetName(), err)
			continue
		}

		if len(results) == 0 {
			fmt.Printf("No se encontraron resultados en %s.\n", p.GetName())
			continue
		}

		for _, wallpaper := range results {
			fmt.Printf("  - ID: %s, URL: %s, Dim: %s\n", wallpaper.ID, wallpaper.URL, wallpaper.Dimension)
		}
	}
}
