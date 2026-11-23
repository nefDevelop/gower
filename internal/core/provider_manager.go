package core

import (
	"fmt"
	"sync"

	"gower/internal/providers"
)

// ProviderManager manages the available wallpaper providers.
type ProviderManager struct {
	mu        sync.RWMutex
	providers map[string]providers.Provider
	order     []string
}

// NewProviderManager creates a new ProviderManager.
func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers: make(map[string]providers.Provider),
	}
}

// RegisterProvider registers a new provider preserving registration order.
func (pm *ProviderManager) RegisterProvider(provider providers.Provider) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	name := provider.GetName()
	if _, exists := pm.providers[name]; !exists {
		pm.order = append(pm.order, name)
	}
	pm.providers[name] = provider
}

// GetProvider retrieves a provider by name.
func (pm *ProviderManager) GetProvider(name string) (providers.Provider, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	provider, ok := pm.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// GetProviders returns all registered providers in registration order.
func (pm *ProviderManager) GetProviders() []providers.Provider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	list := make([]providers.Provider, 0, len(pm.order))
	for _, name := range pm.order {
		list = append(list, pm.providers[name])
	}
	return list
}

// GetProviderNames returns the names of all registered providers in
// registration order.
func (pm *ProviderManager) GetProviderNames() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, len(pm.order))
	copy(names, pm.order)
	return names
}
