package cmd

import (
	"os"
	"path/filepath"

	"gower/internal/core"
	"gower/internal/utils"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage application cache",
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean the cache directory",
	Run: func(cmd *cobra.Command, args []string) {
		appDir, _ := core.GetAppDir()
		cacheDir := filepath.Join(appDir, "cache")

		// Remove all contents
		err := os.RemoveAll(cacheDir)
		if err != nil {
			cmd.Printf("Error cleaning cache: %v\n", err)
			utils.Log.Error("Cache clean failed: %v", err)
			return
		}
		// Recreate structure
		os.MkdirAll(filepath.Join(cacheDir, "wallpapers"), 0755)
		os.MkdirAll(filepath.Join(cacheDir, "thumbs"), 0755)

		if !config.Quiet {
			cmd.Println("Cache cleaned successfully.")
		}
		utils.Log.Info("Cache cleaned successfully")
	},
}

var cacheSizeCmd = &cobra.Command{
	Use:   "size",
	Short: "Show cache size",
	Run: func(cmd *cobra.Command, args []string) {
		appDir, _ := core.GetAppDir()
		cacheDir := filepath.Join(appDir, "cache")

		var size int64
		err := filepath.Walk(cacheDir, func(_ string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				size += info.Size()
			}
			return nil
		})

		if err != nil {
			cmd.Printf("Error calculating cache size: %v\n", err)
			return
		}

		if !config.Quiet {
			cmd.Printf("Cache size: %.2f MB\n", float64(size)/1024/1024)
		}
	},
}

var cachePruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove cached files that are no longer in the feed or favorites",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		if !config.Quiet {
			cmd.Println("Scanning for orphaned cache files...")
		}

		// 1. Collect all known wallpapers
		var allWallpapers []models.Wallpaper
		feed, _ := controller.GetFeedWallpapers()
		if feed != nil {
			allWallpapers = append(allWallpapers, feed...)
		}

		favorites, _ := loadFavorites()
		for _, fav := range favorites {
			allWallpapers = append(allWallpapers, fav.Wallpaper)
		}

		state, _ := loadState()
		if state != nil {
			var stateIDs []string
			if len(state.CurrentWallpapers) > 0 {
				stateIDs = append(stateIDs, state.CurrentWallpapers...)
			} else if state.CurrentWallpaperID != "" {
				stateIDs = append(stateIDs, state.CurrentWallpaperID)
			}
			for _, id := range stateIDs {
				wp, err := controller.GetWallpaper(id)
				if err == nil && wp != nil {
					allWallpapers = append(allWallpapers, *wp)
				}
			}
		}

		// 2. Build a set of files to keep
		filesToKeep := make(map[string]bool)
		appDir, _ := core.GetAppDir()
		thumbsCacheDir := filepath.Join(appDir, "cache", "thumbs")

		for _, wp := range allWallpapers {
			if path, found := controller.FindWallpaperCacheFile(wp); found {
				filesToKeep[path] = true
			}
			thumbPath := filepath.Join(thumbsCacheDir, wp.ID+".jpg")
			filesToKeep[thumbPath] = true
		}

		// 3. Walk cache directories and delete unexpected files
		pruneDir := func(dir string) (int, error) {
			count := 0
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				if !filesToKeep[path] {
					if !config.Quiet {
						cmd.Printf("Pruning orphaned file: %s\n", filepath.Base(path))
					}
					if err := os.Remove(path); err == nil {
						count++
					}
				}
				return nil
			})
			return count, err
		}

		wallpapersDeleted, err1 := pruneDir(filepath.Join(appDir, "cache", "wallpapers"))
		thumbsDeleted, err2 := pruneDir(thumbsCacheDir)

		if err1 != nil || err2 != nil {
			cmd.Printf("An error occurred during pruning: err1=%v, err2=%v\n", err1, err2)
			return
		}

		totalDeleted := wallpapersDeleted + thumbsDeleted
		cmd.Printf("Pruning complete. Removed %d orphaned file(s).\n", totalDeleted)
	},
}

func init() {
	systemCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cacheSizeCmd)
	cacheCmd.AddCommand(cachePruneCmd)
}
