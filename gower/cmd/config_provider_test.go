package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gower/pkg/models"
)

func TestConfigProviderList(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	output, err := executeCommand(rootCmd, "config", "provider", "list")
	if err != nil {
		t.Fatalf("Error executing provider list: %v", err)
	}

	if !strings.Contains(output, "Native Providers:") {
		t.Errorf("Expected 'Native Providers:', got: %s", output)
	}
	if !strings.Contains(output, "[Wallhaven]") {
		t.Errorf("Expected '[Wallhaven]', got: %s", output)
	}
}

func TestConfigProviderGenericAddAndRemove(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	providerName := "test_provider"
	providerURL := "https://api.test.com/search?q={query}"

	// Test Add
	output, err := executeCommand(rootCmd, "config", "provider", "add", providerName, providerURL,
		"--key", "secret_key",
		"--results-path", "data.items",
		"--id-path", "uuid",
		"--url-path", "link",
	)
	if err != nil {
		t.Fatalf("Error executing provider add: %v", err)
	}
	if !strings.Contains(output, "added successfully") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify config.json
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("Error loading config: %v", err)
	}

	found := false
	for _, p := range cfg.GenericProviders {
		if p.Name == providerName {
			found = true
			if p.APIURL != providerURL {
				t.Errorf("Expected URL %s, got %s", providerURL, p.APIURL)
			}
			if p.APIKey != "secret_key" {
				t.Errorf("Expected API Key 'secret_key', got %s", p.APIKey)
			}
			break
		}
	}
	if !found {
		t.Errorf("Provider not found in config")
	}

	// Verify parser file
	parserPath := filepath.Join(tmpDir, ".config", "gower", "data", "parser", providerName+".json")
	if _, err := os.Stat(parserPath); os.IsNotExist(err) {
		t.Errorf("Parser file not created at %s", parserPath)
	}

	data, err := ioutil.ReadFile(parserPath)
	if err != nil {
		t.Fatalf("Error reading parser file: %v", err)
	}

	var mapping models.ResponseMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		t.Fatalf("Error unmarshalling parser file: %v", err)
	}
	if mapping.ResultsPath != "data.items" {
		t.Errorf("Expected ResultsPath 'data.items', got %s", mapping.ResultsPath)
	}

	// Test Remove
	output, err = executeCommand(rootCmd, "config", "provider", "remove", providerName)
	if err != nil {
		t.Fatalf("Error executing provider remove: %v", err)
	}
	if !strings.Contains(output, "removed") {
		t.Errorf("Expected removed message, got: %s", output)
	}

	// Verify removal from config
	cfg, _ = loadConfig()
	for _, p := range cfg.GenericProviders {
		if p.Name == providerName {
			t.Errorf("Provider still exists in config after removal")
		}
	}

	// Verify removal of parser file
	if _, err := os.Stat(parserPath); !os.IsNotExist(err) {
		t.Errorf("Parser file still exists after removal")
	}
}

func TestConfigProviderReddit(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Test Add
	sub := "cyberpunk"
	output, err := executeCommand(rootCmd, "config", "provider", "reddit", "add", sub)
	if err != nil {
		t.Fatalf("Error executing reddit add: %v", err)
	}
	if !strings.Contains(output, "Added 'cyberpunk'") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Test Remove
	output, err = executeCommand(rootCmd, "config", "provider", "reddit", "remove", sub)
	if err != nil {
		t.Fatalf("Error executing reddit remove: %v", err)
	}
	if !strings.Contains(output, "Removed 'cyberpunk'") {
		t.Errorf("Expected removed message, got: %s", output)
	}
}
