package cmd

import (
	"os"
	"strings"
	"testing"
)

// resetExploreFlags reinicia las variables globales de los flags de explore
// para evitar contaminación entre tests.
func resetExploreFlags() {
	exploreProvider = ""
	exploreAll = false
	exploreMinWidth = 0
	exploreColor = ""
}

func TestExploreDefault(t *testing.T) {
	resetExploreFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	// Inicializamos configuración
	executeCommand(rootCmd, "config", "init")

	output, err := executeCommand(rootCmd, "explore", "nature")
	if err != nil {
		t.Fatalf("Error ejecutando explore: %v", err)
	}

	if !strings.Contains(output, "Explorando: 'nature'") {
		t.Errorf("Salida esperada conteniendo 'Explorando: 'nature'', se obtuvo: %s", output)
	}
	// Por defecto debería consultar algún proveedor (wallhaven o reddit)
	if !strings.Contains(output, "Consultando proveedor:") {
		t.Errorf("Se esperaba mensaje de consulta a proveedor")
	}
}

func TestExploreProvider(t *testing.T) {
	resetExploreFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	output, err := executeCommand(rootCmd, "explore", "--provider", "reddit", "space")
	if err != nil {
		t.Fatalf("Error ejecutando explore: %v", err)
	}

	if !strings.Contains(output, "Consultando proveedor: reddit") {
		t.Errorf("Se esperaba 'Consultando proveedor: reddit', se obtuvo: %s", output)
	}
}

func TestExploreAll(t *testing.T) {
	resetExploreFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")
	// Establecer rate limit a 0 para acelerar el test y evitar sleep
	executeCommand(rootCmd, "config", "set", "limits.rate_limit_period=0")

	output, err := executeCommand(rootCmd, "explore", "--all", "cars")
	if err != nil {
		t.Fatalf("Error ejecutando explore: %v", err)
	}

	if !strings.Contains(output, "Consultando proveedor: wallhaven") {
		t.Errorf("Se esperaba wallhaven en la salida")
	}
	if !strings.Contains(output, "Consultando proveedor: reddit") {
		t.Errorf("Se esperaba reddit en la salida")
	}
}

func TestExploreFilters(t *testing.T) {
	resetExploreFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	output, err := executeCommand(rootCmd, "explore", "--min-width", "2560", "--color", "#ff0000", "red")
	if err != nil {
		t.Fatalf("Error ejecutando explore: %v", err)
	}

	if !strings.Contains(output, "Filtro Min-Width: 2560px") {
		t.Errorf("Se esperaba filtro min-width en la salida")
	}
	if !strings.Contains(output, "Filtro Color: #ff0000") {
		t.Errorf("Se esperaba filtro color en la salida")
	}
}
