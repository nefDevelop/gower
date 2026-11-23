package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"image"
	_ "image/gif" // Support GIF decoding
	"image/jpeg"
	_ "image/png" // Support PNG decoding
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gower/internal/utils"
)

const (
	thumbnailWidth = 300
	maxImagePixels = 60_000_000 // ~60 MP: evita bombas de descompresión
)

// ColorManager handles image processing tasks such as thumbnail generation and color analysis.
type ColorManager struct {
	Log *utils.Logger
}

// NewColorManager creates a new instance of ColorManager.
func NewColorManager() *ColorManager {
	return &ColorManager{}
}

var (
	matugenOnce sync.Once
	matugenPath string
)

func findMatugen() string {
	matugenOnce.Do(func() {
		if path, err := exec.LookPath("matugen"); err == nil {
			matugenPath = path
		}
	})
	return matugenPath
}

// decodeImage lee la imagen con un límite de tamaño y rechaza imágenes
// desproporcionadas antes de decodificarlas completas.
func decodeImage(r io.Reader) (image.Image, error) {
	data, err := utils.ReadBodyLimit(r, utils.MaxImageBytes)
	if err != nil {
		return nil, err
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image config: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, fmt.Errorf("invalid image dimensions: %dx%d", cfg.Width, cfg.Height)
	}
	if cfg.Width*cfg.Height > maxImagePixels {
		return nil, fmt.Errorf("image too large: %dx%d", cfg.Width, cfg.Height)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	if img == nil {
		return nil, fmt.Errorf("image.Decode returned nil image")
	}
	return img, nil
}

// GenerateThumbnail creates a thumbnail for the specified image file.
// If src is a URL, it downloads it first.
func (cm *ColorManager) GenerateThumbnail(src, destPath string) (int, int, error) {
	var img image.Image
	var err error

	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		var resp *http.Response
		for attempt := 0; attempt < 3; attempt++ {
			req, reqErr := http.NewRequestWithContext(context.Background(), http.MethodGet, src, nil)
			if reqErr != nil {
				return 0, 0, reqErr
			}
			resp, err = utils.DefaultHTTPClient.Do(req) //nolint:bodyclose // cerrado con utils.CloseResponse
			if err == nil {
				break
			}
			if attempt < 2 {
				time.Sleep(500 * time.Millisecond)
			}
		}
		if err != nil {
			return 0, 0, err
		}
		defer utils.CloseResponse(resp)
		if resp.StatusCode != http.StatusOK {
			return 0, 0, fmt.Errorf("failed to download image: %d", resp.StatusCode)
		}
		img, err = decodeImage(resp.Body)
	} else {
		file, errOpen := os.Open(src)
		if errOpen != nil {
			return 0, 0, errOpen
		}
		defer func() { _ = file.Close() }()
		img, err = decodeImage(file)
	}
	if err != nil {
		return 0, 0, err
	}

	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, 0, fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return 0, 0, fmt.Errorf("invalid image bounds for %s", src)
	}

	width := thumbnailWidth
	if bounds.Dx() < width {
		width = bounds.Dx()
	}
	height := int(float64(width) * float64(bounds.Dy()) / float64(bounds.Dx()))
	if height <= 0 {
		height = 1
	}

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
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, fmt.Errorf("invalid image dimensions: %dx%d", cfg.Width, cfg.Height)
	}
	return cfg.Width, cfg.Height, nil
}

// AnalyzeColor determines the dominant color of the image. Se muestrea la
// imagen en una rejilla en vez de recorrer todos los píxeles: en una imagen 4K
// el recorrido completo son millones de llamadas a image.Image.At.
func (cm *ColorManager) AnalyzeColor(path string) (string, error) {
	if matugen := findMatugen(); matugen != "" {
		if color, err := cm.analyzeColorMatugen(matugen, path); err == nil {
			return color, nil
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	img, err := decodeImage(file)
	if err != nil {
		return "", err
	}

	bounds := img.Bounds()
	dx, dy := bounds.Dx(), bounds.Dy()
	if dx <= 0 || dy <= 0 {
		return "", fmt.Errorf("invalid image bounds")
	}

	stepX := dx / 64
	if stepX < 1 {
		stepX = 1
	}
	stepY := dy / 64
	if stepY < 1 {
		stepY = 1
	}

	var r, g, b, count uint64
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
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

	r = (r / count) >> 8
	g = (g / count) >> 8
	b = (b / count) >> 8

	return fmt.Sprintf("#%02X%02X%02X", r, g, b), nil
}

func (cm *ColorManager) analyzeColorMatugen(matugen, path string) (string, error) {
	cmd := exec.Command(matugen, "image", path, "--json")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

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
	return cm.GetLuminance(hex) < 100
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
		dist := math.Sqrt(math.Pow(float64(r-pr), 2) + math.Pow(float64(g-pg), 2) + math.Pow(float64(b-pb), 2))
		if dist < minDist {
			minDist = dist
			nearest = hex
		}
	}
	return nearest
}

// GenerateDynamicPalette uses K-Means clustering to find k representative colors.
// El orden inicial es determinista (hash de los colores) para que el mismo feed
// produzca siempre la misma paleta.
func (cm *ColorManager) GenerateDynamicPalette(colors []string, k int) []string {
	if len(colors) == 0 || k <= 0 {
		return []string{}
	}
	if len(colors) <= k {
		return colors
	}

	type point struct {
		R, G, B float64
	}
	points := make([]point, 0, len(colors))
	for _, c := range colors {
		r, g, b := HexToRGB(c)
		points = append(points, point{float64(r), float64(g), float64(b)})
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.Join(colors, ",")))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))

	centroids := make([]point, k)
	perm := rng.Perm(len(points))
	for i := 0; i < k; i++ {
		centroids[i] = points[perm[i]]
	}

	for iter := 0; iter < 10; iter++ {
		buckets := make([][]point, k)
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
			if len(buckets[i]) == 0 {
				// Reinicializar un centroide vacío; si no, conserva un valor
				// viejo y la paleta puede acabar con menos de k colores.
				centroids[i] = points[(iter+i)%len(points)]
				continue
			}
			var sumR, sumG, sumB float64
			for _, p := range buckets[i] {
				sumR += p.R
				sumG += p.G
				sumB += p.B
			}
			count := float64(len(buckets[i]))
			centroids[i] = point{R: sumR / count, G: sumG / count, B: sumB / count}
		}
	}

	uniqueColors := make(map[string]bool)
	result := make([]string, 0, k)
	for _, c := range centroids {
		hex := fmt.Sprintf("#%02X%02X%02X", int(c.R), int(c.G), int(c.B))
		if !uniqueColors[hex] {
			uniqueColors[hex] = true
			result = append(result, hex)
		}
	}
	return result
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
