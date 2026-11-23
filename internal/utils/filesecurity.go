package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// fileLocks serializa las secuencias load-modify-save sobre un mismo fichero
// aunque intervengan instancias distintas de SecureJSONManager.
var fileLocks sync.Map // map[string]*sync.Mutex

func fileLockFor(path string) *sync.Mutex {
	key, err := filepath.Abs(path)
	if err != nil {
		key = path
	}
	actual, _ := fileLocks.LoadOrStore(key, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

// WithFileLock ejecuta fn manteniendo un lock exclusivo asociado a path.
// Debe envolver lecturas y escrituras relacionadas, no operaciones sueltas.
func WithFileLock(path string, fn func() error) error {
	mu := fileLockFor(path)
	mu.Lock()
	defer mu.Unlock()
	return fn()
}

// SecureJSONManager handles secure reading and writing of JSON files with backups.
type SecureJSONManager struct {
	BackupInterval time.Duration
	Log            *Logger
}

func (m *SecureJSONManager) log() *Logger {
	if m.Log != nil {
		return m.Log
	}
	return Log
}

// NewSecureJSONManager creates a new manager with a default backup interval of 8 hours.
func NewSecureJSONManager() *SecureJSONManager {
	return &SecureJSONManager{
		BackupInterval: 8 * time.Hour,
	}
}

// WriteJSON writes data to a JSON file atomically and manages backups.
func (m *SecureJSONManager) WriteJSON(filePath string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := m.manageBackup(filePath); err != nil {
		m.log().Error("Failed to create backup for %s: %v", filePath, err)
	}

	return m.atomicWrite(filePath, jsonData)
}

// ReadJSON reads data from a JSON file, falling back to backup if necessary.
func (m *SecureJSONManager) ReadJSON(filePath string, v interface{}) error {
	err := m.readAndUnmarshal(filePath, v)
	if err == nil {
		return nil
	}

	backupPath := filePath + ".bak"
	m.log().Error("Failed to read %s (%v). Attempting backup %s...", filePath, err, backupPath)

	if err := m.readAndUnmarshal(backupPath, v); err != nil {
		return fmt.Errorf("failed to read file and backup: %w", err)
	}

	m.log().Info("Restoring %s from backup...", filePath)
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

	// El fichero temporal se crea con 0600: los JSON incluyen claves de API.
	tmpFile, err := os.CreateTemp(dir, "gower-tmp-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpFile.Name(), filePath); err != nil {
		return err
	}

	// fsync del directorio para que el rename sobreviva a un corte de energía.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

func (m *SecureJSONManager) manageBackup(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	backupPath := filePath + ".bak"

	if info, err := os.Stat(backupPath); err == nil {
		if time.Since(info.ModTime()) < m.BackupInterval {
			return nil
		}
	}

	existingData, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	if !json.Valid(existingData) {
		return fmt.Errorf("existing file is corrupt, skipping backup")
	}

	// El backup conserva secretos: debe tener los mismos permisos que el original.
	return os.WriteFile(backupPath, existingData, 0600)
}
