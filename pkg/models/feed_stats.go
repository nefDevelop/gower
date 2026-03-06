package models

import "time"

// FeedStats represents statistics about the wallpaper feed.
type FeedStats struct {
	Total          int       `json:"total"`
	DarkCount      int       `json:"dark_count"`
	LightCount     int       `json:"light_count"`
	FavoritesCount int       `json:"favorites_count"`
	LastAdded      time.Time `json:"last_added"`
}
