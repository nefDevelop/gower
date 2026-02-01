package utils

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogger_Levels(t *testing.T) {
	t.Run("logs info, error, and debug when debug is true", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(&buf, true)

		logger.Info("info message")
		logger.Error("error message")
		logger.Debug("debug message")

		output := buf.String()
		assert.Contains(t, output, "[INFO] info message")
		assert.Contains(t, output, "[ERROR] error message")
		assert.Contains(t, output, "[DEBUG] debug message")
	})

	t.Run("logs info and error but not debug when debug is false", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(&buf, false)

		logger.Info("info message")
		logger.Error("error message")
		logger.Debug("should not appear")

		output := buf.String()
		assert.Contains(t, output, "[INFO] info message")
		assert.Contains(t, output, "[ERROR] error message")
		assert.NotContains(t, output, "[DEBUG] should not appear")
	})
}

func TestInitLogger_FileCreation(t *testing.T) {
	// This test still checks file creation, but it's less critical now.
	// We keep it to ensure the production code path works.
	tmpDir, err := os.MkdirTemp("", "gower-logger-init-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Temporarily override user home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Reset global logger after test
	defer func() { Log = nil }()

	err = InitLogger(true)
	assert.NoError(t, err)

	logDir := filepath.Join(tmpDir, ".gower", "logs")
	filename := "gower-" + time.Now().Format("2006-01-02") + ".log"
	logPath := filepath.Join(logDir, filename)

	_, err = os.Stat(logPath)
	assert.NoError(t, err, "Log file should be created")
}
