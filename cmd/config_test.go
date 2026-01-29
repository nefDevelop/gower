package cmd

import (
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

	if !strings.Contains(output, "Estructura de configuración creada") {
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
	_, err := executeCommand(rootCmd, "config", "set", "behavior.theme=light")
	if err != nil {
		t.Fatalf("Error ejecutando set: %v", err)
	}

	// Test Get
	output, err := executeCommand(rootCmd, "config", "get", "behavior.theme")
	if err != nil {
		t.Fatalf("Error ejecutando get: %v", err)
	}

	if strings.TrimSpace(output) != "light" {
		t.Errorf("Se esperaba 'light', se obtuvo '%s'", output)
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

	if !strings.Contains(output, "restablecida") {
		t.Errorf("Salida inesperada: %s", output)
	}

	// Verificamos que volvió al valor por defecto (dark)
	output, _ = executeCommand(rootCmd, "config", "get", "behavior.theme")
	if strings.TrimSpace(output) != "dark" {
		t.Errorf("Se esperaba 'dark' después del reset, se obtuvo '%s'", output)
	}
}

func TestConfigExportAndImport(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	exportFile := filepath.Join(tmpDir, "backup.json")

	// Test Export
	_, err := executeCommand(rootCmd, "config", "export", exportFile)
	if err != nil {
		t.Fatalf("Error ejecutando export: %v", err)
	}

	if _, err := os.Stat(exportFile); os.IsNotExist(err) {
		t.Errorf("El archivo exportado no existe")
	}

	// Modificamos la config actual
	executeCommand(rootCmd, "config", "set", "behavior.theme=light")

	// Test Import (debería restaurar 'dark' que es lo que se exportó)
	executeCommand(rootCmd, "config", "import", exportFile)

	output, _ := executeCommand(rootCmd, "config", "get", "behavior.theme")
	if strings.TrimSpace(output) != "dark" {
		t.Errorf("Se esperaba 'dark' después de importar, se obtuvo '%s'", output)
	}
}
