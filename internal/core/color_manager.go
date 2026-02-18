package core

import (
	"encoding/json"
	"fmt"
	"gower/internal/utils"
	"image"
	_ "image/gif" // Support GIF decoding
	"image/jpeg"
	_ "image/png" // Support PNG decoding
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return 0, 0, fmt.Errorf("failed to download image: %d", resp.StatusCode)
		}
		img, _, err = image.Decode(resp.Body)
	} else {
		file, errOpen := os.Open(src)
		if errOpen != nil {
			return 0, 0, errOpen
		}
		defer func() { _ = file.Close() }()
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
	defer func() { _ = out.Close() }()

	return bounds.Dx(), bounds.Dy(), jpeg.Encode(out, dst, &jpeg.Options{Quality: 80})
}

// GetImageDimensions returns the width and height of an image file without decoding the whole file.
func (cm *ColorManager) GetImageDimensions(path string) (int, int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = file.Close() }()

	cfg, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// AnalyzeColor determines the dominant color of the image.
func (cm *ColorManager) AnalyzeColor(path string) (string, error) {
	// 1. Try Matugen if available
	if _, err := exec.LookPath("matugen"); err == nil {
		if color, err := cm.analyzeColorMatugen(path); err == nil {
			return color, nil
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

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

	// Return the actual average color as a hex string
	return fmt.Sprintf("#%02X%02X%02X", r, g, b), nil
}

func (cm *ColorManager) analyzeColorMatugen(path string) (string, error) {
	cmd := exec.Command("matugen", "image", path, "--json")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Attempt to parse JSON output from matugen
	var result struct {
		Source      string `json:"source"`
		SourceColor string `json:"source_color"`
	}

	if err := json.Unmarshal(output, &result); err == nil {
		if result.Source != "" {
			return strings.ToUpper(result.Source), nil
		}
		if result.SourceColor != "" {
			return strings.ToUpper(result.SourceColor), nil
		}
	}

	return "", fmt.Errorf("could not extract color from matugen output")
}

// IsDark determines if a hex color is considered dark based on luminance.
func (cm *ColorManager) IsDark(hex string) bool {
	lum := cm.GetLuminance(hex)
	utils.Log.Debug("Color Analysis - Hex: %s, Luminance: %.2f (Dark: %v)", hex, lum, lum < 100)
	return lum < 100
}

// GetLuminance calculates the luminance of a hex color using Rec. 601 formula.
func (cm *ColorManager) GetLuminance(hex string) float64 {
	r, g, b := HexToRGB(hex)
	return 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
}

// FindNearestColorInPalette finds the closest color in a given dynamic palette.
func (cm *ColorManager) FindNearestColorInPalette(targetHex string, palette []string) string {
	if len(palette) == 0 {
		return targetHex
	}
	r, g, b := HexToRGB(targetHex)
	minDist := math.MaxFloat64
	nearest := palette[0]

	for _, hex := range palette {
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

// GenerateDynamicPalette uses K-Means clustering to find k representative colors.
func (cm *ColorManager) GenerateDynamicPalette(colors []string, k int) []string {
	if len(colors) == 0 {
		return []string{}
	}
	if len(colors) <= k {
		return colors
	}

	// Convert hex strings to RGB points
	type Point struct {
		R, G, B float64
	}
	var points []Point
	for _, c := range colors {
		r, g, b := HexToRGB(c)
		points = append(points, Point{float64(r), float64(g), float64(b)})
	}

	// Initialize Centroids (Randomly pick k points)
	centroids := make([]Point, k)
	perm := rand.Perm(len(points))
	for i := 0; i < k; i++ {
		centroids[i] = points[perm[i]]
	}

	// K-Means Iterations
	for iter := 0; iter < 10; iter++ {
		buckets := make([][]Point, k)
		for _, p := range points {
			minDist := math.MaxFloat64
			idx := 0
			for i, c := range centroids {
				dx := p.R - c.R
				dy := p.G - c.G
				dz := p.B - c.B
				dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
				if dist < minDist {
					minDist = dist
					idx = i
				}
			}
			buckets[idx] = append(buckets[idx], p)
		}
		for i := 0; i < k; i++ {
			if len(buckets[i]) > 0 {
				var sumR, sumG, sumB float64
				for _, p := range buckets[i] {
					sumR += p.R
					sumG += p.G
					sumB += p.B
				}
				count := float64(len(buckets[i]))
				centroids[i] = Point{R: sumR / count, G: sumG / count, B: sumB / count}
			}
		}
	}

	uniqueColors := make(map[string]bool)
	var result []string
	for _, c := range centroids {
		hex := fmt.Sprintf("#%02X%02X%02X", int(c.R), int(c.G), int(c.B))
		if !uniqueColors[hex] {
			uniqueColors[hex] = true
			result = append(result, hex)
		}
	}
	return result
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
