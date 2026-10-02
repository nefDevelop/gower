package cmd

import (
	"strings"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show detailed feed statistics",
	Long: `Display statistics about your wallpaper feed including
breakdowns by provider, theme, color, and resolution.`,
	RunE: runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
	if err := ensureConfig(); err != nil {
		return err
	}
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	controller := core.NewController(cfg)
	feed, err := controller.GetFeedWallpapers()
	if err != nil {
		return err
	}

	stats := computeStats(feed)

	if config.JSONOutput {
		displayJSON(cmd, stats)
		return nil
	}

	cmd.Println("=== Wallpaper Feed Statistics ===")
	cmd.Printf("Total wallpapers: %d\n\n", stats.Total)

	cmd.Println("By Provider:")
	for _, p := range stats.ByProvider {
		bar := strings.Repeat("█", p.Percent/5)
		cmd.Printf("  %-12s %4d  %s %d%%\n", p.Name, p.Count, bar, p.Percent)
	}

	cmd.Println("\nBy Theme:")
	for _, t := range stats.ByTheme {
		cmd.Printf("  %-6s %d\n", t.Name, t.Count)
	}

	cmd.Println("\nBy Resolution:")
	for _, r := range stats.ByResolution {
		cmd.Printf("  %-12s %d\n", r.Name, r.Count)
	}

	cmd.Println("\nBy Aspect Ratio:")
	for _, r := range stats.ByRatio {
		cmd.Printf("  %-10s %d\n", r.Name, r.Count)
	}

	return nil
}

type FeedStats struct {
	Total        int         `json:"total"`
	ByProvider   []StatEntry `json:"by_provider"`
	ByTheme      []StatEntry `json:"by_theme"`
	ByResolution []StatEntry `json:"by_resolution"`
	ByRatio      []StatEntry `json:"by_aspect_ratio"`
}

type StatEntry struct {
	Name    string `json:"name"`
	Count   int    `json:"count"`
	Percent int    `json:"percent,omitempty"`
}

func computeStats(feed []models.Wallpaper) FeedStats {
	stats := FeedStats{Total: len(feed)}

	providers := map[string]int{}
	themes := map[string]int{}
	resolutions := map[string]int{}
	ratios := map[string]int{}

	for _, wp := range feed {
		providers[wp.Source]++
		if wp.Theme != "" {
			themes[wp.Theme]++
		}
		if wp.Dimension != "" {
			resolutions[wp.Dimension]++
		}
		if wp.Ratio != "" {
			ratios[wp.Ratio]++
		}
	}

	stats.ByProvider = toSortedEntries(providers, stats.Total)
	stats.ByTheme = toSortedEntries(themes, stats.Total)
	stats.ByResolution = toSortedEntries(resolutions, stats.Total)
	stats.ByRatio = toSortedEntries(ratios, stats.Total)

	return stats
}

func toSortedEntries(m map[string]int, total int) []StatEntry {
	var entries []StatEntry
	for name, count := range m {
		pct := 0
		if total > 0 {
			pct = count * 100 / total
		}
		entries = append(entries, StatEntry{Name: name, Count: count, Percent: pct})
	}

	// Sort by count descending
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Count > entries[i].Count {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	return entries
}
