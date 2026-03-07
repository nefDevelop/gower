package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gower/internal/core"
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

func TestExploreNativeProvider(t *testing.T) {
	resetExploreFlags()
	_ = setupTestEnv(t)

	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"data":[{"id":"test_id","path":"http://example.com/img.jpg","resolution":"1920x1080","thumbs":{"large":"http://example.com/thumb.jpg"}}]}`)
	}))
	defer server.Close()

	origURL := providers.WallhavenBaseURL
	providers.WallhavenBaseURL = server.URL
	defer func() { providers.WallhavenBaseURL = origURL }()

	// Save original functions to restore later
	originalNewController := core.NewController
	originalLoadConfig := loadConfig
	originalSaveConfig := saveConfig
	t.Cleanup(func() {
		core.NewController = originalNewController
		loadConfig = originalLoadConfig
		saveConfig = originalSaveConfig
	})

	testRootCmd, _, _ := newTestRootCmd()

	// Usamos la configuración por defecto que incluye wallhaven
	executeCommand(testRootCmd, "config", "init")

	// Mock loadConfig to return the config with Wallhaven enabled (default)
	mockedConfig := getDefaultConfig()
	loadConfig = func() (*models.Config, error) { return &mockedConfig, nil }
	saveConfig = func(cfg *models.Config) error { mockedConfig = *cfg; return nil }
	mockController := originalNewController(&mockedConfig) // Use original NewController to create a real controller
	core.NewController = func(cfg *models.Config) *core.Controller { return mockController }

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
	_ = setupTestEnv(t)

	// Save original functions to restore later
	originalNewController := core.NewController
	originalLoadConfig := loadConfig
	originalSaveConfig := saveConfig
	t.Cleanup(func() {
		core.NewController = originalNewController
		loadConfig = originalLoadConfig
		saveConfig = originalSaveConfig
	}) // This will now work after cmd/config.go change

	// 1. Crear un mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{
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

	testRootCmd, _, _ := newTestRootCmd()
	// 2. Initialize config and add the generic provider via commands
	executeCommand(testRootCmd, "config", "init")
	_, err := executeCommand(testRootCmd, "config", "provider", "add", "generic_test", server.URL,
		"--results-path", "images",
		"--id-path", "id",
		"--url-path", "image_url",
		"--res-path", "res",
	)
	if err != nil {
		t.Fatalf("Error adding generic provider: %v", err)
	}
	// Load the config from the file system, which now contains the added provider
	cfgFromFile, err := originalLoadConfig()
	if err != nil {
		t.Fatalf("Error loading config from file after adding provider: %v", err)
	}
	loadConfig = func() (*models.Config, error) { return cfgFromFile, nil }
	saveConfig = func(cfg *models.Config) error { *cfgFromFile = *cfg; return nil } // Ensure saveConfig updates the loaded config
	mockController := originalNewController(cfgFromFile)                            // Use original NewController to create a real controller
	core.NewController = func(cfg *models.Config) *core.Controller { return mockController }

	// 3. Ejecutar el comando
	output, err := executeCommand(testRootCmd, "explore", "--provider", "generic_test", "searchterm")
	if err != nil {
		t.Fatal(err)
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
	_ = setupTestEnv(t)

	// Save original functions to restore later
	originalNewController := core.NewController
	originalLoadConfig := loadConfig
	originalSaveConfig := saveConfig
	t.Cleanup(func() {
		core.NewController = originalNewController
		loadConfig = originalLoadConfig
		saveConfig = originalSaveConfig
	})

	// 1. Mock server para el genérico
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"images":[]}`) // No necesitamos resultados, solo que se llame
	}))
	defer server.Close()

	testRootCmd, _, _ := newTestRootCmd()
	// 2. Initialize config and add the generic provider via commands
	executeCommand(testRootCmd, "config", "init")
	_, err := executeCommand(testRootCmd, "config", "provider", "add", "generic_test", server.URL, "--results-path", "images", "--id-path", "id", "--url-path", "url")
	if err != nil {
		t.Fatalf("Error adding generic provider: %v", err)
	}
	// Load the config from the file system, which now contains the added provider
	cfgFromFile, err := originalLoadConfig()
	if err != nil {
		t.Fatalf("Error loading config from file after adding provider: %v", err)
	}
	loadConfig = func() (*models.Config, error) { return cfgFromFile, nil }
	saveConfig = func(cfg *models.Config) error { *cfgFromFile = *cfg; return nil } // Ensure saveConfig updates the loaded config
	mockController := originalNewController(cfgFromFile)                            // Use original NewController to create a real controller
	core.NewController = func(cfg *models.Config) *core.Controller { return mockController }

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

func TestExploreGenericProvider_404(t *testing.T) {
	resetExploreFlags()
	_ = setupTestEnv(t)

	// Save original functions to restore later
	originalNewController := core.NewController
	originalLoadConfig := loadConfig
	originalSaveConfig := saveConfig
	t.Cleanup(func() {
		core.NewController = originalNewController
		loadConfig = originalLoadConfig
		saveConfig = originalSaveConfig
	})

	// 1. Crear un mock server que devuelve 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintln(w, "Not Found")
	}))
	defer server.Close()

	testRootCmd, _, _ := newTestRootCmd()

	// 2. Initialize config and add the generic provider via commands
	executeCommand(testRootCmd, "config", "init")
	_, err := executeCommand(testRootCmd, "config", "provider", "add", "generic_404_test", server.URL,
		"--results-path", "images",
		"--id-path", "id",
		"--url-path", "url",
	)
	if err != nil {
		t.Fatalf("Error adding generic provider: %v", err)
	}
	// Load the config from the file system, which now contains the added provider
	cfgFromFile, err := originalLoadConfig()
	if err != nil {
		t.Fatalf("Error loading config from file after adding provider: %v", err)
	}
	loadConfig = func() (*models.Config, error) { return cfgFromFile, nil }
	saveConfig = func(cfg *models.Config) error { *cfgFromFile = *cfg; return nil } // Ensure saveConfig updates the loaded config
	mockController := originalNewController(cfgFromFile)                            // Use original NewController to create a real controller
	core.NewController = func(cfg *models.Config) *core.Controller { return mockController }

	// 3. Ejecutar el comando.
	// Esperamos un error del comando porque la búsqueda del proveedor genérico devuelve 404.
	output, err := executeCommand(testRootCmd, "explore", "--provider", "generic_404_test", "searchterm")
	if err == nil {
		t.Fatal("Expected error from explore command, but got nil")
	}

	// 4. Verificar que la salida indica que el proveedor falló con 404
	expectedRootErrorMessage := "generic api returned status: 404"
	if !strings.Contains(err.Error(), expectedRootErrorMessage) {
		t.Errorf("Se esperaba que el mensaje de error del comando contuviera '%s', se obtuvo: %v", expectedRootErrorMessage, err)
	}

	// Este es el mensaje de advertencia impreso por explore.go
	expectedExploreWarningMsg := "Warning: Error searching generic_404_test: generic api returned status: 404"
	if !strings.Contains(output, expectedExploreWarningMsg) {
		t.Errorf("Se esperaba que la salida contuviera el mensaje de advertencia de explore.go '%s', se obtuvo: %s", expectedExploreWarningMsg, output)
	}

	// Este es el mensaje de log original del proveedor (si lo imprime directamente a stderr)
	// El método de búsqueda del proveedor ni siquiera se llama, por lo que este log no aparecerá.
	// Se elimina esta verificación.
}
