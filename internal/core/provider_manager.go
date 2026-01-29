package core

import (
	"fmt"
	"gower/internal/providers"
)

// ProviderManager manages the available wallpaper providers.
type ProviderManager struct {
	providers map[string]providers.Provider
}

// NewProviderManager creates a new ProviderManager.
func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers: make(map[string]providers.Provider),
	}
}

// RegisterProvider registers a new provider.
func (pm *ProviderManager) RegisterProvider(provider providers.Provider) {
	pm.providers[provider.GetName()] = provider
}

// GetProvider retrieves a provider by name.
func (pm *ProviderManager) GetProvider(name string) (providers.Provider, error) {
	provider, ok := pm.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	return provider, nil
}

// GetProviders returns all registered providers.
func (pm *ProviderManager) GetProviders() []providers.Provider {
	var providerList []providers.Provider
	for _, provider := range pm.providers {
		providerList = append(providerList, provider)
	}
	return providerList
}

// GetProviderNames returns the names of all registered providers.
func (pm *ProviderManager) GetProviderNames() []string {
	var providerNames []string
	for name := range pm.providers {
		providerNames = append(providerNames, name)
	}
	return providerNames
}
