package providers

import "gower/pkg/models"

// Provider defines the interface for a wallpaper provider.
type Provider interface {
	// GetName returns the name of the provider.
	GetName() string
	// Search searches for wallpapers based on the given query and options.
	Search(query string, options SearchOptions) ([]models.Wallpaper, error)
}

// SearchOptions defines the options for a wallpaper search.
type SearchOptions struct {
	MinWidth int
	Color    string
}
