package cmd

import (
	"bytes"
	"os"
	"runtime"
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

// setupTestEnv crea un directorio temporal y establece la variable de entorno
// apropiada (HOME o APPDATA) para que os.UserConfigDir() apunte dentro
// del directorio temporal. Esto hace las pruebas herméticas y multiplataforma.
func setupTestEnv(t *testing.T) string {
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", tmpDir)
	} else {
		t.Setenv("HOME", tmpDir)
	}
	// Asegurarse de que XDG_CONFIG_HOME no esté establecido, para que se use el fallback a HOME/.config.
	t.Setenv("XDG_CONFIG_HOME", "")
	return tmpDir
}
