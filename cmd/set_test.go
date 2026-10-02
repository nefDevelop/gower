package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gower/internal/core"
	"gower/pkg/models"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func resetSetFlags() {
	setID = ""
	setURL = ""
	setRandom = false
	setTheme = ""
	setFromFavorites = false
	setMultiMonitor = ""
	setCommand = ""
	setTargetMonitor = ""
}

func TestController_GetWallpaperAndDownload(t *testing.T) {
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("image"))
	}))
	defer server.Close()

	cfg := &models.Config{}
	ctrl := core.NewController(cfg)
	wp := models.Wallpaper{ID: "test-1", URL: server.URL + "/img.jpg"}
	if err := ctrl.AddWallpaperToFeed(wp); err != nil {
		t.Fatalf("Failed to add wallpaper to feed: %v", err)
	}

	// Test GetWallpaper
	got, err := ctrl.GetWallpaper("test-1")
	assert.NoError(t, err)
	assert.Equal(t, wp.URL, got.URL)

	// Test DownloadWallpaper
	path, err := ctrl.DownloadWallpaper(*got)
	assert.NoError(t, err)
	_, err = os.Stat(path)
	assert.NoError(t, err, "Downloaded file should exist at %s", path)
}

func TestSetUndoCommand(t *testing.T) {
	requireRealFactories(t)

	t.Setenv("XDG_CURRENT_DESKTOP", "test")
	resetSetFlags()

	// Sustituye NewWallpaperChanger para no invocar ajustadores reales:
	// en un entorno sin Niri/swww/awww el set fallaria. Se restaura desde
	// el snapshot canónico, no desde el valor actual, para no propagar un
	// stub filtrado por otro test.
	defer func() { core.NewWallpaperChanger = realNewWallpaperChanger }()
	core.NewWallpaperChanger = func(desktopEnv string, respectDarkMode ...bool) *core.WallpaperChanger {
		wc := &core.WallpaperChanger{Env: desktopEnv}
		wc.SetWallpapersFunc = func(_ []string, _ []core.Monitor, _ string) error { return nil }
		wc.DetectMonitorsFunc = func() ([]core.Monitor, error) {
			return []core.Monitor{{Name: "test-monitor"}}, nil
		}
		return wc
	}

	_, cleanup := setupTestHomeWithState(t, &State{
		CurrentWallpaperID:  "current-wp",
		PreviousWallpaperID: "previous-wp",
		PreviousWallpapers:  []string{"previous-wp", "previous-wp-2"},
	})
	defer cleanup()

	// Mock server for image download
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("fake image content")); err != nil {
			t.Logf("Error writing response in mock server: %v", err)
		}
	}))
	defer server.Close()

	// Populate feed with the wallpaper we expect to be set
	cfg, _ := loadConfig()
	ctrl := core.NewController(cfg)
	if err := ctrl.AddWallpaperToFeed(models.Wallpaper{
		ID:     "previous-wp",
		URL:    server.URL + "/image.jpg",
		Source: "test",
	}); err != nil {
		t.Fatalf("Failed to add wallpaper to feed: %v", err)
	}
	if err := ctrl.AddWallpaperToFeed(models.Wallpaper{
		ID:     "previous-wp-2",
		URL:    server.URL + "/image2.jpg",
		Source: "test",
	}); err != nil {
		t.Fatalf("Failed to add wallpaper to feed: %v", err)
	}

	// Execute the undo command and capture output
	// We need to re-initialize the root command for each test run to avoid state leakage
	testRootCmd, _, _ := newTestRootCmd(t)
	output, err := executeCommand(testRootCmd, "set", "undo")

	assert.NoError(t, err)
	assert.Contains(t, output, "Wallpaper(s) set successfully")
	assert.Contains(t, output, "Preparing wallpaper: previous-wp")
	assert.Contains(t, output, "Preparing wallpaper: previous-wp-2")
}

// setupTestHomeWithState is a helper for tests that need a pre-configured state.json
func setupTestHomeWithState(t *testing.T, state *State) (string, func()) {
	resetAllFlags(t)

	tempDir, err := os.MkdirTemp("", "gower-test-home-")
	assert.NoError(t, err)

	originalHome := os.Getenv("HOME")
	t.Setenv("HOME", tempDir)
	// os.UserConfigDir() da prioridad a XDG_CONFIG_HOME sobre HOME: sin
	// neutralizarlo el test leería y escribiría en el config dir real.
	t.Setenv("XDG_CONFIG_HOME", "")

	// Create .gower dir and write state
	gowerDir := filepath.Join(tempDir, ".gower")
	err = os.MkdirAll(gowerDir, 0755)
	assert.NoError(t, err)

	statePath := filepath.Join(gowerDir, "state.json")
	stateData, err := json.Marshal(state)
	assert.NoError(t, err)
	err = os.WriteFile(statePath, stateData, 0644)
	assert.NoError(t, err)

	// Create a dummy config to satisfy ensureConfig
	configPath := filepath.Join(gowerDir, "config.json")
	err = os.WriteFile(configPath, []byte("{}"), 0644)
	assert.NoError(t, err)

	cleanup := func() {
		_ = os.Setenv("HOME", originalHome)
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// newTestRootCmd construye un root temporal sin PersistentPreRun, para que los
// tests no inicialicen el logger real.
//
// Reutiliza los mismos *cobra.Command que rootCmd, y AddCommand les reasigna
// el campo parent, que es estado global. Si no se restaura, los subcomandos
// quedan colgando de este root huérfano: un test posterior que ejecute rootCmd
// los sigue alcanzar, pero al imprimir, cmd.Println sube por la cadena de
// padres hasta este root, cuyo writer es nil, y escribe a os.Stdout en lugar
// del buffer del test. El síntoma es una salida vacía sin ningún error.
func newTestRootCmd(t *testing.T) (*cobra.Command, *CLIConfig, *bytes.Buffer) {
	tempRoot := &cobra.Command{Use: "gower"}
	var cfg CLIConfig
	var out bytes.Buffer
	tempRoot.SetOut(&out)
	tempRoot.SetErr(&out)

	adopted := []*cobra.Command{setCmd, exploreCmd, configCmd}
	for _, c := range adopted {
		// Reset output of subcommands to ensure they inherit from tempRoot
		c.SetOut(nil)
		c.SetErr(nil)
		tempRoot.AddCommand(c)
	}

	t.Cleanup(func() {
		for _, c := range adopted {
			// RemoveCommand quita la entrada de rootCmd y pone parent=nil, así
			// el AddCommand posterior deja exactamente una copia con el
			// parent original.
			rootCmd.RemoveCommand(c)
			c.SetOut(nil)
			c.SetErr(nil)
			rootCmd.AddCommand(c)
		}
	})

	// Re-initialize flags for subcommands if necessary
	// This is a simplified setup. A full setup would re-run all init() functions
	// or use a factory pattern for commands.
	return tempRoot, &cfg, &out
}
