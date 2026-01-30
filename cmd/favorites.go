package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	// "strings"

	"gower/internal/utils"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

type FavoriteWallpaper struct {
	models.Wallpaper
	Notes string `json:"notes,omitempty"`
}

var (
	favPage  int
	favLimit int
	favNotes string
	favForce bool
	favFile  string
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

		// Pagination
		start := (favPage - 1) * favLimit
		if start >= len(favorites) {
			start = len(favorites)
		}
		end := start + favLimit
		if end > len(favorites) {
			end = len(favorites)
		}

		pageItems := favorites[start:end]

		if config.JSONOutput {
			data, _ := json.MarshalIndent(pageItems, "", "  ")
			fmt.Println(string(data))
		} else {
			for _, fav := range pageItems {
				fmt.Printf("ID: %s, URL: %s, Source: %s, Notes: %s\n", fav.ID, fav.URL, fav.Source, fav.Notes)
			}
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
		dummyWallpaper := FavoriteWallpaper{
			Wallpaper: models.Wallpaper{
				ID:     wallpaperID,
				URL:    fmt.Sprintf("http://example.com/wallpaper/%s", wallpaperID),
				Source: "unknown",
			},
			Notes: favNotes,
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
		newFavorites := []FavoriteWallpaper{}
		for _, fav := range favorites {
			if fav.ID == wallpaperID {
				found = true
			} else {
				newFavorites = append(newFavorites, fav)
			}
		}

		if !found {
			if !favForce {
				fmt.Printf("Wallpaper %s not found in favorites.\n", wallpaperID)
			}
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
	Use:   "export",
	Short: "Export favorite wallpapers to a file",
	Args:  cobra.NoArgs,
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

		if favFile != "" {
			if err := ioutil.WriteFile(favFile, data, 0644); err != nil {
				fmt.Printf("Error exporting favorites to %s: %v\n", favFile, err)
				return
			}
			fmt.Printf("Favorites exported to %s.\n", favFile)
		} else {
			fmt.Println(string(data))
		}
	},
}

var favoritesImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import favorite wallpapers from a file",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		filePath := favFile
		if filePath == "" {
			fmt.Println("Error: --file flag is required for import")
			os.Exit(1)
		}

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading import file %s: %v\n", filePath, err)
			return
		}

		var importedFavorites []FavoriteWallpaper
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
		mergedFavorites := make(map[string]FavoriteWallpaper)
		for _, fav := range existingFavorites {
			mergedFavorites[fav.ID] = fav
		}
		for _, fav := range importedFavorites {
			mergedFavorites[fav.ID] = fav // Overwrite if ID exists, or add new
		}

		finalFavorites := []FavoriteWallpaper{}
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

func loadFavorites() ([]FavoriteWallpaper, error) {
	path, err := getFavoritesPath()
	if err != nil {
		return nil, err
	}

	var favorites []FavoriteWallpaper
	manager := utils.NewSecureJSONManager()
	// If file doesn't exist, return empty list without error
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []FavoriteWallpaper{}, nil
	}

	if err := manager.ReadJSON(path, &favorites); err != nil {
		return nil, err
	}
	return favorites, nil
}

func saveFavorites(favorites []FavoriteWallpaper) error {
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

	favoritesListCmd.Flags().IntVar(&favPage, "page", 1, "Page number")
	favoritesListCmd.Flags().IntVar(&favLimit, "limit", 10, "Items per page")

	favoritesAddCmd.Flags().StringVar(&favNotes, "notes", "", "Add notes to the favorite")

	favoritesRemoveCmd.Flags().BoolVar(&favForce, "force", false, "Do not return error if not found")

	favoritesExportCmd.Flags().StringVar(&favFile, "file", "", "Output file path")
	favoritesImportCmd.Flags().StringVar(&favFile, "file", "", "Input file path")
}
