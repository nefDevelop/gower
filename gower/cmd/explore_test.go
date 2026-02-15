package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gower/internal/providers"
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
	var gowerDir string
	if runtime.GOOS == "windows" {
		gowerDir = filepath.Join(dir, "gower")
	} else {
		gowerDir = filepath.Join(dir, ".config", "gower")
	}
	// La función que llama a esta ya ha creado el directorio base,
	// así que solo necesitamos crear el directorio de la app dentro de él.
	os.MkdirAll(gowerDir, 0755)

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
	_ = setupTestEnv(t)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"data":[{"id":"test_id","path":"http://example.com/img.jpg","resolution":"1920x1080","thumbs":{"large":"http://example.com/thumb.jpg"}}]}`)
	}))
	defer server.Close()

	origURL := providers.WallhavenBaseURL
	providers.WallhavenBaseURL = server.URL
	defer func() { providers.WallhavenBaseURL = origURL }()

	testRootCmd, _, _ := newTestRootCmd()

	// Usamos la configuración por defecto que incluye wallhaven
	executeCommand(testRootCmd, "config", "init")

	output, err := executeCommand(testRootCmd, "explore", "--provider", "wallhaven", "test")
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
	tmpDir := setupTestEnv(t)

	// 1. Crear un mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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

	// Create parser mapping file as GenericProvider relies on it
	var appConfigDir string
	if runtime.GOOS == "windows" {
		appConfigDir = filepath.Join(tmpDir, "gower")
	} else {
		appConfigDir = filepath.Join(tmpDir, ".config", "gower")
	}
	parserDir := filepath.Join(appConfigDir, "data", "parser")
	if err := os.MkdirAll(parserDir, 0755); err != nil {
		t.Fatal(err)
	}
	mapping := cfg.GenericProviders[0].ResponseMapping
	data, _ := json.Marshal(mapping)
	if err := os.WriteFile(filepath.Join(parserDir, "generic_test.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	testRootCmd, _, _ := newTestRootCmd()
	// 3. Ejecutar el comando
	output, err := executeCommand(testRootCmd, "explore", "--provider", "generic_test", "searchterm")
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
	tmpDir := setupTestEnv(t)

	// 1. Mock server para el genérico
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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

	testRootCmd, _, _ := newTestRootCmd()
	// 3. Ejecutar con --all
	output, err := executeCommand(testRootCmd, "explore", "--all", "anything")
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
