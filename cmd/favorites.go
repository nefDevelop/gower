package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"gower/internal/core"
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
	favColor string
	favForce bool
	favAll   bool
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
			cmd.Printf("Error loading favorites: %v\n", err)
			return
		}

		// Load controller to access color manager and palette
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		if favColor != "" {
			// Use favorites palette
			_, palette, err := controller.LoadColorPalettes()
			if err != nil {
				cmd.Printf("Warning: could not load color palette: %v\n", err)
			} else {
				var filtered []FavoriteWallpaper
				for _, fav := range favorites {
					targetBucket := controller.ColorManager.FindNearestColorInPalette(favColor, palette)
					favBucket := controller.ColorManager.FindNearestColorInPalette(fav.Color, palette)
					if favBucket == targetBucket {
						filtered = append(filtered, fav)
					}
				}
				favorites = filtered
			}
		}

		if len(favorites) == 0 {
			msg := "No favorite wallpapers yet."
			if favColor != "" {
				msg = "No favorite wallpapers found matching the color."
			}
			cmd.Println(msg)
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
			cmd.Println(string(data))
		} else {
			for _, fav := range pageItems {
				cmd.Printf("ID: %s, URL: %s, Source: %s, Notes: %s\n", fav.ID, fav.URL, fav.Source, fav.Notes)
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

		favorites, err := loadFavorites()
		if err != nil {
			cmd.Printf("Error loading favorites: %v\n", err)
			return
		}

		for _, fav := range favorites {
			if fav.ID == wallpaperID {
				cmd.Printf("Wallpaper %s is already in favorites.\n", wallpaperID)
				return
			}
		}

		// Check if wallpaper exists in feed
		var wallpaperToAdd models.Wallpaper
		feedPath, _ := getFeedPath()
		var feed []models.Wallpaper
		manager := utils.NewSecureJSONManager()

		// Try to read feed, ignore error if not exists
		_ = manager.ReadJSON(feedPath, &feed)

		foundInFeed := false
		for i, wp := range feed {
			if wp.ID == wallpaperID {
				wallpaperToAdd = wp
				// Remove from feed
				feed = append(feed[:i], feed[i+1:]...)
				foundInFeed = true
				break
			}
		}

		if foundInFeed {
			if err := manager.WriteJSON(feedPath, feed); err != nil {
				cmd.Printf("Warning: Could not update feed: %v\n", err)
			}
			cmd.Printf("Moved wallpaper %s from feed to favorites.\n", wallpaperID)
		} else {
			// Fallback to dummy
			wallpaperToAdd = models.Wallpaper{
				ID:     wallpaperID,
				URL:    fmt.Sprintf("http://example.com/wallpaper/%s", wallpaperID),
				Source: "unknown",
			}
		}

		newFav := FavoriteWallpaper{Wallpaper: wallpaperToAdd, Notes: favNotes}

		// Check if we need to save to local folder
		if cfg, err := loadConfig(); err == nil && cfg.Behavior.SaveFavoritesToFolder && cfg.Paths.Wallpapers != "" {
			// Ensure directory exists
			if err := os.MkdirAll(cfg.Paths.Wallpapers, 0755); err != nil {
				cmd.Printf("Warning: Could not create wallpapers directory: %v\n", err)
			} else {
				// Determine filename
				ext := filepath.Ext(newFav.URL)
				if ext == "" {
					ext = ".jpg"
				}
				destFilename := fmt.Sprintf("%s%s", newFav.ID, ext)
				destPath := filepath.Join(cfg.Paths.Wallpapers, destFilename)

				// Copy or Download
				success := false
				if _, err := os.Stat(destPath); err == nil {
					// Already exists
					success = true
				} else {
					// Check if URL is local file
					if _, err := os.Stat(newFav.URL); err == nil {
						// It's a local file, copy it
						if err := copyFile(newFav.URL, destPath); err == nil {
							success = true
						} else {
							cmd.Printf("Warning: Failed to copy local file to favorites folder: %v\n", err)
						}
					} else {
						// It's a URL, download it
						if err := downloadFile(newFav.URL, destPath); err == nil {
							success = true
						} else {
							cmd.Printf("Warning: Failed to download favorite to folder: %v\n", err)
						}
					}
				}

				if success {
					newFav.URL = destPath // Update URL to point to the persistent local file
				}
			}
		}

		favorites = append(favorites, newFav)

		if err := saveFavorites(favorites); err != nil {
			cmd.Printf("Error saving favorites: %v\n", err)
			return
		}

		if cfg, err := loadConfig(); err == nil {
			core.NewController(cfg).RebuildColorIndex()
		}

		cmd.Printf("Wallpaper %s added to favorites list.\n", wallpaperID)
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
			cmd.Printf("Error loading favorites: %v\n", err)
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
				cmd.Printf("Wallpaper %s not found in favorites.\n", wallpaperID)
			}
			return
		}

		if err := saveFavorites(newFavorites); err != nil {
			cmd.Printf("Error saving favorites: %v\n", err)
			return
		}

		if cfg, err := loadConfig(); err == nil {
			core.NewController(cfg).RebuildColorIndex()
		}

		cmd.Printf("Wallpaper %s removed from favorites.\n", wallpaperID)
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
			cmd.Printf("Error loading favorites: %v\n", err)
			return
		}

		data, err := json.MarshalIndent(favorites, "", "  ")
		if err != nil {
			cmd.Printf("Error marshalling favorites: %v\n", err)
			return
		}

		if favFile != "" {
			if err := ioutil.WriteFile(favFile, data, 0644); err != nil {
				cmd.Printf("Error exporting favorites to %s: %v\n", favFile, err)
				return
			}
			cmd.Printf("Favorites exported to %s.\n", favFile)
		} else {
			cmd.Println(string(data))
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
			cmd.Println("Error: --file flag is required for import")
			os.Exit(1)
		}

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			cmd.Printf("Error reading import file %s: %v\n", filePath, err)
			return
		}

		var importedFavorites []FavoriteWallpaper
		if err := json.Unmarshal(data, &importedFavorites); err != nil {
			cmd.Printf("Error unmarshalling import file: %v\n", err)
			return
		}

		if err := saveFavorites(importedFavorites); err != nil {
			cmd.Printf("Error saving favorites: %v\n", err)
			return
		}

		if cfg, err := loadConfig(); err == nil {
			core.NewController(cfg).RebuildColorIndex()
		}

		cmd.Printf("Favorites imported successfully from %s. Total favorites: %d\n", filePath, len(importedFavorites))
	},
}

var favoritesAnalyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze favorites items",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		cmd.Println("Analyzing favorites...")
		progress := func(msg string) {
			if !config.Quiet {
				cmd.Println(msg)
			}
		}

		if err := controller.AnalyzeFavorites(favAll, favForce, progress); err != nil {
			cmd.Printf("Error analyzing favorites: %v\n", err)
			return
		}
		cmd.Println("Analysis complete.")
	},
}

func getFavoritesPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gower", "data", "favorites.json"), nil
}

func getFeedPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".gower", "data", "feed.json"), nil
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
	favoritesCmd.AddCommand(favoritesAnalyzeCmd)

	favoritesListCmd.Flags().IntVar(&favPage, "page", 1, "Page number")
	favoritesListCmd.Flags().IntVar(&favLimit, "limit", 10, "Items per page")
	favoritesListCmd.Flags().StringVar(&favColor, "color", "", "Filter by color (hex)")

	favoritesAddCmd.Flags().StringVar(&favNotes, "notes", "", "Add notes to the favorite")

	favoritesRemoveCmd.Flags().BoolVar(&favForce, "force", false, "Do not return error if not found")

	favoritesAnalyzeCmd.Flags().BoolVar(&favAll, "all", false, "Analyze all items, not just new ones")
	favoritesAnalyzeCmd.Flags().BoolVar(&favForce, "force", false, "Force regeneration of thumbnails")

	favoritesExportCmd.Flags().StringVar(&favFile, "file", "", "Output file path")
	favoritesImportCmd.Flags().StringVar(&favFile, "file", "", "Input file path")
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func downloadFile(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
