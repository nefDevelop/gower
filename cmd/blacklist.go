package cmd

import (
	"fmt"
	"gower/internal/core"
	"os"

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
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
		controller := core.NewController(cfg)

		if err := controller.AddToBlacklist(id); err != nil {
			fmt.Printf("Error adding to blacklist: %v\n", err)
			os.Exit(1)
		}

		// Remove from feed if present
		if err := controller.RemoveFromFeed(id); err != nil {
			fmt.Printf("Warning: Failed to remove from feed: %v\n", err)
		}

		fmt.Printf("Wallpaper %s added to blacklist.\n", id)
	},
}

func init() {
	rootCmd.AddCommand(blacklistCmd)
}
