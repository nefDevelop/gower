package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gower/internal/core"
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
		cmd.Printf("  [Reddit]    Enabled: %v, Sort: %s, Subreddits: %s\n", cfg.Providers.Reddit.Enabled, cfg.Providers.Reddit.Sort, cfg.Providers.Reddit.Subreddit)
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

		appDir, _ := core.GetAppDir()
		parserDir := filepath.Join(appDir, "data", "parser")
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
		appDir, _ := core.GetAppDir()
		parserFile := filepath.Join(appDir, "data", "parser", name+".json")
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
	Use:   "add <subreddit> [sort]",
	Short: "Add a subreddit to the list (optionally with sort: new, hot, top, mix)",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		sub := args[0]
		sort := ""
		if len(args) > 1 {
			sort = strings.ToLower(args[1])
			validSorts := map[string]bool{"new": true, "hot": true, "top": true, "controversial": true, "mix": true}
			if !validSorts[sort] {
				cmd.Printf("Invalid sort option: %s. Valid options: new, hot, top, controversial, mix.\n", sort)
				return
			}
		}

		newItem := sub
		if sort != "" {
			newItem = sub + ":" + sort
		}

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		current := cfg.Providers.Reddit.Subreddit
		// Evitar duplicados: verificar si el newItem (sub o sub:sort) ya existe exactamente
		parts := strings.Split(current, "+")
		if current == "" { // Handle initial empty case for parts
			parts = []string{}
		}

		// Check for exact duplicate (e.g., "wallpapers:top" vs "wallpapers:top")
		for _, p := range parts {
			if strings.EqualFold(p, newItem) {
				cmd.Printf("Subreddit '%s' (with specified sort) seems to be already in the list.\n", newItem)
				return
			}
		}

		// Special case: if adding a plain subreddit (e.g., "wallpapers") and a plain version of it already exists.
		if sort == "" { // Only apply this check if the newItem itself has no sort
			for _, p := range parts {
				if strings.EqualFold(p, sub) && !strings.Contains(p, ":") { // Check if 'p' is also a plain subreddit and matches 'sub'
					cmd.Printf("Subreddit '%s' seems to be already in the list.\n", sub)
					return
				}
			}
		}

		// If no duplicate was found, add the new item.
		updatedSubreddits := append(parts, newItem)
		cfg.Providers.Reddit.Subreddit = strings.Join(updatedSubreddits, "+")

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
			pName := strings.Split(p, ":")[0]
			if strings.EqualFold(pName, sub) {
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

var configProviderRedditSortCmd = &cobra.Command{
	Use:   "sort <new|hot|top|mix>",
	Short: "Set the sort order for Reddit provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		sort := strings.ToLower(args[0])
		validSorts := map[string]bool{"new": true, "hot": true, "top": true, "controversial": true, "mix": true}

		if !validSorts[sort] {
			cmd.Printf("Invalid sort option: %s. Valid options: new, hot, top, controversial, mix.\n", sort)
			return
		}

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}

		cfg.Providers.Reddit.Sort = sort
		if err := saveConfig(cfg); err != nil {
			cmd.Printf("Error saving config: %v\n", err)
			return
		}
		cmd.Printf("Reddit sort order updated to: %s\n", sort)
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
	configProviderRedditCmd.AddCommand(configProviderRedditSortCmd)

	configProviderAddCmd.Flags().StringVar(&providerKey, "key", "", "API Key for the provider")
	configProviderAddCmd.Flags().StringVar(&providerResultsPath, "results-path", "data", "JSON path to results array")
	configProviderAddCmd.Flags().StringVar(&providerIDPath, "id-path", "id", "JSON path for ID")
	configProviderAddCmd.Flags().StringVar(&providerURLPath, "url-path", "url", "JSON path for Image URL")
	configProviderAddCmd.Flags().StringVar(&providerResPath, "res-path", "resolution", "JSON path for Resolution")
}
