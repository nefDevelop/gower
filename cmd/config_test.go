package cmd

import (
	"fmt" // Add this import
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConfigInit(t *testing.T) {
	tmpDir := setupTestEnv(t)

	output, err := executeCommand(rootCmd, "config", "init")
	if err != nil {
		t.Fatalf("Error ejecutando init: %v", err)
	}

	if !strings.Contains(output, "Estructura de configuración creada en:") {
		t.Errorf("Salida inesperada: %s", output)
	}

	var configPath string
	if runtime.GOOS == "windows" {
		// En Windows, UserConfigDir apunta a APPDATA
		configPath = filepath.Join(tmpDir, "gower", "config.json")
	} else {
		// En Linux/macOS, apunta a HOME/.config
		configPath = filepath.Join(tmpDir, ".config", "gower", "config.json")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("El archivo de configuración no fue creado")
	}
}

func TestConfigShow(t *testing.T) {
	_ = setupTestEnv(t) // No necesitamos la ruta del dir, solo configurar el entorno

	executeCommand(rootCmd, "config", "init")

	output, err := executeCommand(rootCmd, "config", "show")
	if err != nil {
		t.Fatalf("Error ejecutando show: %v", err)
	}

	if !strings.Contains(output, "providers") || !strings.Contains(output, "behavior") {
		t.Errorf("La salida no parece ser el JSON de configuración esperado")
	}
}

func TestConfigSetAndGet(t *testing.T) {
	_ = setupTestEnv(t)

	executeCommand(rootCmd, "config", "init")

	// Test Set
	setOutput, err := executeCommand(rootCmd, "config", "set", "behavior.theme=light")
	if err != nil {
		t.Fatalf("Error ejecutando set: %v", err)
	}
	if !strings.Contains(setOutput, "Configuración actualizada: behavior.theme = light") {
		t.Errorf("Salida inesperada para set: %s", setOutput)
	}

	// Test Get
	output, err := executeCommand(rootCmd, "config", "get", "behavior.theme")
	if err != nil {
		t.Fatalf("Error ejecutando get: %v", err)
	}

	if strings.TrimSpace(output) != "light" {
		t.Errorf("Se esperaba 'light', se obtuvo '%s'", output)
	}

	// Test new fields
	executeCommand(rootCmd, "config", "set", "behavior.save_favorites_to_folder=true")
	output, _ = executeCommand(rootCmd, "config", "get", "behavior.save_favorites_to_folder")
	if strings.TrimSpace(output) != "true" {
		t.Errorf("Se esperaba 'true', se obtuvo '%s'", output)
	}

	executeCommand(rootCmd, "config", "set", "paths.index_wallpapers=true")
	output, _ = executeCommand(rootCmd, "config", "get", "paths.index_wallpapers")
	if strings.TrimSpace(output) != "true" {
		t.Errorf("Se esperaba 'true', se obtuvo '%s'", output)
	}
}

func TestConfigReset(t *testing.T) {
	_ = setupTestEnv(t)

	executeCommand(rootCmd, "config", "init")
	// Cambiamos un valor
	executeCommand(rootCmd, "config", "set", "behavior.theme=light")

	// Ejecutamos reset
	output, err := executeCommand(rootCmd, "config", "reset")
	if err != nil {
		t.Fatalf("Error ejecutando reset: %v", err)
	}

	if !strings.Contains(output, "Configuración restablecida a los valores por defecto.") {
		t.Errorf("Salida inesperada: %s", output)
	}

	// Verificamos que volvió al valor por defecto (vacío)
	output, _ = executeCommand(rootCmd, "config", "get", "behavior.theme")
	if strings.TrimSpace(output) != "" {
		t.Errorf("Se esperaba '' después del reset, se obtuvo '%s'", output)
	}
}

func TestConfigExportAndImport(t *testing.T) {
	tmpDir := setupTestEnv(t)

	executeCommand(rootCmd, "config", "init")

	exportFile := filepath.Join(tmpDir, "backup.json")

	// Test Export
	exportOutput, err := executeCommand(rootCmd, "export", "config", "--file", exportFile)
	if err != nil {
		t.Fatalf("Error ejecutando export: %v", err)
	}

	expectedExportOutput := fmt.Sprintf("Configuration exported to: %s", exportFile)
	if !strings.Contains(exportOutput, expectedExportOutput) {
		t.Errorf("Expected output to contain '%s', got '%s'", expectedExportOutput, exportOutput)
	}

	if _, err := os.Stat(exportFile); os.IsNotExist(err) {
		t.Errorf("El archivo exportado no existe")
	}

	// Modificamos la config actual
	executeCommand(rootCmd, "config", "set", "behavior.theme=light")

	// Test Import (debería restaurar 'dark' que es lo que se exportó)
	importOutput, err := executeCommand(rootCmd, "import", "config", exportFile)
	if err != nil {
		t.Fatalf("Error ejecutando import: %v", err)
	}

	if !strings.Contains(importOutput, "Configuration imported successfully") {
		t.Errorf("Expected output to contain 'Configuration imported successfully', got '%s'", importOutput)
	}

	output, _ := executeCommand(rootCmd, "config", "get", "behavior.theme")
	if strings.TrimSpace(output) != "" {
		t.Errorf("Se esperaba '' después de importar, se obtuvo '%s'", output)
	}
}

func TestConfigUpdate(t *testing.T) {
	tmpDir := setupTestEnv(t)

	// 1. Crear un archivo de configuración parcial/antiguo manualmente
	var configDir string
	if runtime.GOOS == "windows" {
		configDir = filepath.Join(tmpDir, "gower")
	} else {
		configDir = filepath.Join(tmpDir, ".config", "gower")
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir for test: %v", err)
	}

	configFile := filepath.Join(configDir, "config.json")
	// Este JSON carece de "from_favorites"
	oldJSON := []byte(`{"behavior":{"theme":"dark","change_interval":60}}`)
	if err := os.WriteFile(configFile, oldJSON, 0644); err != nil {
		t.Fatalf("Failed to write old config: %v", err)
	}

	// 2. Ejecutar config update
	output, err := executeCommand(rootCmd, "config", "update")
	if err != nil {
		t.Fatalf("Error executing config update: %v", err)
	}

	if !strings.Contains(output, "Configuración actualizada con nuevos campos") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// 3. Verificar que el archivo ahora contiene "from_favorites"
	newContent, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read updated config: %v", err)
	}
	if !strings.Contains(string(newContent), "from_favorites") {
		t.Errorf("Updated config file should contain 'from_favorites'")
	}
}

func TestConfigFromFavorites(t *testing.T) {
	_ = setupTestEnv(t)

	executeCommand(rootCmd, "config", "init")

	// Test Set
	executeCommand(rootCmd, "config", "set", "behavior.from_favorites=true")

	// Test Get
	output, err := executeCommand(rootCmd, "config", "get", "behavior.from_favorites")
	if err != nil {
		t.Fatalf("Error executing get: %v", err)
	}
	if strings.TrimSpace(output) != "true" {
		t.Errorf("Expected 'true', got '%s'", output)
	}
}

func TestConfigGetConfigFolder(t *testing.T) {
	tmpDir := setupTestEnv(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	output, err := executeCommand(rootCmd, "config", "get", "config-folder")
	if err != nil {
		t.Fatalf("Error executing config get config-folder: %v", err)
	}

	expectedConfigDir := ""
	if runtime.GOOS == "windows" {
		expectedConfigDir = filepath.Join(tmpDir, "gower")
	} else {
		expectedConfigDir = filepath.Join(tmpDir, ".config", "gower")
	}

	if strings.TrimSpace(output) != expectedConfigDir {
		t.Errorf("Expected config folder '%s', got '%s'", expectedConfigDir, strings.TrimSpace(output))
	}
}
