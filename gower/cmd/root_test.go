package cmd

import (
	"strings"
	"testing"
)

func TestGlobalFlags(t *testing.T) {
	// Test --version
	output, err := executeCommand(rootCmd, "--version")
	if err != nil {
		t.Fatalf("Error executing version: %v", err)
	}
	if !strings.Contains(output, "gower version 0.1.0") {
		t.Errorf("Expected version output, got: %s", output)
	}

	// Test help output for global flags
	output, err = executeCommand(rootCmd, "--help")
	if err != nil {
		t.Fatalf("Error executing help: %v", err)
	}

	expectedFlags := []string{
		"--verbose", "-v",
		"--debug",
		"--quiet", "-q",
		"--json",
		"--table",
		"--no-color",
		"--config",
		"--dry-run",
	}

	for _, flag := range expectedFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("Help output missing flag: %s", flag)
		}
	}
}
