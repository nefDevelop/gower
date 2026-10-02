package cmd

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"testing"

	"gower/internal/core"

	"github.com/spf13/cobra"
)

// Fábricas reales de core, capturadas una única vez al inicio del proceso de
// test y nunca reasignadas. Los tests que las sustituyen deben restaurar a
// partir de aquí, no de lo que encuentren en el momento: así el orden de
// ejecución es irrelevante y un stub filtrado por un test_abortado no puede
// contaminar a los siguientes.
var (
	realNewController       = core.NewController
	realNewWallpaperChanger = core.NewWallpaperChanger
)

// contaminatedFactory devuelve el nombre de la fábrica de core que no coincide
// con el snapshot, o "" si todas están intactas.
func contaminatedFactory() string {
	for _, f := range []struct {
		name     string
		got, exp any
	}{
		{"core.NewController", core.NewController, realNewController},
		{"core.NewWallpaperChanger", core.NewWallpaperChanger, realNewWallpaperChanger},
	} {
		if reflect.ValueOf(f.got).Pointer() != reflect.ValueOf(f.exp).Pointer() {
			return f.name
		}
	}
	return ""
}

// requireRealFactories falla si algún test anterior dejó instalada una fábrica
// stub en core. Se invoca al inicio de cada helper que sustituye las fábricas,
// que es justo donde una contaminación silenciosa haría que el test ejercitara
// algo distinto de lo que cree. TestMain sólo detecta fugas que sobreviven
// hasta el final del paquete, que es un caso mucho más raro.
func requireRealFactories(t *testing.T) {
	t.Helper()
	if name := contaminatedFactory(); name != "" {
		t.Fatalf("%s no fue restaurado por un test anterior: el estado global de core está contaminado", name)
	}
}

// TestMain comprueba al terminar que las fábricas de core siguen siendo las
// reales, como red de seguridad global.
func TestMain(m *testing.M) {
	code := m.Run()
	if name := contaminatedFactory(); name != "" {
		fmt.Fprintf(os.Stderr, "cmd: %s no fue restaurado por algún test\n", name)
		code = 1
	}
	os.Exit(code)
}

// executeCommand executes a Cobra command and captures its output.
func executeCommand(root *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

// setupTestHome creates a temporary directory and sets the HOME environment variable.
func setupTestHome(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "gower-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", tmpDir)
	// For Windows compatibility
	t.Setenv("USERPROFILE", tmpDir)
	// os.UserConfigDir() da prioridad a XDG_CONFIG_HOME sobre HOME, así que
	// hay que neutralizarlo o el test opera sobre el config dir real.
	t.Setenv("XDG_CONFIG_HOME", "")
	return tmpDir
}

// setupTestEnv crea un directorio temporal y establece la variable de entorno
// apropiada (HOME o APPDATA) para que os.UserConfigDir() apunte dentro
// del directorio temporal. Esto hace las pruebas herméticas y multiplataforma.
func setupTestEnv(t *testing.T) string {
	tmpDir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", tmpDir)
	} else {
		t.Setenv("HOME", tmpDir)
		t.Setenv("USERPROFILE", tmpDir)
	}
	// Asegurarse de que XDG_CONFIG_HOME no esté establecido, para que se use el fallback a HOME/.config.
	t.Setenv("XDG_CONFIG_HOME", "")
	return tmpDir
}
