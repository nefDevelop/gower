package providers

import "gower/pkg/models"

// GenericProvider implements a provider based on configuration.
type GenericProvider struct {
	Config models.GenericProviderConfig
}

func (p *GenericProvider) GetName() string {
	return p.Config.Name
}

func (p *GenericProvider) Search(query string, opts SearchOptions) ([]models.Wallpaper, error) {
	// Placeholder for generic provider implementation
	return []models.Wallpaper{}, nil
}
