package core

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stubWallpaperExecution sustituye las costuras que acaban tocando el sistema
// y registra lo que se ejecutaria en su lugar. Sin esto los tests invocan de
// verdad gsettings, dbus-send, swaymsg, swaybg, nitrogen, dms y quickshell: en
// un escritorio real eso cambia el wallpaper del usuario, abre nitrogen y deja
// demonios de awww/swww en segundo plano.
type stubWallpaperExecution struct {
	commands []string
	daemons  []string
	runErr   error
}

func (s *stubWallpaperExecution) install(t *testing.T) {
	t.Helper()
	origRun, origStart := runCommand, startBackgroundProcess
	origExists, origIsProc, origIsRunning := commandExists, isProcessRunning, IsProcessRunning

	t.Cleanup(func() {
		runCommand, startBackgroundProcess = origRun, origStart
		commandExists, isProcessRunning, IsProcessRunning = origExists, origIsProc, origIsRunning
	})

	runCommand = func(cmd *exec.Cmd) error {
		s.commands = append(s.commands, cmd.Path+" "+strings.Join(cmd.Args[1:], " "))
		return s.runErr
	}
	startBackgroundProcess = func(name string) error {
		s.daemons = append(s.daemons, name)
		return nil
	}
	commandExists = func(string) bool { return true }
	isProcessRunning = func(string) bool { return false }
	IsProcessRunning = func(string) bool { return false }
}

func (s *stubWallpaperExecution) binaries() []string {
	out := make([]string, 0, len(s.commands))
	for _, c := range s.commands {
		out = append(out, filepath.Base(strings.Fields(c)[0]))
	}
	return out
}

func TestNewWallpaperChanger_Manual(t *testing.T) {
	wc := NewWallpaperChanger("kde")
	if wc.Env != "kde" {
		t.Errorf("Expected Env to be 'kde', got '%s'", wc.Env)
	}

	wc = NewWallpaperChanger("gnome")
	if wc.Env != "gnome" {
		t.Errorf("Expected Env to be 'gnome', got '%s'", wc.Env)
	}

	// Test normalization
	wc = NewWallpaperChanger("KDE Plasma")
	if wc.Env != "kde" {
		t.Errorf("Expected Env to be normalized to 'kde', got '%s'", wc.Env)
	}
}

func TestDetectDesktopEnv(t *testing.T) {
	// Mock isProcessRunning to always return false to test env vars
	originalIsProcessRunning := isProcessRunning
	defer func() { isProcessRunning = originalIsProcessRunning }()
	isProcessRunning = func(name string) bool { return false }

	// Save original env var
	originalEnv := os.Getenv("XDG_CURRENT_DESKTOP")
	defer func() { _ = os.Setenv("XDG_CURRENT_DESKTOP", originalEnv) }()

	// Test GNOME
	_ = os.Setenv("XDG_CURRENT_DESKTOP", "GNOME")
	env := DetectDesktopEnv()
	if !strings.Contains(env, "gnome") {
		t.Errorf("Expected to detect 'gnome', got '%s'", env)
	}

	// Test KDE
	_ = os.Setenv("XDG_CURRENT_DESKTOP", "KDE")
	env = DetectDesktopEnv()
	if !strings.Contains(env, "kde") {
		t.Errorf("Expected to detect 'kde', got '%s'", env)
	}
}

// This test is limited because it can't actually execute the commands.
// Comprueba que cada entorno construye el comando correcto y que SetWallpapers
// propaga el error del ejecutor, sin lanzar nada real.
func TestSetWallpaper(t *testing.T) {
	wallpaper := filepath.Join(t.TempDir(), "wallpaper.jpg")
	if err := os.WriteFile(wallpaper, []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	// Entorno -> binario que se espera ejecutar. Con DetectMonitorsFunc fija a
	// un unico monitor llamado, el resultado no depende de los monitores que la
	// maquina tenga realmente (hyprctl, swaymsg o xrandr).
	monitors := []Monitor{{ID: "eDP-1", Name: "eDP-1", Width: 1920, Height: 1080, Primary: true}}

	testCases := []struct {
		env     string
		wantBin string
	}{
		{env: "kde", wantBin: "dbus-send"},
		{env: "gnome", wantBin: "gsettings"},
		{env: "feh", wantBin: "feh"},
		{env: "nitrogen", wantBin: "nitrogen"},
		{env: "sway", wantBin: "swaymsg"},
		{env: "niri", wantBin: "awww"},
		{env: "dms", wantBin: "dms"},
		{env: "swww", wantBin: "swww"},
		{env: "awww", wantBin: "awww"},
	}

	for _, tc := range testCases {
		t.Run(tc.env, func(t *testing.T) {
			stub := &stubWallpaperExecution{}
			stub.install(t)

			wc := NewWallpaperChanger(tc.env)
			wc.DetectMonitorsFunc = func() ([]Monitor, error) { return monitors, nil }
			if err := wc.SetWallpapers([]string{wallpaper}, monitors, "clone"); err != nil {
				t.Fatalf("SetWallpapers devolvio error inesperado: %v", err)
			}
			if len(stub.commands) == 0 {
				t.Fatalf("Se esperaba algun comando para '%s', no se ejecuto ninguno", tc.env)
			}
			if got := stub.binaries(); !contains(got, tc.wantBin) {
				t.Errorf("Se esperaba ejecutar '%s', se ejecutaron %v", tc.wantBin, got)
			}
		})
	}

	t.Run("unsupported", func(t *testing.T) {
		stub := &stubWallpaperExecution{}
		stub.install(t)

		wc := NewWallpaperChanger("unsupported")
		err := wc.SetWallpapers([]string{wallpaper}, []Monitor{{Name: "eDP-1"}}, "clone")
		if err == nil {
			t.Fatal("Se esperaba un error para un entorno no soportado")
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("Se esperaba 'unsupported' en el error, se obtuvo '%s'", err)
		}
		if len(stub.commands) != 0 {
			t.Errorf("No deberia ejecutar nada, ejecuto %v", stub.commands)
		}
	})

	t.Run("propaga error del ejecutor", func(t *testing.T) {
		stub := &stubWallpaperExecution{runErr: os.ErrPermission}
		stub.install(t)

		wc := NewWallpaperChanger("swww")
		wc.DetectMonitorsFunc = func() ([]Monitor, error) {
			return []Monitor{{Name: "eDP-1"}}, nil
		}
		err := wc.SetWallpapers([]string{wallpaper}, []Monitor{{Name: "eDP-1"}}, "clone")
		if err == nil {
			t.Fatal("Se esperaba propagar el error del ejecutor")
		}
		// applyToMonitors agrega con %v, no con %w, asi que el error no queda
		// envuelto para errors.Is: solo se puede comprobar el mensaje.
		if !strings.Contains(err.Error(), os.ErrPermission.Error()) {
			t.Errorf("Se esperaba mencionar '%s', se obtuvo %v", os.ErrPermission, err)
		}
	})
}

// El entorno niri arranca un demonio si no hay ninguno activo. Con las costuras
// puesta debe registrarlos, no lanzarlos.
func TestSetWallpaper_NiriStartsDaemon(t *testing.T) {
	stub := &stubWallpaperExecution{}
	stub.install(t)

	wallpaper := filepath.Join(t.TempDir(), "wallpaper.jpg")
	if err := os.WriteFile(wallpaper, []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	monitors := []Monitor{{ID: "eDP-1", Name: "eDP-1"}}
	wc := NewWallpaperChanger("niri")
	wc.DetectMonitorsFunc = func() ([]Monitor, error) { return monitors, nil }
	if err := wc.SetWallpapers([]string{wallpaper}, monitors, "clone"); err != nil {
		t.Fatalf("SetWallpapers devolvio error inesperado: %v", err)
	}
	if len(stub.daemons) != 1 || stub.daemons[0] != "awww-daemon" {
		t.Errorf("Se esperaba arrancar 'awww-daemon', se arranco %v", stub.daemons)
	}
	if !contains(stub.binaries(), "awww") {
		t.Errorf("Se esperaba ejecutar 'awww', se ejecutaron %v", stub.binaries())
	}
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		env     string
		monitor Monitor
		path    string
		index   int
		wantErr bool
		errMsg  string
	}{
		{
			env:     "test",
			monitor: Monitor{Name: "eDP-1"},
			path:    "/tmp/wallpaper.jpg",
			wantErr: true,
			errMsg:  "unsupported",
		},
		{
			env:     "feh",
			monitor: Monitor{Name: "DP-1"},
			path:    "/tmp/wallpaper.jpg",
			index:   0,
			wantErr: false,
		},
		{
			env:     "swww",
			monitor: Monitor{Name: "eDP-1"},
			path:    "/tmp/wallpaper.jpg",
			wantErr: false,
		},
		{
			env:     "nitrogen",
			monitor: Monitor{Name: "default"},
			path:    "/tmp/wallpaper.jpg",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			wc := NewWallpaperChanger(tt.env)
			cmd, err := wc.buildCommand(tt.monitor, tt.path, tt.index)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			} else if cmd == nil {
				t.Error("Expected command, got nil")
			}
		})
	}
}

func TestBuildCommand_FehIndex(t *testing.T) {
	wc := NewWallpaperChanger("feh")
	cmd, err := wc.buildCommand(Monitor{Name: "DP-1"}, "/tmp/wallpaper.jpg", 2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	args := cmd.Args
	hasDisplay := false
	for i, a := range args {
		if a == "--display" && i+1 < len(args) {
			if args[i+1] == ":0.2" {
				hasDisplay = true
			}
		}
	}
	if !hasDisplay {
		t.Errorf("Expected --display :0.2 in command args, got %v", args)
	}
}

func TestBuildCommand_SwayWithMonitor(t *testing.T) {
	wc := NewWallpaperChanger("sway")
	cmd, err := wc.buildCommand(Monitor{Name: "eDP-1"}, "/tmp/wp.jpg", 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	args := cmd.Args
	if len(args) < 5 || args[0] != "swaymsg" {
		t.Errorf("Expected swaymsg command, got %v", args)
	}
}

func TestBuildCommand_SwayDefaultMonitor(t *testing.T) {
	wc := NewWallpaperChanger("sway")
	cmd, err := wc.buildCommand(Monitor{Name: ""}, "/tmp/wp.jpg", 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	args := cmd.Args
	if len(args) < 3 || args[0] != "swaybg" {
		t.Errorf("Expected swaybg command when no monitor name, got %v", args)
	}
}

// applyToMonitor no delega en SetWallpapersFunc: lanza la IPC de dms y, si esta
// falla, cae a quickshell. Con runCommand sustituido se comprueba que se intenta
// la IPC y que su fallo no se propaga, sin tocar el compositor real.
func TestApplyToMonitor_DMS_Fallback(t *testing.T) {
	stub := &stubWallpaperExecution{runErr: errors.New("dms no disponible")}
	stub.install(t)

	wc := NewWallpaperChanger("dms")
	if err := wc.applyToMonitor(Monitor{Name: "eDP-1"}, "/tmp/wp.jpg", 0); err != nil {
		t.Errorf("applyToMonitor no deberia propagar el fallo de la IPC: %v", err)
	}
	if !contains(stub.binaries(), "dms") {
		t.Errorf("Se esperaba la IPC de dms, se ejecutaron %v", stub.binaries())
	}
}

func TestFilterXWaylandMonitors(t *testing.T) {
	tests := []struct {
		name     string
		input    []Monitor
		expected int
	}{
		{
			name: "Mixed monitors",
			input: []Monitor{
				{Name: "DP-1"},
				{Name: "HDMI-1"},
				{Name: "XWAYLAND0"},
			},
			expected: 2, // Should remove XWAYLAND0
		},
		{
			name: "Only real monitors",
			input: []Monitor{
				{Name: "eDP-1"},
			},
			expected: 1,
		},
		{
			name: "Only XWayland (fallback)",
			input: []Monitor{
				{Name: "XWAYLAND0"},
			},
			expected: 1, // Should keep it if it's the only one
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterXWaylandMonitors(tt.input)
			if len(result) != tt.expected {
				t.Errorf("Expected %d monitors, got %d", tt.expected, len(result))
			}
			for _, m := range result {
				if len(result) > 1 && strings.HasPrefix(m.Name, "XWAYLAND") {
					t.Errorf("Result contains XWAYLAND monitor when real monitors exist")
				}
			}
		})
	}
}
