package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SecureJSONManager handles secure reading and writing of JSON files with backups.
type SecureJSONManager struct {
	BackupInterval time.Duration
}

// NewSecureJSONManager creates a new manager with a default backup interval of 8 hours.
func NewSecureJSONManager() *SecureJSONManager {
	return &SecureJSONManager{
		BackupInterval: 8 * time.Hour,
	}
}

// WriteJSON writes data to a JSON file atomically and manages backups.
func (m *SecureJSONManager) WriteJSON(filePath string, data interface{}) error {
	// 1. Marshal new data to ensure it's valid before touching anything
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// 2. Manage Backup of existing file
	if err := m.manageBackup(filePath); err != nil {
		// Log warning but proceed with write, as saving current work is priority
		fmt.Printf("Warning: Failed to create backup for %s: %v\n", filePath, err)
	}

	// 3. Atomic Write
	return m.atomicWrite(filePath, jsonData)
}

// ReadJSON reads data from a JSON file, falling back to backup if necessary.
func (m *SecureJSONManager) ReadJSON(filePath string, v interface{}) error {
	// Try reading main file
	err := m.readAndUnmarshal(filePath, v)
	if err == nil {
		return nil
	}

	// If main file fails, try backup
	backupPath := filePath + ".bak"
	fmt.Printf("Warning: Failed to read %s (%v). Attempting backup %s...\n", filePath, err, backupPath)

	if err := m.readAndUnmarshal(backupPath, v); err != nil {
		return fmt.Errorf("failed to read file and backup: %w", err)
	}

	// Restore main file from backup since main is corrupt
	fmt.Printf("Restoring %s from backup...\n", filePath)
	if data, err := os.ReadFile(backupPath); err == nil {
		_ = m.atomicWrite(filePath, data)
	}

	return nil
}

func (m *SecureJSONManager) readAndUnmarshal(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (m *SecureJSONManager) atomicWrite(filePath string, data []byte) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create temp file in the same directory to ensure atomic rename works
	tmpFile, err := os.CreateTemp(dir, "gower-tmp-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name()) // Clean up if something goes wrong before rename

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	// Rename is atomic on POSIX
	return os.Rename(tmpFile.Name(), filePath)
}

func (m *SecureJSONManager) manageBackup(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // Nothing to backup yet
	}

	backupPath := filePath + ".bak"

	// Check if backup exists and is recent
	info, err := os.Stat(backupPath)
	if err == nil {
		if time.Since(info.ModTime()) < m.BackupInterval {
			return nil // Backup is recent enough
		}
	}

	// Read existing file to verify integrity before backing up
	existingData, err := os.ReadFile(filePath)
	if err != nil {
		return nil // Can't read existing, maybe corrupt, don't backup corrupt data
	}

	if !json.Valid(existingData) {
		return fmt.Errorf("existing file is corrupt, skipping backup")
	}

	// Write to backup
	return os.WriteFile(backupPath, existingData, 0644)
}
