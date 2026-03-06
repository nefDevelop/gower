package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gower/internal/core"
	"gower/internal/utils"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

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
				var filtered []core.FavoriteWallpaper
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

		newFav := core.FavoriteWallpaper{Wallpaper: wallpaperToAdd, Notes: favNotes}

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
			_ = core.NewController(cfg).RebuildColorIndex()
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
		newFavorites := []core.FavoriteWallpaper{}
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
			_ = core.NewController(cfg).RebuildColorIndex()
		}

		cmd.Printf("Wallpaper %s removed from favorites.\n", wallpaperID)
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
				if strings.Contains(msg, "Error") {
					cmd.Printf("  %s %s\n", colorize(symbolCross, colorRed), msg)
				} else if strings.Contains(msg, "Downloading") {
					cmd.Printf("  %s %s\n", colorize("⬇", colorCyan), msg)
				} else if strings.Contains(msg, "Deleting") || strings.Contains(msg, "Removing") {
					cmd.Printf("  %s %s\n", colorize("🗑", colorRed), msg)
				} else if strings.Contains(msg, "Skipping") {
					cmd.Printf("  %s %s\n", colorize("⏭", colorYellow), msg)
				} else {
					cmd.Printf("%s %s\n", colorize("::", colorBlue), msg)
				}
			}
		}

		if err := controller.AnalyzeFavorites(favAll, favForce, progress); err != nil {
			cmd.Printf("Error analyzing favorites: %v\n", err)
			return
		}
		cmd.Println(colorize(symbolCheck+" Analysis complete.", colorGreen))
	},
}

func getFavoritesPath() (string, error) {
	appDir, err := core.GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "data", "favorites.json"), nil
}

func getFeedPath() (string, error) {
	appDir, err := core.GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "data", "feed.json"), nil
}

func loadFavorites() ([]core.FavoriteWallpaper, error) {
	path, err := getFavoritesPath()
	if err != nil {
		return nil, err
	}

	var favorites []core.FavoriteWallpaper
	manager := utils.NewSecureJSONManager()
	// If file doesn't exist, return empty list without error
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []core.FavoriteWallpaper{}, nil
	}

	if err := manager.ReadJSON(path, &favorites); err != nil {
		return nil, err
	}
	return favorites, nil
}

func saveFavorites(favorites []core.FavoriteWallpaper) error {
	path, err := getFavoritesPath()
	if err != nil {
		return err
	}
	manager := utils.NewSecureJSONManager()
	return manager.WriteJSON(path, favorites)
}

// ColorPalettes struct matches the structure of colors.json
type ColorPalettes struct {
	FeedPalette      []string `json:"feed_palette"`
	FavoritesPalette []string `json:"favorites_palette"`
}

// getColorsFilePath returns the path to the colors.json file
func getColorsFilePath() (string, error) {
	appDir, err := core.GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "data", "colors.json"), nil
}

// loadColorPalettes loads the ColorPalettes from colors.json
func loadColorPalettes() (*ColorPalettes, error) {
	path, err := getColorsFilePath()
	if err != nil {
		return nil, err
	}

	var palettes ColorPalettes
	manager := utils.NewSecureJSONManager()

	// If the file doesn't exist, return empty palettes without error
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ColorPalettes{
			FeedPalette:      []string{},
			FavoritesPalette: []string{},
		}, nil
	}

	if err := manager.ReadJSON(path, &palettes); err != nil {
		return nil, fmt.Errorf("error reading colors.json: %v", err)
	}
	return &palettes, nil
}

var favoritesGetColorsCmd = &cobra.Command{
	Use:   "get colors",
	Short: "Get the color palette of favorited wallpapers from colors.json",
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig() // Ensure config is loaded or initialized

		palettes, err := loadColorPalettes()
		if err != nil {
			cmd.Printf("Error loading color palettes: %v\n", err)
			return
		}

		if config.JSONOutput {
			data, _ := json.MarshalIndent(palettes.FavoritesPalette, "", "  ")
			cmd.Println(string(data))
		} else {
			if len(palettes.FavoritesPalette) == 0 {
				cmd.Println("No favorite colors found in palette.")
				return
			}
			for _, color := range palettes.FavoritesPalette {
				cmd.Println(color)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(favoritesCmd)

	favoritesCmd.AddCommand(favoritesListCmd)
	favoritesCmd.AddCommand(favoritesAddCmd)
	favoritesCmd.AddCommand(favoritesRemoveCmd)
	favoritesCmd.AddCommand(favoritesAnalyzeCmd)
	favoritesCmd.AddCommand(favoritesGetColorsCmd)

	favoritesListCmd.Flags().IntVar(&favPage, "page", 1, "Page number")
	favoritesListCmd.Flags().IntVar(&favLimit, "limit", 10, "Items per page")
	favoritesListCmd.Flags().StringVar(&favColor, "color", "", "Filter by color (hex)")

	favoritesAddCmd.Flags().StringVar(&favNotes, "notes", "", "Add notes to the favorite")

	favoritesRemoveCmd.Flags().BoolVar(&favForce, "force", false, "Do not return error if not found")

	favoritesAnalyzeCmd.Flags().BoolVar(&favAll, "all", false, "Analyze all items, not just new ones")
	favoritesAnalyzeCmd.Flags().BoolVar(&favForce, "force", false, "Force regeneration of thumbnails")
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func downloadFile(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	return err
}
