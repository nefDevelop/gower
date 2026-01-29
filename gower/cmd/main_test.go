package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// executeCommand executes a Cobra command and captures its output.
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// setupTestHome creates a temporary directory and sets the HOME environment variable.
func setupTestHome(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "gower-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", tmpDir)
	// For Windows compatibility
	t.Setenv("USERPROFILE", tmpDir)
	return tmpDir
}
