// internal/core/storage_manager.go
package core

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type StorageManager struct {
	configPath string
}

func NewStorageManager() *StorageManager {
	// Configurar viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Añadir rutas de búsqueda (alineado con cmd/config.go)
	configDir, _ := os.UserConfigDir()
	viper.AddConfigPath(filepath.Join(configDir, "gower"))
	viper.AddConfigPath(".")

	// Valores por defecto
	viper.SetDefault("providers.wallhaven.enabled", true)
	viper.SetDefault("providers.wallhaven.api_key", "")
	viper.SetDefault("search.min_width", 1920)
	viper.SetDefault("search.min_height", 1080)
	viper.SetDefault("behavior.change_interval", 30)
	viper.SetDefault("behavior.theme", "dark")
	viper.SetDefault("behavior.multi_monitor", "clone")

	// Intentar leer la configuración existente
	viper.ReadInConfig()

	return &StorageManager{
		configPath: getConfigPath(),
	}
}

func (sm *StorageManager) SaveConfig(key string, value interface{}) error {
	viper.Set(key, value)
	return viper.WriteConfig()
}

func (sm *StorageManager) GetConfig(key string) interface{} {
	return viper.Get(key)
}

func (sm *StorageManager) GetConfigString(key string) string {
	return viper.GetString(key)
}

func (sm *StorageManager) GetConfigInt(key string) int {
	return viper.GetInt(key)
}

func getConfigPath() string {
	if path := viper.ConfigFileUsed(); path != "" {
		return path
	}
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "gower", "config.yaml")
}
