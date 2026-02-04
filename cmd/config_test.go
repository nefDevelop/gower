package cmd

import (
	"fmt" // Add this import
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigInit(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	output, err := executeCommand(rootCmd, "config", "init")
	if err != nil {
		t.Fatalf("Error ejecutando init: %v", err)
	}

	if !strings.Contains(output, "Estructura de configuración creada en:") {
		t.Errorf("Salida inesperada: %s", output)
	}

	configPath := filepath.Join(tmpDir, ".gower", "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("El archivo de configuración no fue creado")
	}
}

func TestConfigShow(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

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
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

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
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

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
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	exportFile := filepath.Join(tmpDir, "backup.json")

	// Test Export
	exportOutput, err := executeCommand(rootCmd, "config", "export", exportFile)
	if err != nil {
		t.Fatalf("Error ejecutando export: %v", err)
	}

	expectedExportOutput := fmt.Sprintf("Configuración exportada a: %s", exportFile)
	if !strings.Contains(exportOutput, expectedExportOutput) {
		t.Errorf("Expected output to contain '%s', got '%s'", expectedExportOutput, exportOutput)
	}

	if _, err := os.Stat(exportFile); os.IsNotExist(err) {
		t.Errorf("El archivo exportado no existe")
	}

	// Modificamos la config actual
	executeCommand(rootCmd, "config", "set", "behavior.theme=light")

	// Test Import (debería restaurar 'dark' que es lo que se exportó)
	importOutput, err := executeCommand(rootCmd, "config", "import", exportFile)
	if err != nil {
		t.Fatalf("Error ejecutando import: %v", err)
	}

	if !strings.Contains(importOutput, "Configuración importada exitosamente.") {
		t.Errorf("Expected output to contain 'Configuración importada exitosamente.', got '%s'", importOutput)
	}

	output, _ := executeCommand(rootCmd, "config", "get", "behavior.theme")
	if strings.TrimSpace(output) != "" {
		t.Errorf("Se esperaba '' después de importar, se obtuvo '%s'", output)
	}
}
