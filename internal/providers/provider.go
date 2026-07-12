package providers

import (
	"context"

	"gower/pkg/models"
)

// SearchOptions defines the criteria for searching wallpapers.
type SearchOptions struct {
	Limit       int
	Page        int
	Sort        string
	Category    string
	MinWidth    int
	MinHeight   int
	AspectRatio string
	Color       string
	ForceUpdate bool
	ExcludeIDs  map[string]bool
}

// Provider defines the interface for wallpaper providers.
type Provider interface {
	GetName() string
	Search(ctx context.Context, query string, options SearchOptions) ([]models.Wallpaper, error)
}
