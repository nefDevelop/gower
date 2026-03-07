package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStorageVerifyCmd_AllGood(t *testing.T) {
	tmpDir := setupTestEnv(t)

	baseDir := filepath.Join(tmpDir, ".config", "gower")
	files := []string{"config.json", "data/feed.json", "data/favorites.json", "data/blacklist.json"}
	for _, f := range files {
		path := filepath.Join(baseDir, f)
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", path, err)
		}
	}

	output, err := executeCommand(rootCmd, "system", "storage", "verify")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.Contains(output, "All storage files are valid.") {
		t.Errorf("Expected 'All storage files are valid.', got '%s'", output)
	}
}

func TestStorageVerifyCmd_MissingFile(t *testing.T) {
	setupTestEnv(t)

	output, err := executeCommand(rootCmd, "system", "storage", "verify")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.Contains(output, "MISSING: config.json") {
		t.Errorf("Expected 'MISSING: config.json', got '%s'", output)
	}
}

func TestStorageVerifyCmd_CorruptFile(t *testing.T) {
	tmpDir := setupTestEnv(t)

	baseDir := filepath.Join(tmpDir, ".config", "gower")
	path := filepath.Join(baseDir, "config.json")
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	output, err := executeCommand(rootCmd, "system", "storage", "verify")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.Contains(output, "CORRUPT: config.json is not valid JSON") {
		t.Errorf("Expected 'CORRUPT: config.json is not valid JSON', got '%s'", output)
	}
}

func TestStorageRepairCmd_FromBackup(t *testing.T) {
	tmpDir := setupTestEnv(t)

	baseDir := filepath.Join(tmpDir, ".config", "gower")
	filePath := filepath.Join(baseDir, "config.json")
	backupPath := filePath + ".bak"

	_ = os.MkdirAll(baseDir, 0755)

	// Create a corrupt main file and a valid backup
	if err := os.WriteFile(filePath, []byte("{"), 0644); err != nil {
		t.Fatalf("Failed to write corrupt file: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte(`{"key":"value"}`), 0644); err != nil {
		t.Fatalf("Failed to write backup file: %v", err)
	}

	output, err := executeCommand(rootCmd, "system", "storage", "repair")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.Contains(output, "Successfully repaired config.json") {
		t.Errorf("Expected 'Successfully repaired config.json', got '%s'", output)
	}

	// Verify the file was repaired
	repairedData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read repaired file: %v", err)
	}
	if string(repairedData) != `{"key":"value"}` {
		t.Errorf("File content was not restored from backup")
	}
}

func TestStorageRepairCmd_NoBackup(t *testing.T) {
	tmpDir := setupTestEnv(t)

	baseDir := filepath.Join(tmpDir, ".config", "gower")
	filePath := filepath.Join(baseDir, "config.json")

	_ = os.MkdirAll(baseDir, 0755)

	// Create a corrupt main file and no backup
	if err := os.WriteFile(filePath, []byte("{"), 0644); err != nil {
		t.Fatalf("Failed to write corrupt file: %v", err)
	}

	output, err := executeCommand(rootCmd, "system", "storage", "repair")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.Contains(output, "No backup found for config.json. Cannot repair.") {
		t.Errorf("Expected 'No backup found for config.json. Cannot repair.', got '%s'", output)
	}
}
