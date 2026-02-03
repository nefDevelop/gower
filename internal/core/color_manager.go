package core

import (
	"fmt"
	"image"
	_ "image/gif" // Support GIF decoding
	"image/jpeg"
	_ "image/png" // Support PNG decoding
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// ColorManager handles image processing tasks such as thumbnail generation and color analysis.
type ColorManager struct{}

// StandardPalette defines a set of base colors for quantization.
var StandardPalette = []string{
	"#000000", // Black
	"#FFFFFF", // White
	"#808080", // Gray
	"#FF0000", // Red
	"#00FF00", // Green
	"#0000FF", // Blue
	"#FFFF00", // Yellow
	"#00FFFF", // Cyan
	"#FF00FF", // Magenta
	"#FFA500", // Orange
	"#800080", // Purple
	"#A52A2A", // Brown
	"#FFC0CB", // Pink
	"#008080", // Teal
	"#000080", // Navy
	"#800000", // Maroon
}

// NewColorManager creates a new instance of ColorManager.
func NewColorManager() *ColorManager {
	return &ColorManager{}
}

// GenerateThumbnail creates a thumbnail for the specified image file.
// If src is a URL, it downloads it first.
func (cm *ColorManager) GenerateThumbnail(src, destPath string) (int, int, error) {
	var img image.Image
	var err error

	// Check if src is URL or local path
	if len(src) > 4 && src[:4] == "http" {
		client := &http.Client{Timeout: 15 * time.Second}
		var resp *http.Response
		var errGet error

		// Retry up to 3 times for network glitches
		for i := 0; i < 3; i++ {
			resp, errGet = client.Get(src)
			if errGet == nil {
				break
			}
			time.Sleep(1 * time.Second)
		}

		if errGet != nil {
			return 0, 0, errGet
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return 0, 0, fmt.Errorf("failed to download image: %d", resp.StatusCode)
		}
		img, _, err = image.Decode(resp.Body)
	} else {
		file, errOpen := os.Open(src)
		if errOpen != nil {
			return 0, 0, errOpen
		}
		defer file.Close()
		img, _, err = image.Decode(file)
	}

	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}
	if img == nil {
		return 0, 0, fmt.Errorf("image.Decode returned nil image for %s", src) // Explicit check
	}

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, 0, fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	// Simple resizing (Nearest Neighbor equivalent for standard lib)
	bounds := img.Bounds()
	width := 300
	ratio := float64(bounds.Dy()) / float64(bounds.Dx())
	height := int(float64(width) * ratio)

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x * bounds.Dx() / width
			srcY := y * bounds.Dy() / height
			dst.Set(x, y, img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}

	out, err := os.Create(destPath)
	if err != nil {
		return 0, 0, err
	}
	defer out.Close()

	return bounds.Dx(), bounds.Dy(), jpeg.Encode(out, dst, &jpeg.Options{Quality: 80})
}

// GetImageDimensions returns the width and height of an image file without decoding the whole file.
func (cm *ColorManager) GetImageDimensions(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// AnalyzeColor determines the dominant color of the image.
func (cm *ColorManager) AnalyzeColor(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}

	// Calculate average color
	bounds := img.Bounds()
	var r, g, b, count uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pr, pg, pb, _ := img.At(x, y).RGBA()
			r += uint64(pr)
			g += uint64(pg)
			b += uint64(pb)
			count++
		}
	}

	if count == 0 {
		return "#000000", nil
	}

	// RGBA returns 0-65535, scale down to 0-255
	r = (r / count) >> 8
	g = (g / count) >> 8
	b = (b / count) >> 8

	// Find nearest color in palette
	return FindNearestColor(int(r), int(g), int(b)), nil
}

func FindNearestColor(r, g, b int) string {
	minDist := math.MaxFloat64
	nearest := "#000000"

	for _, hex := range StandardPalette {
		pr, pg, pb := HexToRGB(hex)
		// Euclidean distance
		dist := math.Sqrt(math.Pow(float64(r-pr), 2) + math.Pow(float64(g-pg), 2) + math.Pow(float64(b-pb), 2))
		if dist < minDist {
			minDist = dist
			nearest = hex
		}
	}
	return nearest
}

func HexToRGB(hex string) (int, int, int) {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 0, 0, 0
	}
	val, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, 0, 0
	}
	return int(val >> 16), int((val >> 8) & 0xFF), int(val & 0xFF)
}

// UpdateIndex adds the color to the search index.
// Currently a placeholder.
func (cm *ColorManager) UpdateIndex(hexColor string) error {
	// TODO: Implement indexing logic.
	return nil
}
