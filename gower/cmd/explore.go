package cmd

import (
	"fmt"
	"time"

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

	var providers []string

	if exploreAll {
		if cfg.Providers.Wallhaven.Enabled {
			providers = append(providers, "wallhaven")
		}
		if cfg.Providers.Reddit.Enabled {
			providers = append(providers, "reddit")
		}
		if cfg.Providers.Nasa.Enabled {
			providers = append(providers, "nasa")
		}
	} else if exploreProvider != "" {
		providers = []string{exploreProvider}
	} else {
		// Default: Wallhaven if enabled, else Reddit
		if cfg.Providers.Wallhaven.Enabled {
			providers = append(providers, "wallhaven")
		} else if cfg.Providers.Reddit.Enabled {
			providers = append(providers, "reddit")
		}
	}

	if len(providers) == 0 {
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

	// Rate limit logic
	rateLimitSeconds := cfg.Limits.RateLimitPeriod
	if rateLimitSeconds <= 0 {
		rateLimitSeconds = 60 // Default fallback
	}
	rateLimit := time.Duration(rateLimitSeconds) * time.Second

	for i, p := range providers {
		// Si es --all y no es el primero, esperamos el rate limit
		if exploreAll && i > 0 {
			fmt.Printf("Esperando %v (Rate Limit) antes de consultar %s...\n", rateLimit, p)
			time.Sleep(rateLimit)
		}

		fmt.Printf("Consultando proveedor: %s...\n", p)
		// TODO: Llamar al controlador para realizar la búsqueda real
	}
}
