package providers

import (
	"fmt"
	"gower/pkg/models"
)

// WallhavenProvider is the provider for wallhaven.cc.
type WallhavenProvider struct {}

// GetName returns the name of the provider.
func (p *WallhavenProvider) GetName() string {
	return "wallhaven"
}

// Search searches for wallpapers on wallhaven.cc.
func (p *WallhavenProvider) Search(query string, options SearchOptions) ([]models.Wallpaper, error) {
	fmt.Printf("Searching wallhaven for: %s\n", query)
	fmt.Printf("Options: %+v\n", options)

	// This is a dummy implementation.
	// In a real implementation, we would make an HTTP request to the wallhaven API.
	return []models.Wallpaper{
		{
			ID:        "wh-123",
			URL:       "https://wallhaven.cc/w/123",
			Source:    "wallhaven",
			Dimension: "1920x1080",
		},
		{
			ID:        "wh-456",
			URL:       "https://wallhaven.cc/w/456",
			Source:    "wallhaven",
			Dimension: "3840x2160",
		},
	}, nil
}
