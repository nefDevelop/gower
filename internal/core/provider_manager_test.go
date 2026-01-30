package core

import (
	"gower/pkg/models"
	"testing"
)

// MockProvider is a mock implementation of the Provider interface for testing.
type MockProvider struct {
	name string
}

func (m *MockProvider) GetName() string {
	return m.name
}

func (m *MockProvider) GetWallpapers(options map[string]string) ([]*models.Wallpaper, error) {
	return nil, nil
}

func (m *MockProvider) GetCategories() []string {
	return nil
}

func TestProviderManager(t *testing.T) {
	pm := NewProviderManager()
	mockProvider1 := &MockProvider{name: "mock1"}
	mockProvider2 := &MockProvider{name: "mock2"}

	// Test RegisterProvider
	pm.RegisterProvider(mockProvider1)
	pm.RegisterProvider(mockProvider2)

	// Test GetProvider
	p, err := pm.GetProvider("mock1")
	if err != nil {
		t.Errorf("GetProvider failed for mock1: %v", err)
	}
	if p.GetName() != "mock1" {
		t.Errorf("Expected provider mock1, got %s", p.GetName())
	}

	_, err = pm.GetProvider("nonexistent")
	if err == nil {
		t.Errorf("Expected error for nonexistent provider, but got nil")
	}

	// Test GetProviders
	allProviders := pm.GetProviders()
	if len(allProviders) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(allProviders))
	}

	// Test GetProviderNames
	names := pm.GetProviderNames()
	if len(names) != 2 {
		t.Errorf("Expected 2 provider names, got %d", len(names))
	}

	// Check if the names are correct (order is not guaranteed)
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}
	if !nameMap["mock1"] || !nameMap["mock2"] {
		t.Errorf("Provider names are incorrect: %v", names)
	}
}
