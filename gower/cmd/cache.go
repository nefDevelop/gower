package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
		home, _ := os.UserHomeDir()
		cacheDir := filepath.Join(home, ".gower", "cache")

		// Remove all contents
		err := os.RemoveAll(cacheDir)
		if err != nil {
			fmt.Printf("Error cleaning cache: %v\n", err)
			return
		}
		// Recreate structure
		os.MkdirAll(filepath.Join(cacheDir, "wallpapers"), 0755)
		os.MkdirAll(filepath.Join(cacheDir, "thumbs"), 0755)

		fmt.Println("Cache cleaned successfully.")
	},
}

var cacheSizeCmd = &cobra.Command{
	Use:   "size",
	Short: "Show cache size",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		cacheDir := filepath.Join(home, ".gower", "cache")

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
			fmt.Printf("Error calculating cache size: %v\n", err)
			return
		}

		fmt.Printf("Cache size: %.2f MB\n", float64(size)/1024/1024)
	},
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cacheSizeCmd)
}
