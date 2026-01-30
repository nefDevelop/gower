package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDaemonStatusNotRunning(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	output, err := executeCommand(rootCmd, "daemon", "status")
	if err != nil {
		t.Fatalf("Error executing daemon status: %v", err)
	}
	if !strings.Contains(output, "Daemon is stopped") {
		t.Errorf("Expected 'Daemon is stopped', got: %s", output)
	}
}

func TestDaemonStatusRunning(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	// Start a dummy process to simulate the daemon
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start dummy process: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Write PID file
	pidFile := filepath.Join(tmpDir, ".gower", "gower.pid")
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := ioutil.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	output, err := executeCommand(rootCmd, "daemon", "status")
	if err != nil {
		t.Fatalf("Error executing daemon status: %v", err)
	}
	if !strings.Contains(output, "Daemon is running") {
		t.Errorf("Expected 'Daemon is running', got: %s", output)
	}
}

func TestDaemonStop(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	// Start a dummy process
	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start dummy process: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	pidFile := filepath.Join(tmpDir, ".gower", "gower.pid")
	os.MkdirAll(filepath.Dir(pidFile), 0755)
	ioutil.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)

	// Stop
	output, err := executeCommand(rootCmd, "daemon", "stop")
	if err != nil {
		t.Fatalf("Error executing daemon stop: %v", err)
	}
	if !strings.Contains(output, "Stop signal sent") {
		t.Errorf("Expected 'Stop signal sent', got: %s", output)
	}

	// Wait a bit to ensure signal is processed (though we can't easily verify sleep received it without exit code check)
	time.Sleep(100 * time.Millisecond)

	// Test Force Stop (removes PID file)
	executeCommand(rootCmd, "daemon", "stop", "--force")
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Errorf("PID file should be removed with --force")
	}
}
