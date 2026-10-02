package cmd

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"testing"

	"gower/internal/core"
	"gower/pkg/models"

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

// cmdGlobals agrupa todo el estado global mutable del paquete cmd, para poder
// snapshotarlo y restaurarlo de una vez.
type cmdGlobals struct {
	config CLIConfig

	exploreProvider                                                  string
	exploreAll, exploreForceUpdate                                   bool
	exploreMinWidth, exploreMinHeight, explorePage                   int
	exploreAspectRatio, exploreColor                                 string
	statusJSON, statusProviders, statusStorage                       bool
	statusDaemon, statusSystem, statusMonitors, statusWallpaper      bool
	favPage, favLimit                                                int
	favNotes, favColor, favFile                                      string
	favForce, favAll                                                 bool
	feedPage, feedLimit, watchInterval                               int
	feedTheme, feedColor, feedSort                                   string
	feedRefresh, feedForce, feedDetailed, feedAll, feedFromFavorites bool
	downloadOutput, downloadTheme                                    string
	downloadRandom, downloadFromFavorites, downloadTag               bool
	downloadToCollection                                             bool
	setID, setURL, setTheme, setMultiMonitor, setCommand             string
	setTargetMonitor                                                 string
	setRandom, setFromFavorites                                      bool
	exportFile                                                       string
	exportIncludeImages                                              bool
	providerKey, providerResultsPath, providerIDPath                 string
	providerURLPath, providerResPath                                 string
	wpDelete, wpFile, wpForce                                        bool
	daemonInterval                                                   int
	daemonTheme                                                      string
	daemonFromFavorites, daemonForce, daemonJSON, daemonForeground   bool

	loadConfig  func() (*models.Config, error)
	saveConfig  func(*models.Config) error
	loadState   func() (*State, error)
	saveState   func(*State) error
	statePathFn func() (string, error)
}

func snapshotGlobals() cmdGlobals {
	return cmdGlobals{
		config:                config,
		exploreProvider:       exploreProvider,
		exploreAll:            exploreAll,
		exploreForceUpdate:    exploreForceUpdate,
		exploreMinWidth:       exploreMinWidth,
		exploreMinHeight:      exploreMinHeight,
		explorePage:           explorePage,
		exploreAspectRatio:    exploreAspectRatio,
		exploreColor:          exploreColor,
		statusJSON:            statusJSON,
		statusProviders:       statusProviders,
		statusStorage:         statusStorage,
		statusDaemon:          statusDaemon,
		statusSystem:          statusSystem,
		statusMonitors:        statusMonitors,
		statusWallpaper:       statusWallpaper,
		favPage:               favPage,
		favLimit:              favLimit,
		favNotes:              favNotes,
		favColor:              favColor,
		favFile:               favFile,
		favForce:              favForce,
		favAll:                favAll,
		feedPage:              feedPage,
		feedLimit:             feedLimit,
		watchInterval:         watchInterval,
		feedTheme:             feedTheme,
		feedColor:             feedColor,
		feedSort:              feedSort,
		feedRefresh:           feedRefresh,
		feedForce:             feedForce,
		feedDetailed:          feedDetailed,
		feedAll:               feedAll,
		feedFromFavorites:     feedFromFavorites,
		downloadOutput:        downloadOutput,
		downloadTheme:         downloadTheme,
		downloadRandom:        downloadRandom,
		downloadFromFavorites: downloadFromFavorites,
		downloadTag:           downloadTag,
		downloadToCollection:  downloadToCollection,
		setID:                 setID,
		setURL:                setURL,
		setTheme:              setTheme,
		setMultiMonitor:       setMultiMonitor,
		setCommand:            setCommand,
		setTargetMonitor:      setTargetMonitor,
		setRandom:             setRandom,
		setFromFavorites:      setFromFavorites,
		exportFile:            exportFile,
		exportIncludeImages:   exportIncludeImages,
		providerKey:           providerKey,
		providerResultsPath:   providerResultsPath,
		providerIDPath:        providerIDPath,
		providerURLPath:       providerURLPath,
		providerResPath:       providerResPath,
		wpDelete:              wpDelete,
		wpFile:                wpFile,
		wpForce:               wpForce,
		daemonInterval:        daemonInterval,
		daemonTheme:           daemonTheme,
		daemonFromFavorites:   daemonFromFavorites,
		daemonForce:           daemonForce,
		daemonJSON:            daemonJSON,
		daemonForeground:      daemonForeground,
		loadConfig:            loadConfig,
		saveConfig:            saveConfig,
		loadState:             loadState,
		saveState:             saveState,
		statePathFn:           stateFilePath,
	}
}

func restoreGlobals(g cmdGlobals) {
	config = g.config
	exploreProvider, exploreAll, exploreForceUpdate = g.exploreProvider, g.exploreAll, g.exploreForceUpdate
	exploreMinWidth, exploreMinHeight, explorePage = g.exploreMinWidth, g.exploreMinHeight, g.explorePage
	exploreAspectRatio, exploreColor = g.exploreAspectRatio, g.exploreColor

	statusJSON, statusProviders, statusStorage = g.statusJSON, g.statusProviders, g.statusStorage
	statusDaemon, statusSystem, statusMonitors, statusWallpaper = g.statusDaemon, g.statusSystem, g.statusMonitors, g.statusWallpaper

	favPage, favLimit, favNotes, favColor, favFile = g.favPage, g.favLimit, g.favNotes, g.favColor, g.favFile
	favForce, favAll = g.favForce, g.favAll

	feedPage, feedLimit, watchInterval = g.feedPage, g.feedLimit, g.watchInterval
	feedTheme, feedColor, feedSort = g.feedTheme, g.feedColor, g.feedSort
	feedRefresh, feedForce, feedDetailed, feedAll, feedFromFavorites = g.feedRefresh, g.feedForce, g.feedDetailed, g.feedAll, g.feedFromFavorites

	downloadOutput, downloadTheme = g.downloadOutput, g.downloadTheme
	downloadRandom, downloadFromFavorites, downloadTag, downloadToCollection = g.downloadRandom, g.downloadFromFavorites, g.downloadTag, g.downloadToCollection

	setID, setURL, setTheme, setMultiMonitor, setCommand, setTargetMonitor = g.setID, g.setURL, g.setTheme, g.setMultiMonitor, g.setCommand, g.setTargetMonitor
	setRandom, setFromFavorites = g.setRandom, g.setFromFavorites

	exportFile, exportIncludeImages = g.exportFile, g.exportIncludeImages

	providerKey, providerResultsPath = g.providerKey, g.providerResultsPath
	providerIDPath, providerURLPath, providerResPath = g.providerIDPath, g.providerURLPath, g.providerResPath

	wpDelete, wpFile, wpForce = g.wpDelete, g.wpFile, g.wpForce

	daemonInterval, daemonTheme = g.daemonInterval, g.daemonTheme
	daemonFromFavorites, daemonForce, daemonJSON, daemonForeground = g.daemonFromFavorites, g.daemonForce, g.daemonJSON, g.daemonForeground

	loadConfig, saveConfig = g.loadConfig, g.saveConfig
	loadState, saveState, stateFilePath = g.loadState, g.saveState, g.statePathFn
}

// resetAllFlags registra en t.Cleanup la restauración de todo el estado global
// mutable del paquete: los flags de cada comando, la config global y los seams
// de funciones que los tests sustituyen.
//
// Antes cada test restauraba a mano sólo lo que tocaba, así que un test que
// abortaba con t.Fatal dejaba flags sucios que rompían de forma silenciosa al
// siguiente: config.JSONOutput y --json cambian el formato de la salida,
// wpDelete borra wallpapers en lugar de mostrarlos, y exportFile hace que
// "export config" escriba a una ruta temporal ya borrada en vez de imprimir.
//
// Usa t.Cleanup y no un reset al inicio porque así la restauración se
// garantiza aunque el test aborte, y porque un reset al inicio dejaría intacto
// el valor que un test anterior hubiera filtrado.
func resetAllFlags(t *testing.T) {
	t.Helper()
	saved := snapshotGlobals()
	t.Cleanup(func() {
		restoreGlobals(saved)
		// Los argumentos del root también son estado compartido entre tests.
		rootCmd.SetArgs(nil)
	})
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
	resetAllFlags(t)

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
	resetAllFlags(t)

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
