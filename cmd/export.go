package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export application data",
	Long:  `Export various application data like configuration, favorites, and feed.`,
}

var exportAllCmd = &cobra.Command{
	Use:   "all [destination_dir]",
	Short: "Export all application data",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()

		destDir := fmt.Sprintf("gower_export_%s", time.Now().Format("20060102_150405"))
		if len(args) > 0 {
			destDir = args[0]
		}

		if err := os.MkdirAll(destDir, 0755); err != nil {
			fmt.Printf("Error creating destination directory %s: %v\n", destDir, err)
			return
		}

		// Export Config
		configPath, _ := getConfigPath()
		configData, err := ioutil.ReadFile(configPath)
		if err != nil {
			fmt.Printf("Warning: Could not read config.json: %v\n", err)
		} else {
			if err := ioutil.WriteFile(filepath.Join(destDir, "config.json"), configData, 0644); err != nil {
				fmt.Printf("Error exporting config.json: %v\n", err)
			} else {
				fmt.Printf("Exported config.json to %s\n", filepath.Join(destDir, "config.json"))
			}
		}

		// Export Favorites
		favoritesPath, _ := getFavoritesPath()
		favoritesData, err := ioutil.ReadFile(favoritesPath)
		if err != nil {
			fmt.Printf("Warning: Could not read favorites.json: %v\n", err)
		} else {
			if err := ioutil.WriteFile(filepath.Join(destDir, "favorites.json"), favoritesData, 0644); err != nil {
				fmt.Printf("Error exporting favorites.json: %v\n", err)
			} else {
				fmt.Printf("Exported favorites.json to %s\n", filepath.Join(destDir, "favorites.json"))
			}
		}

		// Export Feed
		// Need to load config to initialize controller
		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config for feed export: %v\n", err)
			return
		}
		controller := core.NewController(cfg)
		feedWallpapers, err := controller.GetFeedWallpapers()
		if err != nil {
			fmt.Printf("Warning: Could not get feed wallpapers: %v\n", err)
		} else {
			feedData, err := json.MarshalIndent(feedWallpapers, "", "  ")
			if err != nil {
				fmt.Printf("Error marshalling feed data: %v\n", err)
			} else {
				if err := ioutil.WriteFile(filepath.Join(destDir, "feed.json"), feedData, 0644); err != nil {
					fmt.Printf("Error exporting feed.json: %v\n", err)
				} else {
					fmt.Printf("Exported feed.json to %s\n", filepath.Join(destDir, "feed.json"))
				}
			}
		}

		fmt.Printf("All data exported to directory: %s\n", destDir)
	},
}

var exportConfigCmd = &cobra.Command{
	Use:   "config [file]",
	Short: "Export configuration",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		configPath, _ := getConfigPath()
		data, err := ioutil.ReadFile(configPath)
		if err != nil {
			fmt.Printf("Error reading configuration: %v\n", err)
			return
		}

		if len(args) > 0 {
			if err := ioutil.WriteFile(args[0], data, 0644); err != nil {
				fmt.Printf("Error exporting config to %s: %v\n", args[0], err)
				return
			}
			fmt.Printf("Configuration exported to: %s\n", args[0])
		} else {
			fmt.Println(string(data))
		}
	},
}

var exportFeedCmd = &cobra.Command{
	Use:   "feed [file]",
	Short: "Export wallpaper feed/history",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		cfg, err := loadConfig()
		if err != nil {
			fmt.Printf("Error loading config for feed export: %v\n", err)
			return
		}
		controller := core.NewController(cfg)
		feedWallpapers, err := controller.GetFeedWallpapers()
		if err != nil {
			fmt.Printf("Error getting feed wallpapers: %v\n", err)
			return
		}

		data, err := json.MarshalIndent(feedWallpapers, "", "  ")
		if err != nil {
			fmt.Printf("Error marshalling feed data: %v\n", err)
			return
		}

		if len(args) > 0 {
			if err := ioutil.WriteFile(args[0], data, 0644); err != nil {
				fmt.Printf("Error exporting feed to %s: %v\n", args[0], err)
				return
			}
			fmt.Printf("Feed exported to: %s\n", args[0])
		} else {
			fmt.Println(string(data))
		}
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.AddCommand(exportAllCmd)
	exportCmd.AddCommand(exportConfigCmd)
	exportCmd.AddCommand(exportFeedCmd)
}
