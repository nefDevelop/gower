// cmd/explore.go

package cmd

import (
	"fmt"
	// "os"
	"github.com/spf13/cobra"
)

var (
	exploreQuery      string
	exploreProvider   string
	explorePage       int
	explorePerPage    int
	exploreMinWidth   int
	exploreMinHeight  int
	exploreAspect     string
	explorePurity     string
	exploreCategories string
	exploreSubreddit  string
	exploreAPOD       bool
)

var exploreCmd = &cobra.Command{
	Use:   "explore [query]",
	Short: "Discover new wallpapers",
	Args:  cobra.MinimumNArgs(0),
	Run:   runExplore,
}

func runExplore(cmd *cobra.Command, args []string) {
	fmt.Println("Ejecutando el comando 'explore'...")
}

func init() {
	rootCmd.AddCommand(exploreCmd)
	exploreCmd.Flags().StringVarP(&exploreProvider, "provider", "p", "all",
		"provider [wallhaven|reddit|nasa|all]")
	exploreCmd.Flags().IntVar(&explorePage, "page", 1,
		"page number")
	exploreCmd.Flags().IntVar(&explorePerPage, "per-page", 30,
		"items per page")
	exploreCmd.Flags().IntVar(&exploreMinWidth, "min-width", 1920,
		"minimum width")
	exploreCmd.Flags().IntVar(&exploreMinHeight, "min-height", 1080,
		"minimum height")
	exploreCmd.Flags().StringVar(&exploreAspect, "aspect-ratio", "",
		"aspect ratio (e.g., 16:9)")
	exploreCmd.Flags().StringVar(&explorePurity, "purity", "sfw",
		"purity [sfw|sketchy|nsfw]")
	exploreCmd.Flags().StringVar(&exploreCategories, "categories", "111",
		"categories [1=enabled,0=disabled] (general|anime|people)")
	exploreCmd.Flags().StringVar(&exploreSubreddit, "subreddit", "wallpapers",
		"subreddit for Reddit provider")
	exploreCmd.Flags().BoolVar(&exploreAPOD, "apod", false,
		"get Astronomy Picture of the Day (NASA)")
}
