package cmd

import "github.com/spf13/cobra"

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "System maintenance and utilities",
}

func init() {
	rootCmd.AddCommand(systemCmd)
}
