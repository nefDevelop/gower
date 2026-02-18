package cmd

import (
	"fmt"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var (
	wpDelete bool
	wpFile   bool
	wpForce  bool
)

var wallpaperCmd = &cobra.Command{
	Use:   "wallpaper <id>",
	Short: "View or manage a specific wallpaper",
	Long:  `Displays detailed information about a specific wallpaper. Can also be used to remove it from the feed and/or delete the physical file.`,
	Example: `  # View details
  gower wallpaper wh_12345

  # Delete from feed and disk
  gower wallpaper image.jpg --delete --file`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := ensureConfig(); err != nil {
			cmd.Println(err)
			return
		}
		id := args[0]

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		wp, err := controller.GetWallpaper(id)
		if err != nil {
			cmd.Printf("Error: %v\n", err)
			return
		}

		if wpDelete {
			// --- Deletion Logic ---
			if wpFile && wp.Source == "local" && !wpForce {
				cmd.Printf("WARNING: This will permanently delete the file from your disk: %s\n", wp.URL)
				cmd.Print("Are you sure? (y/N): ")
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					cmd.Println("Operation cancelled.")
					return
				}
			}

			// The controller function handles both feed removal and optional file deletion
			if err := controller.DeleteWallpaper(id, wpFile); err != nil {
				cmd.Printf("Error processing wallpaper: %v\n", err)
				return
			}

			msg := fmt.Sprintf("Wallpaper %s removed from feed.", id)
			if wpFile {
				if wp.Source == "local" {
					msg += " File deleted from disk."
				} else {
					msg += " Cached file deleted."
				}
			}
			cmd.Println(msg)

		} else {
			// --- Display Logic ---
			if wpFile || wpForce {
				cmd.Println("Error: --file and --force flags can only be used with --delete.")
				return
			}
			displaySingleWallpaper(cmd, *wp)
		}
	},
}

func displaySingleWallpaper(cmd *cobra.Command, wp models.Wallpaper) {
	if config.JSONOutput {
		displayJSON(cmd, wp)
	} else {
		cmd.Printf("Details for Wallpaper: %s\n", wp.ID)
		cmd.Printf("  Source:    %s\n", wp.Source)
		cmd.Printf("  URL:       %s\n", wp.URL)
		cmd.Printf("  Path:      %s\n", wp.Path)
		cmd.Printf("  Dimension: %s\n", wp.Dimension)
		cmd.Printf("  Theme:     %s\n", wp.Theme)
		cmd.Printf("  Color:     %s\n", wp.Color)
		cmd.Printf("  Seen:      %t\n", wp.Seen)
	}
}

func init() {
	rootCmd.AddCommand(wallpaperCmd)
	wallpaperCmd.Flags().BoolVar(&wpDelete, "delete", false, "Remove the wallpaper from the feed")
	wallpaperCmd.Flags().BoolVar(&wpFile, "file", false, "Delete the physical file (use with --delete)")
	wallpaperCmd.Flags().BoolVar(&wpForce, "force", false, "Force file deletion without confirmation (use with --delete)")
}
