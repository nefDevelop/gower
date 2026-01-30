package core

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // Support PNG decoding
	"net/http"
	"os"
	"path/filepath"
)

// ColorManager handles image processing tasks such as thumbnail generation and color analysis.
type ColorManager struct{}

// NewColorManager creates a new instance of ColorManager.
func NewColorManager() *ColorManager {
	return &ColorManager{}
}

// GenerateThumbnail creates a thumbnail for the specified image file.
// If src is a URL, it downloads it first.
func (cm *ColorManager) GenerateThumbnail(src, destPath string) error {
	var img image.Image
	var err error

	// Check if src is URL or local path
	if len(src) > 4 && src[:4] == "http" {
		resp, err := http.Get(src)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download image: %d", resp.StatusCode)
		}
		img, _, err = image.Decode(resp.Body)
	} else {
		file, err := os.Open(src)
		if err != nil {
			return err
		}
		defer file.Close()
		img, _, err = image.Decode(file)
	}

	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create thumbnail directory: %w", err)
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
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, dst, &jpeg.Options{Quality: 80})
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

	return fmt.Sprintf("#%02x%02x%02x", uint8(r), uint8(g), uint8(b)), nil
}

// UpdateIndex adds the color to the search index.
// Currently a placeholder.
func (cm *ColorManager) UpdateIndex(hexColor string) error {
	// TODO: Implement indexing logic.
	return nil
}
