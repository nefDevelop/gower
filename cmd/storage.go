package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gower/internal/utils"

	"github.com/spf13/cobra"
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Manage application storage (JSON files)",
}

var storageVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify integrity of data files",
	Run: func(cmd *cobra.Command, args []string) {
		files := []string{"config.json", "data/feed.json", "data/favorites.json", "data/blacklist.json"}
		home, _ := os.UserHomeDir()
		baseDir := filepath.Join(home, ".gower")

		allGood := true
		for _, f := range files {
			path := filepath.Join(baseDir, f)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				fmt.Printf("MISSING: %s\n", f)
				allGood = false
				continue
			}

			data, err := ioutil.ReadFile(path)
			if err != nil {
				fmt.Printf("ERROR reading %s: %v\n", f, err)
				allGood = false
				continue
			}

			if !json.Valid(data) {
				fmt.Printf("CORRUPT: %s is not valid JSON\n", f)
				allGood = false
			} else {
				fmt.Printf("OK: %s\n", f)
			}
		}

		if allGood {
			fmt.Println("All storage files are valid.")
		} else {
			fmt.Println("Some files have issues. Try 'gower storage repair'.")
		}
	},
}

var storageRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Attempt to repair corrupt files from backups",
	Run: func(cmd *cobra.Command, args []string) {
		files := []string{"config.json", "data/feed.json", "data/favorites.json", "data/blacklist.json"}
		home, _ := os.UserHomeDir()
		baseDir := filepath.Join(home, ".gower")

		for _, f := range files {
			path := filepath.Join(baseDir, f)
			backupPath := path + ".bak"

			// Check if main file is corrupt or missing
			needsRepair := false
			if _, err := os.Stat(path); os.IsNotExist(err) {
				needsRepair = true
				fmt.Printf("%s is missing.\n", f)
			} else {
				data, err := ioutil.ReadFile(path)
				if err != nil || !json.Valid(data) {
					needsRepair = true
					fmt.Printf("%s is corrupt.\n", f)
				}
			}

			if needsRepair {
				// Try to restore from backup
				if _, err := os.Stat(backupPath); err == nil {
					fmt.Printf("Restoring %s from backup...\n", f)
					data, err := ioutil.ReadFile(backupPath)
					if err == nil && json.Valid(data) {
						if err := ioutil.WriteFile(path, data, 0644); err == nil {
							fmt.Printf("Successfully repaired %s\n", f)
							utils.Log.Info("Storage repair: Successfully repaired %s from backup", f)
						} else {
							fmt.Printf("Failed to write repaired file %s: %v\n", f, err)
							utils.Log.Error("Storage repair: Failed to write repaired file %s: %v", f, err)
						}
					} else {
						fmt.Printf("Backup for %s is also corrupt or unreadable.\n", f)
					}
				} else {
					fmt.Printf("No backup found for %s. Cannot repair.\n", f)
				}
			} else {
				fmt.Printf("%s is healthy.\n", f)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(storageCmd)
	storageCmd.AddCommand(storageVerifyCmd)
	storageCmd.AddCommand(storageRepairCmd)
}
