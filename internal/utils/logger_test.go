package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupLoggerTest(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "gower-logger-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	os.Setenv("HOME", tmpDir)

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
		// Reset the global logger
		Log = nil
	}
}

func TestInitLogger(t *testing.T) {
	tmpDir, cleanup := setupLoggerTest(t)
	defer cleanup()

	err := InitLogger(true)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}

	logDir := filepath.Join(tmpDir, ".gower", "logs")
	filename := "gower-" + time.Now().Format("2006-01-02") + ".log"
	logPath := filepath.Join(logDir, filename)

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created at %s", logPath)
	}
}

func TestLoggerMethods(t *testing.T) {
	tmpDir, cleanup := setupLoggerTest(t)
	defer cleanup()

	// Test with debug enabled
	err := InitLogger(true)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}

	Log.Info("Info message")
	Log.Error("Error message")
	Log.Debug("Debug message")

	logDir := filepath.Join(tmpDir, ".gower", "logs")
	filename := "gower-" + time.Now().Format("2006-01-02") + ".log"
	logPath := filepath.Join(logDir, filename)

	content, err := ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "[INFO] Info message") {
		t.Errorf("Log should contain info message")
	}
	if !strings.Contains(logContent, "[ERROR] Error message") {
		t.Errorf("Log should contain error message")
	}
	if !strings.Contains(logContent, "[DEBUG] Debug message") {
		t.Errorf("Log should contain debug message when debug is true")
	}

	// Clean up the file for the next test
	if err := os.Remove(logPath); err != nil {
		t.Fatalf("Failed to remove log file: %v", err)
	}

	// Test with debug disabled
	err = InitLogger(false)
	if err != nil {
		t.Fatalf("InitLogger failed: %v", err)
	}

	Log.Debug("This should not be logged")

	content, err = ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if strings.Contains(string(content), "This should not be logged") {
		t.Errorf("Log should not contain debug message when debug is false")
	}
}
