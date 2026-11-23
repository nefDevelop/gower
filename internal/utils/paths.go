package utils

import (
	"os"
	"path/filepath"
)

// AppDir resuelve el directorio de datos de gower siguiendo XDG, con
// compatibilidad para la ubicación legacy ~/.gower.
func AppDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, errHome := os.UserHomeDir()
		if errHome != nil {
			return "", errHome
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	primaryDir := filepath.Join(configDir, "gower")

	if homeDir, err := os.UserHomeDir(); err == nil {
		legacyDir := filepath.Join(homeDir, ".gower")
		if _, err := os.Stat(filepath.Join(legacyDir, "config.json")); err == nil {
			if _, err := os.Stat(filepath.Join(primaryDir, "config.json")); os.IsNotExist(err) {
				return legacyDir, nil
			}
		}
	}
	return primaryDir, nil
}

// LogDir devuelve el directorio de logs dentro del directorio de datos, que es
// el mismo que usa la limpieza de retención del daemon.
func LogDir() (string, error) {
	dir, err := AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "logs"), nil
}
