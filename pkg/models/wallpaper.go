package models

// Wallpaper represents a wallpaper image and its metadata.
type Wallpaper struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Path      string `json:"path,omitempty"` // Local path, empty if not downloaded
	Source    string `json:"source"`

	// Metadata
	Category  string `json:"category,omitempty"`
	Dimension string `json:"dimension,omitempty"` // Resolution (e.g. 1920x1080)
	Ratio     string `json:"ratio,omitempty"`
	Theme     string `json:"theme,omitempty"` // Analyzed color theme
	Color     string `json:"color,omitempty"` // Dominant color hex
}
