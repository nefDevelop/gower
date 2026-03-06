package core

import (
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func createDummyImageWithColor(t *testing.T, path string, width, height int, c color.Color) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create dummy image: %v", err)
	}
	defer func() { _ = f.Close() }()

	if err := png.Encode(f, img); err != nil {
		t.Fatalf("Failed to encode dummy image: %v", err)
	}
}

func TestHexToRGB(t *testing.T) {
	r, g, b := HexToRGB("#FF0000")
	if r != 255 || g != 0 || b != 0 {
		t.Errorf("Expected (255, 0, 0), got (%d, %d, %d)", r, g, b)
	}

	r, g, b = HexToRGB("00FF00")
	if r != 0 || g != 255 || b != 0 {
		t.Errorf("Expected (0, 255, 0), got (%d, %d, %d)", r, g, b)
	}
}

func TestColorManager_FindNearestColorInPalette(t *testing.T) {
	cm := NewColorManager()
	palette := []string{"#FF0000", "#0000FF", "#00FF00"}

	// Test with a color that is in the palette
	nearest := cm.FindNearestColorInPalette("#FF0000", palette)
	if nearest != "#FF0000" {
		t.Errorf("Expected #FF0000, got %s", nearest)
	}

	// Test with a color that is close to a palette color (dark red)
	nearest = cm.FindNearestColorInPalette("#E01010", palette)
	if nearest != "#FF0000" {
		t.Errorf("Expected #FF0000 for dark red, got %s", nearest)
	}
}

func TestColorManager_GenerateThumbnail_Local(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gower-colormanager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	srcPath := filepath.Join(tmpDir, "src.png")
	destPath := filepath.Join(tmpDir, "thumb.jpg")
	createDummyImageWithColor(t, srcPath, 600, 400, color.RGBA{R: 255, A: 255})

	cm := NewColorManager()
	w, h, err := cm.GenerateThumbnail(srcPath, destPath)
	if err != nil {
		t.Fatalf("GenerateThumbnail failed: %v", err)
	}

	if w != 600 || h != 400 {
		t.Errorf("Expected original dimensions (600, 400), got (%d, %d)", w, h)
	}

	thumb, err := os.Open(destPath)
	if err != nil {
		t.Fatalf("Failed to open thumbnail: %v", err)
	}
	defer func() { _ = thumb.Close() }()

	img, _, err := image.DecodeConfig(thumb)
	if err != nil {
		t.Fatalf("Failed to decode thumbnail config: %v", err)
	}

	if img.Width != 300 {
		t.Errorf("Expected thumbnail width 300, got %d", img.Width)
	}
}

func TestColorManager_GenerateThumbnail_URL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gower-colormanager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a dummy image and serve it
	srcPath := filepath.Join(tmpDir, "src.png")
	createDummyImageWithColor(t, srcPath, 600, 400, color.RGBA{R: 255, A: 255})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, srcPath)
	}))
	defer server.Close()

	destPath := filepath.Join(tmpDir, "thumb.jpg")
	cm := NewColorManager()
	_, _, err = cm.GenerateThumbnail(server.URL, destPath)
	if err != nil {
		t.Fatalf("GenerateThumbnail with URL failed: %v", err)
	}

	_, err = os.Stat(destPath)
	if os.IsNotExist(err) {
		t.Errorf("Thumbnail was not created from URL")
	}
}

func TestColorManager_AnalyzeColor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gower-colormanager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	imgPath := filepath.Join(tmpDir, "red.png")
	createDummyImageWithColor(t, imgPath, 100, 100, color.RGBA{R: 255, A: 255})

	cm := NewColorManager()
	hex, err := cm.AnalyzeColor(imgPath)
	if err != nil {
		t.Fatalf("AnalyzeColor failed: %v", err)
	}

	if hex != "#FF0000" {
		t.Errorf("Expected dominant color #FF0000, got %s", hex)
	}
}

func TestColorManager_GetLuminance(t *testing.T) {
	cm := NewColorManager()

	// White
	lum := cm.GetLuminance("#FFFFFF")
	if lum < 254.0 {
		t.Errorf("Expected luminance close to 255 for white, got %f", lum)
	}

	// Black
	lum = cm.GetLuminance("#000000")
	if lum > 1.0 {
		t.Errorf("Expected luminance close to 0 for black, got %f", lum)
	}

	// Red (0.299 * 255 = 76.245)
	lum = cm.GetLuminance("#FF0000")
	if lum < 76.0 || lum > 77.0 {
		t.Errorf("Expected luminance around 76.245 for red, got %f", lum)
	}
}
