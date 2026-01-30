package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var (
	providerKey         string
	providerResultsPath string
	providerIDPath      string
	providerURLPath     string
	providerResPath     string
)

// configProviderCmd representa el comando base para gestionar proveedores
var configProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage wallpaper providers",
	Long:  `Add, remove, or edit wallpaper providers (both native like Reddit and generic ones).`,
}

// --- Comandos Generales ---

var configProviderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured providers",
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		cmd.Println("Native Providers:")
		cmd.Printf("  [Wallhaven] Enabled: %v\n", cfg.Providers.Wallhaven.Enabled)
		cmd.Printf("  [Reddit]    Enabled: %v, Subreddits: %s\n", cfg.Providers.Reddit.Enabled, cfg.Providers.Reddit.Subreddit)
		cmd.Printf("  [Nasa]      Enabled: %v\n", cfg.Providers.Nasa.Enabled)

		if len(cfg.GenericProviders) > 0 {
			cmd.Println("\nGeneric Providers:")
			for _, p := range cfg.GenericProviders {
				status := "Disabled"
				if p.Enabled {
					status = "Enabled"
				}
				cmd.Printf("  [%s] %s - URL: %s\n", p.Name, status, p.APIURL)
			}
		} else {
			cmd.Println("\nNo generic providers configured.")
		}
	},
}

// --- Comandos para Generic Providers ---

var configProviderAddCmd = &cobra.Command{
	Use:     "add <name> <url>",
	Short:   "Add a new generic provider",
	Long:    `Add a new generic provider. The URL can contain {query} and {apikey} placeholders.`,
	Example: `  gower config provider add myapi "https://api.example.com/search?q={query}" --key "12345"`,
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		name := args[0]
		url := args[1]

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		// Verificar si ya existe
		for _, p := range cfg.GenericProviders {
			if p.Name == name {
				cmd.Printf("Error: Provider '%s' already exists.\n", name)
				return
			}
		}

		newProvider := models.GenericProviderConfig{
			Name:    name,
			Enabled: true,
			APIURL:  url,
			APIKey:  providerKey,
			// ResponseMapping se guarda en archivo externo
		}

		cfg.GenericProviders = append(cfg.GenericProviders, newProvider)

		if err := saveConfig(cfg); err != nil {
			cmd.Printf("Error saving config: %v\n", err)
			return
		}

		// Guardar configuración del parser en data/parser/<name>.json
		mapping := models.ResponseMapping{
			ResultsPath:   providerResultsPath,
			IDPath:        providerIDPath,
			URLPath:       providerURLPath,
			DimensionPath: providerResPath,
		}

		homeDir, _ := os.UserHomeDir()
		parserDir := filepath.Join(homeDir, ".gower", "data", "parser")
		if err := os.MkdirAll(parserDir, 0755); err != nil {
			cmd.Printf("Warning: Could not create parser directory: %v\n", err)
		}

		parserFile := filepath.Join(parserDir, name+".json")
		data, _ := json.MarshalIndent(mapping, "", "  ")
		if err := ioutil.WriteFile(parserFile, data, 0644); err != nil {
			cmd.Printf("Warning: Could not save parser config: %v\n", err)
		}

		cmd.Printf("Provider '%s' added successfully.\n", name)
	},
}

var configProviderRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a generic provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		name := args[0]

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		found := false
		var newProviders []models.GenericProviderConfig
		for _, p := range cfg.GenericProviders {
			if p.Name == name {
				found = true
			} else {
				newProviders = append(newProviders, p)
			}
		}

		if !found {
			cmd.Printf("Provider '%s' not found.\n", name)
			return
		}

		cfg.GenericProviders = newProviders
		if err := saveConfig(cfg); err != nil {
			cmd.Printf("Error saving config: %v\n", err)
			return
		}

		// Eliminar archivo del parser
		homeDir, _ := os.UserHomeDir()
		parserFile := filepath.Join(homeDir, ".gower", "data", "parser", name+".json")
		if err := os.Remove(parserFile); err != nil && !os.IsNotExist(err) {
			cmd.Printf("Warning: Could not remove parser file: %v\n", err)
		}

		cmd.Printf("Provider '%s' removed.\n", name)
	},
}

// --- Comandos para Reddit ---

var configProviderRedditCmd = &cobra.Command{
	Use:   "reddit",
	Short: "Manage Reddit provider settings",
}

var configProviderRedditAddCmd = &cobra.Command{
	Use:   "add <subreddit>",
	Short: "Add a subreddit to the list",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		sub := args[0]

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		current := cfg.Providers.Reddit.Subreddit
		if current == "" {
			cfg.Providers.Reddit.Subreddit = sub
		} else {
			// Evitar duplicados simples
			if strings.Contains(current, sub) {
				cmd.Printf("Subreddit '%s' seems to be already in the list.\n", sub)
				return
			}
			cfg.Providers.Reddit.Subreddit = current + "+" + sub
		}

		if err := saveConfig(cfg); err != nil {
			cmd.Printf("Error saving config: %v\n", err)
			return
		}
		cmd.Printf("Added '%s' to Reddit sources. New list: %s\n", sub, cfg.Providers.Reddit.Subreddit)
	},
}

var configProviderRedditRemoveCmd = &cobra.Command{
	Use:   "remove <subreddit>",
	Short: "Remove a subreddit from the list",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		sub := args[0]

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		current := cfg.Providers.Reddit.Subreddit
		parts := strings.Split(current, "+")
		var newParts []string
		found := false
		for _, p := range parts {
			if p == sub {
				found = true
				continue
			}
			newParts = append(newParts, p)
		}

		if !found {
			cmd.Printf("Subreddit '%s' not found in list.\n", sub)
			return
		}

		cfg.Providers.Reddit.Subreddit = strings.Join(newParts, "+")
		if err := saveConfig(cfg); err != nil {
			cmd.Printf("Error saving config: %v\n", err)
			return
		}
		cmd.Printf("Removed '%s' from Reddit sources. New list: %s\n", sub, cfg.Providers.Reddit.Subreddit)
	},
}

func init() {
	configCmd.AddCommand(configProviderCmd)

	configProviderCmd.AddCommand(configProviderListCmd)
	configProviderCmd.AddCommand(configProviderAddCmd)
	configProviderCmd.AddCommand(configProviderRemoveCmd)
	configProviderCmd.AddCommand(configProviderRedditCmd)

	configProviderRedditCmd.AddCommand(configProviderRedditAddCmd)
	configProviderRedditCmd.AddCommand(configProviderRedditRemoveCmd)

	configProviderAddCmd.Flags().StringVar(&providerKey, "key", "", "API Key for the provider")
	configProviderAddCmd.Flags().StringVar(&providerResultsPath, "results-path", "data", "JSON path to results array")
	configProviderAddCmd.Flags().StringVar(&providerIDPath, "id-path", "id", "JSON path for ID")
	configProviderAddCmd.Flags().StringVar(&providerURLPath, "url-path", "url", "JSON path for Image URL")
	configProviderAddCmd.Flags().StringVar(&providerResPath, "res-path", "resolution", "JSON path for Resolution")
}
