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
		if err := ensureConfig(); err != nil {
			cmd.Println(err)
			return
		}
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
				cmd.PrintErrf("Error loading color palette: %v\n", err)
				return
			}
			if len(palette) == 0 {
				cmd.PrintErrln("Error: no color palette available (run 'gower favorites analyze' first)")
				return
			}
			targetBucket := controller.ColorManager.FindNearestColorInPalette(favColor, palette)
			filtered := make([]core.FavoriteWallpaper, 0, len(favorites))
			for _, fav := range favorites {
				favBucket := controller.ColorManager.FindNearestColorInPalette(fav.Color, palette)
				if favBucket == targetBucket {
					filtered = append(filtered, fav)
				}
			}
			favorites = filtered
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
		// ui.items_per_page actúa como default de --limit, salvo que el usuario
		// lo fije explícitamente en la línea de comandos.
		if !cmd.Flags().Changed("limit") && cfg.UI.ItemsPerPage > 0 {
			favLimit = cfg.UI.ItemsPerPage
		}
		if favPage < 1 {
			cmd.PrintErrln("Error: --page must be >= 1")
			setCommandError(fmt.Errorf("--page must be >= 1"))
			return
		}
		if favLimit < 1 {
			cmd.PrintErrln("Error: --limit must be >= 1")
			setCommandError(fmt.Errorf("--limit must be >= 1"))
			return
		}
		start := (favPage - 1) * favLimit
		if start > len(favorites) {
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
				// El color solo se muestra si hay datos y la configuracion lo pide:
				// el analisis de colores lo rellena, pero los favoritos anadidos
				// a mano pueden no tenerlo.
				color := ""
				if cfg.UI.ShowColors && fav.Color != "" {
					color = fmt.Sprintf(", Color: %s", fav.Color)
				}
				cmd.Printf("ID: %s, URL: %s, Source: %s, Notes: %s%s\n", fav.ID, fav.URL, fav.Source, fav.Notes, color)
			}
		}
	},
}

var favoritesAddCmd = &cobra.Command{
	Use:   "add <id>",
	Short: "Add a wallpaper to favorites",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureConfig(); err != nil {
			cmd.Println(err)
			return
		}
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
		feedPath, err := getFeedPath()
		if err != nil {
			cmd.PrintErrf("Error resolving feed path: %v\n", err)
			setCommandError(err)
			return
		}
		manager := utils.NewSecureJSONManager()

		foundInFeed := false
		// La secuencia leer-quitar-escribir debe ejecutarse bajo el mismo lock
		// que usa el daemon para no perder actualizaciones.
		lockErr := utils.WithFileLock(feedPath, func() error {
			var feed []models.Wallpaper
			if err := manager.ReadJSON(feedPath, &feed); err != nil && !os.IsNotExist(err) {
				return err
			}
			for i, wp := range feed {
				if wp.ID == wallpaperID {
					wallpaperToAdd = wp
					feed = append(feed[:i], feed[i+1:]...)
					foundInFeed = true
					break
				}
			}
			if foundInFeed {
				return manager.WriteJSON(feedPath, feed)
			}
			return nil
		})
		if lockErr != nil {
			cmd.PrintErrf("Error updating feed: %v\n", lockErr)
			setCommandError(lockErr)
			return
		}

		if !foundInFeed {
			// Se permite añadir un ID que aún no está en el feed, pero sin
			// fabricar una URL falsa: queda como entrada desconocida.
			wallpaperToAdd = models.Wallpaper{
				ID:     wallpaperID,
				Source: "unknown",
			}
			cmd.PrintErrf("Warning: wallpaper %s is not in the feed; favorites entry has no URL.\n", wallpaperID)
		} else {
			cmd.Printf("Moved wallpaper %s from feed to favorites.\n", wallpaperID)
		}

		newFav := core.FavoriteWallpaper{Wallpaper: wallpaperToAdd, Notes: favNotes}

		if config.DryRun {
			if !config.Quiet {
				cmd.Printf("[DRY-RUN] Se añadiría %s a favoritos.\n", wallpaperID)
			}
			return
		}

		// Check if we need to save to local folder
		if cfg, err := loadConfig(); err == nil && cfg.Behavior.SaveFavoritesToFolder && cfg.Paths.Wallpapers != "" {
			if err := os.MkdirAll(cfg.Paths.Wallpapers, 0755); err != nil {
				cmd.PrintErrf("Warning: Could not create wallpapers directory: %v\n", err)
			} else {
				ext := filepath.Ext(strings.Split(newFav.URL, "?")[0])
				if ext == "" {
					ext = ".jpg"
				}
				destFilename := utils.SafeFilename(newFav.ID) + ext
				destPath := filepath.Join(cfg.Paths.Wallpapers, destFilename)
				if !utils.IsWithinDir(destPath, cfg.Paths.Wallpapers) {
					cmd.PrintErrf("Warning: refusing to write outside %s\n", cfg.Paths.Wallpapers)
				} else {

					// Copy or Download
					success := false
					if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
						success = true
					} else if _, err := os.Stat(newFav.URL); err == nil {
						if err := copyFile(newFav.URL, destPath); err == nil {
							success = true
						} else {
							cmd.PrintErrf("Warning: Failed to copy local file to favorites folder: %v\n", err)
						}
					} else {
						if err := downloadFile(newFav.URL, destPath); err == nil {
							success = true
						} else {
							cmd.PrintErrf("Warning: Failed to download favorite to folder: %v\n", err)
						}
					}

					if success {
						newFav.URL = destPath // Update URL to point to the persistent local file
					}
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
		if err := ensureConfig(); err != nil {
			cmd.Println(err)
			return
		}
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

		if config.DryRun {
			if !config.Quiet {
				cmd.Printf("[DRY-RUN] Se quitaría %s de favoritos.\n", wallpaperID)
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
		return nil, fmt.Errorf("error reading colors.json: %w", err)
	}
	return &palettes, nil
}

var favoritesGetColorsCmd = &cobra.Command{
	Use:   "get colors",
	Short: "Get the color palette of favorited wallpapers from colors.json",
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureConfig(); err != nil {
			cmd.Println(err)
			return
		}

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
	return writeFileAtomicCmd(dst, sourceFile)
}

func downloadFile(url, dst string) error {
	resp, err := utils.DownloadHTTPClient.Get(url) //nolint:bodyclose // cerrado con utils.CloseResponse
	if err != nil {
		return err
	}
	defer utils.CloseResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
		return fmt.Errorf("unexpected content type: %s", ct)
	}

	return writeFileAtomicCmd(dst, io.LimitReader(resp.Body, utils.MaxImageBytes))
}

// writeFileAtomicCmd escribe en un temporal del mismo directorio y lo renombra
// al final, para no dejar un fichero parcial si la descarga se corta.
func writeFileAtomicCmd(dst string, r io.Reader) error {
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".gower-fav-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, dst)
}
