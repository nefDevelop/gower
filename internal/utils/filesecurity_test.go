package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type TestData struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestSecureJSONManager_WriteAndRead(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gower-utils-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manager := NewSecureJSONManager()
	filePath := filepath.Join(tmpDir, "test.json")
	data := TestData{Name: "test", Value: 123}

	// Test Write
	if err := manager.WriteJSON(filePath, data); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("File was not created")
	}

	// Test Read
	var readData TestData
	if err := manager.ReadJSON(filePath, &readData); err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}

	if readData.Name != data.Name || readData.Value != data.Value {
		t.Errorf("Read data mismatch: got %+v, want %+v", readData, data)
	}
}

func TestSecureJSONManager_Backup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gower-utils-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manager := NewSecureJSONManager()
	// Force backup on every write for testing
	manager.BackupInterval = 0

	filePath := filepath.Join(tmpDir, "test.json")
	data1 := TestData{Name: "v1", Value: 1}

	// First write
	if err := manager.WriteJSON(filePath, data1); err != nil {
		t.Fatal(err)
	}

	// Second write should trigger backup of v1
	data2 := TestData{Name: "v2", Value: 2}
	// Sleep briefly to ensure modtime difference if filesystem resolution is low
	time.Sleep(10 * time.Millisecond)
	if err := manager.WriteJSON(filePath, data2); err != nil {
		t.Fatal(err)
	}

	// Check backup exists
	backupPath := filePath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file was not created")
	}

	// Verify backup content is v1
	var backupData TestData
	if err := manager.ReadJSON(backupPath, &backupData); err != nil {
		t.Fatal(err)
	}
	if backupData.Name != "v1" {
		t.Errorf("Backup content mismatch: got %s, want v1", backupData.Name)
	}
}

func TestSecureJSONManager_RestoreFromBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gower-utils-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manager := NewSecureJSONManager()
	filePath := filepath.Join(tmpDir, "test.json")
	backupPath := filePath + ".bak"

	// Create a corrupt main file
	if err := os.WriteFile(filePath, []byte("{invalid-json"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a valid backup
	validData := TestData{Name: "backup", Value: 999}
	jsonData, _ := json.Marshal(validData)
	if err := os.WriteFile(backupPath, jsonData, 0644); err != nil {
		t.Fatal(err)
	}

	// Try to read
	var result TestData
	if err := manager.ReadJSON(filePath, &result); err != nil {
		t.Fatalf("ReadJSON failed to recover: %v", err)
	}

	if result.Name != "backup" {
		t.Errorf("Failed to restore data from backup")
	}

	// Verify main file was overwritten with valid data
	content, _ := os.ReadFile(filePath)
	if !json.Valid(content) {
		t.Errorf("Main file was not repaired")
	}
}

func TestSecureJSONManager_CorruptBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gower-utils-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manager := NewSecureJSONManager()
	filePath := filepath.Join(tmpDir, "test.json")
	backupPath := filePath + ".bak"

	// Create a valid main file
	validData := TestData{Name: "main", Value: 111}
	jsonData, _ := json.Marshal(validData)
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a corrupt backup file
	if err := os.WriteFile(backupPath, []byte("{invalid-json"), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to read. It should succeed by reading the main file and not error out
	// due to the corrupt backup.
	var result TestData
	if err := manager.ReadJSON(filePath, &result); err != nil {
		t.Fatalf("ReadJSON should have succeeded but failed: %v", err)
	}

	if result.Name != "main" {
		t.Errorf("Read incorrect data: expected 'main', got '%s'", result.Name)
	}
}
