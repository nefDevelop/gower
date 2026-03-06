//go:build windows

package core

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

// setWallpaperWindows establece el fondo de pantalla en Windows usando una llamada al sistema.
func setWallpaperWindows(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("could not get absolute path for wallpaper: %w", err)
	}

	// La función SystemParametersInfoW requiere una cadena de texto codificada en UTF-16.
	pathPtr, err := syscall.UTF16PtrFromString(absPath)
	if err != nil {
		return fmt.Errorf("could not convert path to UTF-16: %w", err)
	}

	// Cargar user32.dll y obtener la dirección de la función.
	user32, err := syscall.LoadLibrary("user32.dll")
	if err != nil {
		return err
	}
	defer syscall.FreeLibrary(user32)

	proc, err := syscall.GetProcAddress(user32, "SystemParametersInfoW")
	if err != nil {
		return err
	}

	const (
		SPI_SETDESKWALLPAPER = 0x0014
		SPIF_UPDATEINIFILE   = 0x01
		SPIF_SENDCHANGE      = 0x02
	)

	// Llamar a la función del sistema.
	ret, _, sysErr := syscall.SyscallN(proc,
		uintptr(SPI_SETDESKWALLPAPER),
		0, // uiParam, no usado para esta acción
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(SPIF_UPDATEINIFILE|SPIF_SENDCHANGE))

	// Un valor de retorno distinto de cero indica éxito.
	if ret == 0 {
		return fmt.Errorf("SystemParametersInfoW call failed: %w", sysErr)
	}

	return nil
}
