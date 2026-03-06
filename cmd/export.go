package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gower/internal/core"

	"github.com/spf13/cobra"
)

var (
	exportFile          string
	exportIncludeImages bool
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export application data",
	Long:  `Export various application data like configuration, favorites, and feed.`,
}

var exportAllCmd = &cobra.Command{
	Use:   "all [destination_dir]",
	Short: "Export all application data",
	Long:  `Export all data. Use --file to export to a ZIP archive, or provide a directory argument for folder export.`,
	Args:  cobra.MaximumNArgs(1), // Keep arg for directory mode backward compatibility
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()

		// ZIP MODE
		if exportFile != "" {
			if err := exportAllToZip(cmd, exportFile, exportIncludeImages); err != nil {
				cmd.Printf("Error creating zip export: %v\n", err)
				return
			}
			cmd.Printf("All data exported to: %s\n", exportFile)
			return
		}

		// DIRECTORY MODE
		destDir := fmt.Sprintf("gower_export_%s", time.Now().Format("20060102_150405"))
		if len(args) > 0 {
			destDir = args[0]
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			cmd.Printf("Error creating destination directory %s: %v\n", destDir, err)
			return
		}

		// Export Config
		configPath, _ := getConfigPath()
		configData, err := os.ReadFile(configPath)
		if err != nil {
			cmd.Printf("Warning: Could not read config.json: %v\n", err)
		} else {
			if err := os.WriteFile(filepath.Join(destDir, "config.json"), configData, 0644); err != nil {
				cmd.Printf("Error exporting config.json: %v\n", err)
			} else {
				cmd.Printf("Exported config.json to %s\n", filepath.Join(destDir, "config.json"))
			}
		}

		// Export Favorites
		favoritesPath, _ := getFavoritesPath()
		favoritesData, err := os.ReadFile(favoritesPath)
		if err != nil {
			cmd.Printf("Warning: Could not read favorites.json: %v\n", err)
		} else {
			if err := os.WriteFile(filepath.Join(destDir, "favorites.json"), favoritesData, 0644); err != nil {
				cmd.Printf("Error exporting favorites.json: %v\n", err)
			} else {
				cmd.Printf("Exported favorites.json to %s\n", filepath.Join(destDir, "favorites.json"))
			}
		}

		// Export Feed
		// Need to load config to initialize controller
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config for feed export: %v\n", err)
			return
		}
		controller := core.NewController(cfg)
		feedWallpapers, err := controller.GetFeedWallpapers()
		if err != nil {
			cmd.Printf("Warning: Could not get feed wallpapers: %v\n", err)
		} else {
			feedData, err := json.MarshalIndent(feedWallpapers, "", "  ")
			if err != nil {
				cmd.Printf("Error marshalling feed data: %v\n", err)
			} else {
				if err := os.WriteFile(filepath.Join(destDir, "feed.json"), feedData, 0644); err != nil {
					cmd.Printf("Error exporting feed.json: %v\n", err)
				} else {
					cmd.Printf("Exported feed.json to %s\n", filepath.Join(destDir, "feed.json"))
				}
			}
		}

		cmd.Printf("All data exported to directory: %s\n", destDir)
	},
}

var exportConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Export configuration",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		configPath, _ := getConfigPath()
		data, err := os.ReadFile(configPath)
		if err != nil {
			cmd.Printf("Error reading configuration: %v\n", err)
			return
		}

		if exportFile != "" {
			if err := os.WriteFile(exportFile, data, 0644); err != nil {
				cmd.Printf("Error exporting config to %s: %v\n", exportFile, err)
				return
			}
			cmd.Printf("Configuration exported to: %s\n", exportFile)
		} else {
			cmd.Println(string(data))
		}
	},
}

var exportFeedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Export wallpaper feed/history",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config for feed export: %v\n", err)
			return
		}
		controller := core.NewController(cfg)
		feedWallpapers, err := controller.GetFeedWallpapers()
		if err != nil {
			cmd.Printf("Error getting feed wallpapers: %v\n", err)
			return
		}

		data, err := json.MarshalIndent(feedWallpapers, "", "  ")
		if err != nil {
			cmd.Printf("Error marshalling feed data: %v\n", err)
			return
		}

		if exportFile != "" {
			if err := os.WriteFile(exportFile, data, 0644); err != nil {
				cmd.Printf("Error exporting feed to %s: %v\n", exportFile, err)
				return
			}
			cmd.Printf("Feed exported to: %s\n", exportFile)
		} else {
			cmd.Println(string(data))
		}
	},
}

var exportFavoritesCmd = &cobra.Command{
	Use:   "favorites",
	Short: "Export favorite wallpapers",
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

		if exportFile != "" {
			if err := os.WriteFile(exportFile, data, 0644); err != nil {
				cmd.Printf("Error exporting favorites to %s: %v\n", exportFile, err)
				return
			}
			cmd.Printf("Favorites exported to %s.\n", exportFile)
		} else {
			cmd.Println(string(data))
		}
	},
}

func exportAllToZip(cmd *cobra.Command, filename string, includeImages bool) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	w := zip.NewWriter(f)
	defer func() { _ = w.Close() }()

	// Helper to add file
	addFile := func(srcPath, zipName string) {
		if srcPath == "" {
			return
		}
		data, err := os.ReadFile(srcPath)
		if err != nil {
			cmd.Printf("Warning: Could not read %s for zip: %v\n", srcPath, err)
			return
		}
		zf, err := w.Create(zipName)
		if err != nil {
			cmd.Printf("Warning: Could not create zip entry %s: %v\n", zipName, err)
			return
		}
		_, _ = zf.Write(data)
	}

	// Config
	p, _ := getConfigPath()
	addFile(p, "config.json")

	// Favorites
	p, _ = getFavoritesPath()
	addFile(p, "favorites.json")

	// Feed
	p, _ = getFeedPath()
	addFile(p, "feed.json")

	// Images
	if includeImages {
		appDir, _ := core.GetAppDir()
		cacheDir := filepath.Join(appDir, "cache", "wallpapers")
		_ = filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(cacheDir, path)
			if err != nil {
				return nil
			}

			// Open file
			srcFile, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer func() { _ = srcFile.Close() }()

			// Create zip entry
			zipPath := filepath.Join("images", rel)
			zf, err := w.Create(zipPath)
			if err != nil {
				return nil
			}
			_, _ = io.Copy(zf, srcFile)
			return nil
		})
	}

	return nil
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.AddCommand(exportAllCmd)
	exportCmd.AddCommand(exportConfigCmd)
	exportCmd.AddCommand(exportFeedCmd)
	exportCmd.AddCommand(exportFavoritesCmd)

	exportAllCmd.Flags().StringVar(&exportFile, "file", "", "Output zip file path")
	exportAllCmd.Flags().BoolVar(&exportIncludeImages, "include-images", false, "Include downloaded images in export")

	exportConfigCmd.Flags().StringVar(&exportFile, "file", "", "Output file path")
	exportFeedCmd.Flags().StringVar(&exportFile, "file", "", "Output file path")
	exportFavoritesCmd.Flags().StringVar(&exportFile, "file", "", "Output file path")
}
