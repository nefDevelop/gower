package cmd

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupStorageTest(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "gower-storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	os.Setenv("HOME", tmpDir)
	baseDir := filepath.Join(tmpDir, ".gower")
	os.MkdirAll(filepath.Join(baseDir, "data"), 0755)

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

func TestStorageVerifyCmd_AllGood(t *testing.T) {
	tmpDir, cleanup := setupStorageTest(t)
	defer cleanup()

	baseDir := filepath.Join(tmpDir, ".gower")
	files := []string{"config.json", "data/feed.json", "data/favorites.json", "data/blacklist.json"}
	for _, f := range files {
		path := filepath.Join(baseDir, f)
		if err := ioutil.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
	}

	output, err := executeCommand(rootCmd, "storage", "verify")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.Contains(output, "All storage files are valid.") {
		t.Errorf("Expected 'All storage files are valid.', got '%s'", output)
	}
}

func TestStorageVerifyCmd_MissingFile(t *testing.T) {
	_, cleanup := setupStorageTest(t)
	defer cleanup()

	output, err := executeCommand(rootCmd, "storage", "verify")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.Contains(output, "MISSING: config.json") {
		t.Errorf("Expected 'MISSING: config.json', got '%s'", output)
	}
}

func TestStorageVerifyCmd_CorruptFile(t *testing.T) {
	tmpDir, cleanup := setupStorageTest(t)
	defer cleanup()

	baseDir := filepath.Join(tmpDir, ".gower")
	path := filepath.Join(baseDir, "config.json")
	if err := ioutil.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	output, err := executeCommand(rootCmd, "storage", "verify")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.Contains(output, "CORRUPT: config.json is not valid JSON") {
		t.Errorf("Expected 'CORRUPT: config.json is not valid JSON', got '%s'", output)
	}
}

func TestStorageRepairCmd_FromBackup(t *testing.T) {
	tmpDir, cleanup := setupStorageTest(t)
	defer cleanup()

	baseDir := filepath.Join(tmpDir, ".gower")
	filePath := filepath.Join(baseDir, "config.json")
	backupPath := filePath + ".bak"

	// Create a corrupt main file and a valid backup
	if err := ioutil.WriteFile(filePath, []byte("{"), 0644); err != nil {
		t.Fatalf("Failed to write corrupt file: %v", err)
	}
	if err := ioutil.WriteFile(backupPath, []byte(`{"key":"value"}`), 0644); err != nil {
		t.Fatalf("Failed to write backup file: %v", err)
	}

	output, err := executeCommand(rootCmd, "storage", "repair")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.Contains(output, "Successfully repaired config.json") {
		t.Errorf("Expected 'Successfully repaired config.json', got '%s'", output)
	}

	// Verify the file was repaired
	repairedData, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read repaired file: %v", err)
	}
	if string(repairedData) != `{"key":"value"}` {
		t.Errorf("File content was not restored from backup")
	}
}

func TestStorageRepairCmd_NoBackup(t *testing.T) {
	tmpDir, cleanup := setupStorageTest(t)
	defer cleanup()

	baseDir := filepath.Join(tmpDir, ".gower")
	filePath := filepath.Join(baseDir, "config.json")

	// Create a corrupt main file and no backup
	if err := ioutil.WriteFile(filePath, []byte("{"), 0644); err != nil {
		t.Fatalf("Failed to write corrupt file: %v", err)
	}

	output, err := executeCommand(rootCmd, "storage", "repair")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.Contains(output, "No backup found for config.json. Cannot repair.") {
		t.Errorf("Expected 'No backup found for config.json. Cannot repair.', got '%s'", output)
	}
}
