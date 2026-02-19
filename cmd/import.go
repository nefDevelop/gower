package cmd

import (
	"encoding/json"
	"os"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import application data",
}

var importConfigCmd = &cobra.Command{
	Use:   "config <file>",
	Short: "Import configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		data, err := os.ReadFile(args[0])
		if err != nil {
			cmd.Printf("Error reading file: %v\n", err)
			return
		}

		var tmp models.Config
		if err := json.Unmarshal(data, &tmp); err != nil {
			cmd.Printf("Invalid configuration file: %v\n", err)
			return
		}

		path, _ := getConfigPath()
		if err := os.WriteFile(path, data, 0644); err != nil {
			cmd.Printf("Error saving configuration: %v\n", err)
			return
		}
		if !config.Quiet {
			cmd.Println(colorize(symbolCheck+" Configuration imported successfully.", colorGreen))
		}
	},
}

var importFavoritesCmd = &cobra.Command{
	Use:   "favorites",
	Short: "Import favorite wallpapers from a file",
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		filePath := favFile
		if filePath == "" {
			cmd.Println("Error: --file flag is required for import")
			os.Exit(1)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			cmd.Printf("Error reading import file %s: %v\n", filePath, err)
			return
		}

		var importedFavorites []core.FavoriteWallpaper
		if err := json.Unmarshal(data, &importedFavorites); err != nil {
			cmd.Printf("Error unmarshalling import file: %v\n", err)
			return
		}

		if err := saveFavorites(importedFavorites); err != nil {
			cmd.Printf("Error saving favorites: %v\n", err)
			return
		}

		if cfg, err := loadConfig(); err == nil {
			_ = core.NewController(cfg).RebuildColorIndex()
		}

		cmd.Printf("Favorites imported successfully from %s. Total favorites: %d\n", filePath, len(importedFavorites))
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
	importCmd.AddCommand(importConfigCmd)
	importCmd.AddCommand(importFavoritesCmd)

	importFavoritesCmd.Flags().StringVar(&favFile, "file", "", "Input file path")
}
