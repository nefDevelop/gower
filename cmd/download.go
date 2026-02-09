package cmd

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var (
	downloadOutput        string
	downloadRandom        bool
	downloadTheme         string
	downloadFromFavorites bool
	downloadTag           bool
	downloadToCollection  bool
)

var downloadCmd = &cobra.Command{
	Use:   "download [id|url|random]",
	Short: "Download a wallpaper",
	Long:  `Download a wallpaper to the cache or a specific directory.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDownload,
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().StringVarP(&downloadOutput, "output", "o", "", "Output file path or directory")
	downloadCmd.Flags().BoolVarP(&downloadRandom, "random", "r", false, "Download a random wallpaper")
	downloadCmd.Flags().StringVar(&downloadTheme, "theme", "", "Theme filter for random [dark|light]")
	downloadCmd.Flags().BoolVar(&downloadFromFavorites, "from-favorites", false, "Random from favorites only")
	downloadCmd.Flags().BoolVar(&downloadTag, "tag", false, "Append theme tag [d]/[l] to filename")
	downloadCmd.Flags().BoolVar(&downloadToCollection, "to-collection", false, "Save to the collection folder defined in config (paths.wallpapers)")
}

func runDownload(cmd *cobra.Command, args []string) error {
	if err := ensureConfig(); err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	controller := core.NewController(cfg)

	var wallpaper *models.Wallpaper

	// Determine target
	if len(args) > 0 {
		input := args[0]
		if input == "random" {
			return runDownloadRandom(cmd, controller, cfg)
		}

		if len(input) > 4 && input[:4] == "http" {
			wallpaper = &models.Wallpaper{
				ID:     "manual_url",
				URL:    input,
				Source: "manual",
			}
		} else {
			wp, err := controller.GetWallpaper(input)
			if err != nil {
				return fmt.Errorf("error finding wallpaper: %w", err)
			}
			wallpaper = wp
		}
	} else if downloadRandom {
		return runDownloadRandom(cmd, controller, cfg)
	} else {
		return cmd.Help()
	}

	return performDownload(cmd, controller, *wallpaper, cfg)
}

func runDownloadRandom(cmd *cobra.Command, controller *core.Controller, cfg *models.Config) error {
	var wallpaper models.Wallpaper
	var err error

	if downloadFromFavorites {
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
		wallpaper, err = controller.GetRandomFromFeed(downloadTheme)
		if err != nil {
			return fmt.Errorf("error getting random wallpaper: %w", err)
		}
	}

	return performDownload(cmd, controller, wallpaper, cfg)
}

func performDownload(cmd *cobra.Command, controller *core.Controller, wp models.Wallpaper, cfg *models.Config) error {
	if !config.Quiet {
		cmd.Printf("Downloading wallpaper: %s (Source: %s)\n", wp.ID, wp.Source)
	}

	cachePath, err := controller.DownloadWallpaper(wp)
	if err != nil {
		return fmt.Errorf("error downloading: %w", err)
	}

	if !config.Quiet {
		cmd.Printf("Downloaded to cache: %s\n", cachePath)
	}

	// Determine final destination
	targetPath := downloadOutput
	if downloadToCollection {
		targetPath = cfg.Paths.Wallpapers
	} else if targetPath == "" && cfg.Paths.Wallpapers != "" && downloadOutput == "" && !downloadToCollection {
		targetPath = cfg.Paths.Wallpapers
	}

	if targetPath != "" {
		src, err := os.Open(cachePath)
		if err != nil {
			return err
		}
		defer src.Close()

		outPath := targetPath
		info, err := os.Stat(outPath)
		if err == nil && info.IsDir() {
			outPath = filepath.Join(outPath, filepath.Base(cachePath))

			if downloadTag && wp.Theme != "" {
				tag := ""
				switch wp.Theme {
				case "dark", "d":
					tag = "[d]"
				case "light", "l":
					tag = "[l]"
				}
				if tag != "" {
					ext := filepath.Ext(outPath)
					name := strings.TrimSuffix(outPath, ext)
					outPath = name + " " + tag + ext
				}
			}
		}

		dst, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
		if !config.Quiet {
			cmd.Printf("Saved to: %s\n", outPath)
		}
	}

	return nil
}
