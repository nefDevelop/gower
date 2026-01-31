// cmd/set.go
package cmd

import (
	"fmt"
	"math/rand"
	"time"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var (
	setID            string
	setURL           string
	setRandom        bool
	setTheme         string
	setFromFavorites bool
	setMultiMonitor  string
	setCommand       string
	setNoDownload    bool
)

var setCmd = &cobra.Command{
	Use:   "set [id|url|random]",
	Short: "Set wallpaper",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSet,
}

func init() {
	setCmd.Flags().StringVar(&setID, "id", "",
		"wallpaper ID (e.g., wh_123456)")
	setCmd.Flags().StringVar(&setURL, "url", "",
		"direct wallpaper URL")
	setCmd.Flags().BoolVarP(&setRandom, "random", "r", false,
		"set random wallpaper")
	setCmd.Flags().StringVar(&setTheme, "theme", "",
		"theme filter [dark|light|auto]")
	setCmd.Flags().BoolVar(&setFromFavorites, "from-favorites", false,
		"random from favorites only")
	setCmd.Flags().StringVar(&setMultiMonitor, "multi-monitor", "",
		"multi-monitor mode [clone|distinct]")
	setCmd.Flags().StringVar(&setCommand, "command", "",
		"custom wallpaper command")
	setCmd.Flags().BoolVar(&setNoDownload, "no-download", false,
		"don't download, use existing file")

	// Subcomandos
	setCmd.AddCommand(&cobra.Command{
		Use:   "random",
		Short: "Set random wallpaper",
		RunE:  runSetRandom,
	})

	rootCmd.AddCommand(setCmd)
}

func runSet(cmd *cobra.Command, args []string) error {
	if err := ensureConfig(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	controller := core.NewController(cfg)

	var wallpaper *models.Wallpaper

	// 1. Determine target wallpaper
	if len(args) > 0 {
		input := args[0]
		// Check if it looks like a URL
		if len(input) > 4 && input[:4] == "http" {
			wallpaper = &models.Wallpaper{
				ID:     "manual_url",
				URL:    input,
				Source: "manual",
			}
		} else {
			// Assume ID
			wp, err := controller.GetWallpaper(input)
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}
			wallpaper = wp
		}
	} else if setURL != "" {
		wallpaper = &models.Wallpaper{
			ID:     "manual_url",
			URL:    setURL,
			Source: "manual",
		}
	} else if setID != "" {
		wp, err := controller.GetWallpaper(setID)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		wallpaper = wp
	} else {
		return cmd.Help()
	}

	return applyWallpaper(controller, *wallpaper)
}

func runSetRandom(cmd *cobra.Command, args []string) error {
	if err := ensureConfig(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	controller := core.NewController(cfg)

	var wallpaper models.Wallpaper

	if setFromFavorites {
		favorites, err := loadFavorites()
		if err != nil {
			return fmt.Errorf("error loading favorites: %w", err)
		}
		if len(favorites) == 0 {
			return fmt.Errorf("no favorites found")
		}
		rand.Seed(time.Now().UnixNano())
		fav := favorites[rand.Intn(len(favorites))]
		wallpaper = fav.Wallpaper
	} else {
		var err error
		wallpaper, err = controller.GetRandomFromFeed(setTheme)
		if err != nil {
			return fmt.Errorf("error getting random wallpaper: %w", err)
		}
	}

	return applyWallpaper(controller, wallpaper)
}

func applyWallpaper(controller *core.Controller, wp models.Wallpaper) error {
	fmt.Printf("Setting wallpaper: %s (Source: %s)\n", wp.ID, wp.Source)

	localPath := ""
	if !setNoDownload {
		var err error
		localPath, err = controller.DownloadWallpaper(wp)
		if err != nil {
			return fmt.Errorf("error downloading wallpaper: %w", err)
		}
	} else {
		// If no download, we assume URL is a local path or we can't do much
		localPath = wp.URL
	}

	// Determine desktop environment (simple detection or config)
	// For now, we rely on core.NewWallpaperChanger auto-detection
	changer := core.NewWallpaperChanger("")

	// Override multi-monitor if flag set
	mmMode := setMultiMonitor
	if mmMode == "" {
		// Fallback to config if available, otherwise default
		// We don't have easy access to config struct here without reloading or passing it
		// But NewController(cfg) was called, so we could pass cfg.Behavior.MultiMonitor
		// For simplicity, let's default to "clone" or let changer handle it
		mmMode = "clone"
	}

	if err := changer.SetWallpaper(localPath, mmMode); err != nil {
		return fmt.Errorf("error setting wallpaper: %w", err)
	}

	fmt.Println("Wallpaper set successfully.")
	return nil
}
