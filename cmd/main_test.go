package cmd

import (
	"bytes"
	"path/filepath"
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
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// For Windows compatibility
	t.Setenv("USERPROFILE", tmpDir)
	t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	return tmpDir
}

// setupTestEnv crea un directorio temporal y establece las variables de entorno
// necesarias para que os.UserConfigDir() apunte dentro del directorio temporal.
func setupTestEnv(t *testing.T) string {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("APPDATA", filepath.Join(tmpDir, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))
	t.Setenv("USERPROFILE", tmpDir)
	return tmpDir
}
