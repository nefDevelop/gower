package cmd

import (
	"gower/internal/core"

	"github.com/spf13/cobra"
)

var blacklistCmd = &cobra.Command{
	Use:   "blacklist",
	Short: "Manage the wallpaper blacklist",
	Long:  `Add, remove, or list wallpapers in the blacklist.`,
}

var blacklistAddCmd = &cobra.Command{
	Use:   "add <id>",
	Short: "Add a wallpaper to the blacklist",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		id := args[0]

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		if err := controller.AddToBlacklist(id); err != nil {
			cmd.Printf("Error adding to blacklist: %v\n", err)
			return
		}

		// Remove from feed if present
		if err := controller.RemoveFromFeed(id); err != nil {
			cmd.Printf("Warning: Failed to remove from feed: %v\n", err)
		}

		cmd.Printf("Wallpaper %s added to blacklist.\n", id)
	},
}

var blacklistRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a wallpaper from the blacklist",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		id := args[0]

		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		if err := controller.RemoveFromBlacklist(id); err != nil {
			cmd.Printf("Error removing from blacklist: %v\n", err)
			return
		}

		cmd.Printf("Wallpaper %s removed from blacklist.\n", id)
	},
}

var blacklistListCmd = &cobra.Command{
	Use:   "list",
	Short: "List blacklisted wallpapers",
	Run: func(cmd *cobra.Command, args []string) {
		ensureConfig()
		cfg, err := loadConfig()
		if err != nil {
			cmd.Printf("Error loading config: %v\n", err)
			return
		}
		controller := core.NewController(cfg)

		blacklist, err := controller.GetBlacklist()
		if err != nil {
			cmd.Printf("Error getting blacklist: %v\n", err)
			return
		}

		cmd.Println("Blacklisted IDs:")
		for _, id := range blacklist {
			cmd.Printf(" - %s\n", id)
		}
	},
}

func init() {
	rootCmd.AddCommand(blacklistCmd)
	blacklistCmd.AddCommand(blacklistAddCmd)
	blacklistCmd.AddCommand(blacklistRemoveCmd)
	blacklistCmd.AddCommand(blacklistListCmd)
}
