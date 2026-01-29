package models

// Wallpaper represents a wallpaper's metadata.
type Wallpaper struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Path      string `json:"path"` // Local path after download
	Purity    string `json:"purity"`
	Category  string `json:"category"`
	Dimension string `json:"dimension"`
	Resolution string `json:"resolution"` // Added for clarity
	Ratio     string `json:"ratio"`
	Source    string `json:"source"` // e.g., wallhaven, reddit, nasa
	Theme     string `json:"theme"`  // e.g., dark, light
	// Add other relevant fields like tags, colors, uploader, etc.
}
