package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gower/pkg/models"
)

// resetExploreFlags reinicia las variables globales de los flags de explore
// para evitar contaminación entre tests.
func resetExploreFlags() {
	exploreProvider = ""
	exploreAll = false
	exploreMinWidth = 0
	exploreMinHeight = 0
	exploreAspectRatio = ""
	exploreColor = ""
	explorePage = 1
	exploreForceUpdate = false
}

// createTestConfig crea un archivo config.json personalizado para un test.
func createTestConfig(t *testing.T, dir string, config *models.Config) {
	gowerDir := filepath.Join(dir, ".gower")
	if err := os.MkdirAll(gowerDir, 0755); err != nil {
		t.Fatalf("Failed to create .gower dir: %v", err)
	}
	configPath := filepath.Join(gowerDir, "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
}

func TestExploreNativeProvider(t *testing.T) {
	resetExploreFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	// Usamos la configuración por defecto que incluye wallhaven
	executeCommand(rootCmd, "config", "init")

	output, err := executeCommand(rootCmd, "explore", "--provider", "wallhaven", "test")
	if err != nil {
		t.Fatalf("Error executing explore: %v", err)
	}

	if !strings.Contains(output, "Querying provider: wallhaven") {
		t.Errorf("Expected output to contain 'Querying provider: wallhaven', got: %s", output)
	}
	if !strings.Contains(output, "ID: wh_") {
		t.Errorf("Expected output to contain wallhaven data (ID: wh_...), got: %s", output)
	}
}

func TestExploreGenericProvider(t *testing.T) {
	resetExploreFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	// 1. Crear un mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{
			"images": [
				{
					"id": "generic-1",
					"image_url": "http://example.com/img1.jpg",
					"res": "1920x1080"
				}
			]
		}`)
	}))
	defer server.Close()

	// 2. Crear una configuración personalizada
	cfg := getDefaultConfig()
	cfg.GenericProviders = []models.GenericProviderConfig{
		{
			Name:    "generic_test",
			Enabled: true,
			APIURL:  server.URL, // Apuntar al mock server
			ResponseMapping: models.ResponseMapping{
				ResultsPath:   "images",
				IDPath:        "id",
				URLPath:       "image_url",
				DimensionPath: "res",
			},
		},
	}
	createTestConfig(t, tmpDir, &cfg)

	// 3. Ejecutar el comando
	output, err := executeCommand(rootCmd, "explore", "--provider", "generic_test", "searchterm")
	if err != nil {
		t.Fatalf("Error executing explore with generic provider: %v", err)
	}

	// 4. Verificar la salida
	if !strings.Contains(output, "Querying provider: generic_test") {
		t.Errorf("Expected output to contain 'Querying provider: generic_test', got: %s", output)
	}
	if !strings.Contains(output, "ID: generic_test-generic-1") {
		t.Errorf("Expected output to contain data from generic provider, got: %s", output)
	}
	if !strings.Contains(output, "URL: http://example.com/img1.jpg") {
		t.Errorf("Expected output to contain URL from generic provider, got: %s", output)
	}
}

func TestExploreAllProviders(t *testing.T) {
	resetExploreFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	// 1. Mock server para el genérico
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"images":[]}`) // No necesitamos resultados, solo que se llame
	}))
	defer server.Close()

	// 2. Config con proveedor nativo y genérico
	cfg := getDefaultConfig() // Esto ya incluye wallhaven
	cfg.GenericProviders = []models.GenericProviderConfig{
		{
			Name:    "generic_test",
			Enabled: true,
			APIURL:  server.URL,
			ResponseMapping: models.ResponseMapping{
				ResultsPath: "images",
				IDPath:      "id",
				URLPath:     "url",
			},
		},
	}
	createTestConfig(t, tmpDir, &cfg)

	// 3. Ejecutar con --all
	output, err := executeCommand(rootCmd, "explore", "--all", "anything")
	if err != nil {
		t.Fatalf("Error executing explore with --all: %v", err)
	}

	// 4. Verificar que ambos son consultados
	if !strings.Contains(output, "Querying provider: wallhaven") {
		t.Errorf("Expected to see wallhaven being consulted, got: %s", output)
	}
	if !strings.Contains(output, "Querying provider: generic_test") {
		t.Errorf("Expected to see generic_test being consulted, got: %s", output)
	}
}
