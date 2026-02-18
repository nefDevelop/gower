package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gower/internal/core"
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
		baseDir, _ := core.GetAppDir()

		allGood := true
		for _, f := range files {
			path := filepath.Join(baseDir, f)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				cmd.Printf("MISSING: %s\n", f)
				allGood = false
				continue
			}

			data, err := os.ReadFile(path)
			if err != nil {
				cmd.Printf("ERROR reading %s: %v\n", f, err)
				allGood = false
				continue
			}

			if !json.Valid(data) {
				cmd.Printf("CORRUPT: %s is not valid JSON\n", f)
				allGood = false
			} else {
				cmd.Printf("OK: %s\n", f)
			}
		}

		if allGood {
			cmd.Println("All storage files are valid.")
		} else {
			cmd.Println("Some files have issues. Try 'gower storage repair'.")
		}
	},
}

var storageRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Attempt to repair corrupt files from backups",
	Run: func(cmd *cobra.Command, args []string) {
		files := []string{"config.json", "data/feed.json", "data/favorites.json", "data/blacklist.json"}
		baseDir, _ := core.GetAppDir()

		for _, f := range files {
			path := filepath.Join(baseDir, f)
			backupPath := path + ".bak"

			// Check if main file is corrupt or missing
			needsRepair := false
			if _, err := os.Stat(path); os.IsNotExist(err) {
				needsRepair = true
				cmd.Printf("%s is missing.\n", f)
			} else {
				data, err := os.ReadFile(path)
				if err != nil || !json.Valid(data) {
					needsRepair = true
					cmd.Printf("%s is corrupt.\n", f)
				}
			}

			if needsRepair {
				// Try to restore from backup
				if _, err := os.Stat(backupPath); err == nil {
					cmd.Printf("Restoring %s from backup...\n", f)
					data, err := os.ReadFile(backupPath)
					if err == nil && json.Valid(data) {
						if err := os.WriteFile(path, data, 0644); err == nil {
							cmd.Printf("Successfully repaired %s\n", f)
							utils.Log.Info("Storage repair: Successfully repaired %s from backup", f)
						} else {
							cmd.Printf("Failed to write repaired file %s: %v\n", f, err)
							utils.Log.Error("Storage repair: Failed to write repaired file %s: %v", f, err)
						}
					} else {
						cmd.Printf("Backup for %s is also corrupt or unreadable.\n", f)
					}
				} else {
					cmd.Printf("No backup found for %s. Cannot repair.\n", f)
				}
			} else {
				cmd.Printf("%s is healthy.\n", f)
			}
		}
	},
}

func init() {
	systemCmd.AddCommand(storageCmd)
	storageCmd.AddCommand(storageVerifyCmd)
	storageCmd.AddCommand(storageRepairCmd)
}
