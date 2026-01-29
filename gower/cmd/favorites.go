package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gower/internal/utils"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var favoritesCmd = &cobra.Command{
	Use:   "favorites",
	Short: "Manage favorite wallpapers",
}

var favoritesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all favorited wallpapers",
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		favorites, err := loadFavorites()
		if err != nil {
			fmt.Printf("Error loading favorites: %v\n", err)
			return
		}

		if len(favorites) == 0 {
			fmt.Println("No favorite wallpapers yet.")
			return
		}

		for _, fav := range favorites {
			fmt.Printf("ID: %s, URL: %s, Source: %s\n", fav.ID, fav.URL, fav.Source)
		}
	},
}

var favoritesAddCmd = &cobra.Command{
	Use:   "add <id>",
	Short: "Add a wallpaper to favorites",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		wallpaperID := args[0]

		// In a real scenario, you would fetch the wallpaper details using its ID
		// from the original provider. For now, we'll create a dummy wallpaper.
		// This part needs to be integrated with the explore/download logic.
		dummyWallpaper := models.Wallpaper{
			ID:     wallpaperID,
			URL:    fmt.Sprintf("http://example.com/wallpaper/%s", wallpaperID),
			Source: "unknown", // Source should be determined when fetching details
		}

		favorites, err := loadFavorites()
		if err != nil {
			fmt.Printf("Error loading favorites: %v\n", err)
			return
		}

		for _, fav := range favorites {
			if fav.ID == wallpaperID {
				fmt.Printf("Wallpaper %s is already in favorites.\n", wallpaperID)
				return
			}
		}

		favorites = append(favorites, dummyWallpaper)
		if err := saveFavorites(favorites); err != nil {
			fmt.Printf("Error saving favorites: %v\n", err)
			return
		}
		fmt.Printf("Wallpaper %s added to favorites.\n", wallpaperID)
	},
}

var favoritesRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a wallpaper from favorites",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		wallpaperID := args[0]

		favorites, err := loadFavorites()
		if err != nil {
			fmt.Printf("Error loading favorites: %v\n", err)
			return
		}

		found := false
		newFavorites := []models.Wallpaper{}
		for _, fav := range favorites {
			if fav.ID == wallpaperID {
				found = true
			} else {
				newFavorites = append(newFavorites, fav)
			}
		}

		if !found {
			fmt.Printf("Wallpaper %s not found in favorites.\n", wallpaperID)
			return
		}

		if err := saveFavorites(newFavorites); err != nil {
			fmt.Printf("Error saving favorites: %v\n", err)
			return
		}
		fmt.Printf("Wallpaper %s removed from favorites.\n", wallpaperID)
	},
}

var favoritesExportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "Export favorite wallpapers to a file",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		favorites, err := loadFavorites()
		if err != nil {
			fmt.Printf("Error loading favorites: %v\n", err)
			return
		}

		data, err := json.MarshalIndent(favorites, "", "  ")
		if err != nil {
			fmt.Printf("Error marshalling favorites: %v\n", err)
			return
		}

		if len(args) > 0 {
			if err := ioutil.WriteFile(args[0], data, 0644); err != nil {
				fmt.Printf("Error exporting favorites to %s: %v\n", args[0], err)
				return
			}
			fmt.Printf("Favorites exported to %s.\n", args[0])
		} else {
			fmt.Println(string(data))
		}
	},
}

var favoritesImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import favorite wallpapers from a file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		filePath := args[0]

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading import file %s: %v\n", filePath, err)
			return
		}

		var importedFavorites []models.Wallpaper
		if err := json.Unmarshal(data, &importedFavorites); err != nil {
			fmt.Printf("Error unmarshalling import file: %v\n", err)
			return
		}

		// Load existing favorites to merge
		existingFavorites, err := loadFavorites()
		if err != nil {
			fmt.Printf("Error loading existing favorites: %v\n", err)
			return
		}

		// Simple merge: add new ones, overwrite if ID exists (or skip if we want to avoid duplicates)
		// For simplicity, let's just append and then deduplicate.
		// A more robust solution would involve a map for faster lookups.
		mergedFavorites := make(map[string]models.Wallpaper)
		for _, fav := range existingFavorites {
			mergedFavorites[fav.ID] = fav
		}
		for _, fav := range importedFavorites {
			mergedFavorites[fav.ID] = fav // Overwrite if ID exists, or add new
		}

		finalFavorites := []models.Wallpaper{}
		for _, fav := range mergedFavorites {
			finalFavorites = append(finalFavorites, fav)
		}

		if err := saveFavorites(finalFavorites); err != nil {
			fmt.Printf("Error saving merged favorites: %v\n", err)
			return
		}
		fmt.Printf("Favorites imported successfully from %s. Total favorites: %d\n", filePath, len(finalFavorites))
	},
}

func getFavoritesPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gower", "data", "favorites.json"), nil
}

func loadFavorites() ([]models.Wallpaper, error) {
	path, err := getFavoritesPath()
	if err != nil {
		return nil, err
	}

	var favorites []models.Wallpaper
	manager := utils.NewSecureJSONManager()
	// If file doesn't exist, return empty list without error
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []models.Wallpaper{}, nil
	}

	if err := manager.ReadJSON(path, &favorites); err != nil {
		return nil, err
	}
	return favorites, nil
}

func saveFavorites(favorites []models.Wallpaper) error {
	path, err := getFavoritesPath()
	if err != nil {
		return err
	}
	manager := utils.NewSecureJSONManager()
	return manager.WriteJSON(path, favorites)
}

func init() {
	rootCmd.AddCommand(favoritesCmd)

	favoritesCmd.AddCommand(favoritesListCmd)
	favoritesCmd.AddCommand(favoritesAddCmd)
	favoritesCmd.AddCommand(favoritesRemoveCmd)
	favoritesCmd.AddCommand(favoritesExportCmd)
	favoritesCmd.AddCommand(favoritesImportCmd)
}