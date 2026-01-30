package cmd

import (
	"gower/internal/core"

	"github.com/spf13/cobra"
)

var blacklistCmd = &cobra.Command{
	Use:   "blacklist <id>",
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

func init() {
	rootCmd.AddCommand(blacklistCmd)
}
