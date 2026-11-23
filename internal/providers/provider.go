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

// normalizeLimit acota un límite de resultados a un rango válido.
func normalizeLimit(limit, fallback, max int) int {
	if limit <= 0 {
		limit = fallback
	}
	if max > 0 && limit > max {
		limit = max
	}
	return limit
}

// waitRateLimiter espera respetando el contexto de la petición.
func waitRateLimiter(rl interface{ Wait(context.Context) error }, ctx context.Context) error {
	if rl == nil {
		return nil
	}
	return rl.Wait(ctx)
}
