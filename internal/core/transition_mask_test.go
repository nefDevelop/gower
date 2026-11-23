package core

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeMaskTestImage crea una imagen mitad roja / mitad azul con una línea
// blanca vertical en el centro, útil para probar color, bordes, luminancia y hue.
func writeMaskTestImage(t *testing.T, path string) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 64, 36))
	for y := 0; y < 36; y++ {
		for x := 0; x < 64; x++ {
			c := color.RGBA{R: 220, G: 30, B: 30, A: 255}
			if x >= 32 {
				c = color.RGBA{R: 30, G: 30, B: 220, A: 255}
			}
			if x == 31 || x == 32 {
				c = color.RGBA{R: 255, G: 255, B: 255, A: 255}
			}
			img.Set(x, y, c)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateTransitionMask(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.png")
	writeMaskTestImage(t, src)

	for _, kind := range []string{"luminance", "hue", "color", "edges", "saliency"} {
		t.Run(kind, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), kind+".png")
			if err := GenerateTransitionMask(src, kind, "", out); err != nil {
				t.Fatalf("GenerateTransitionMask(%s): %v", kind, err)
			}

			f, err := os.Open(out)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Fatalf("decode mask: %v", err)
			}
			b := img.Bounds()
			if b.Dx() != maskWidth || b.Dy() != maskHeight {
				t.Fatalf("mask size = %dx%d, want %dx%d", b.Dx(), b.Dy(), maskWidth, maskHeight)
			}

			left := color.GrayModel.Convert(img.At(40, 135)).(color.Gray).Y
			right := color.GrayModel.Convert(img.At(maskWidth-40, 135)).(color.Gray).Y

			switch kind {
			case "color":
				if left == right {
					t.Errorf("color mask no distingue las dos mitades: %d == %d", left, right)
				}
			case "luminance":
				if left == right {
					t.Errorf("luminance mask no distingue rojo/azul: %d == %d", left, right)
				}
			case "hue":
				if left == right {
					t.Errorf("hue mask no distingue rojo/azul: %d == %d", left, right)
				}
			case "edges":
				gray, ok := img.(*image.Gray)
				if !ok {
					t.Fatalf("edges mask is not grayscale: %T", img)
				}
				maxVal := uint8(0)
				for _, v := range gray.Pix {
					if v > maxVal {
						maxVal = v
					}
				}
				if maxVal == 0 {
					t.Error("edges mask no detectó ningún borde")
				}
			case "saliency":
				// El centro tiene la línea blanca de alto contraste; debe ser
				// más oscuro (se revela antes) que una esquina plana.
				center := color.GrayModel.Convert(img.At(maskWidth/2, 135)).(color.Gray).Y
				corner := color.GrayModel.Convert(img.At(10, 10)).(color.Gray).Y
				if center >= corner {
					t.Errorf("saliency mask no prioriza el centro: centro=%d esquina=%d", center, corner)
				}
			}
		})
	}
}

func TestGenerateTransitionMaskFileUsesCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	src := filepath.Join(t.TempDir(), "src.png")
	writeMaskTestImage(t, src)

	path, err := GenerateTransitionMaskFile(src, "color", "")
	if err != nil {
		t.Fatalf("GenerateTransitionMaskFile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("mask not created: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("mask path should be absolute, got %q", path)
	}

	again, err := GenerateTransitionMaskFile(src, "color", "")
	if err != nil {
		t.Fatalf("second GenerateTransitionMaskFile: %v", err)
	}
	if again != path {
		t.Errorf("cache path changed: %q vs %q", path, again)
	}
}

// writeDiffTestImage crea una imagen con la mitad izquierda igual que
// writeMaskTestImage (rojo) y la derecha en verde: solo cambia la derecha.
func writeDiffTestImage(t *testing.T, path string) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 64, 36))
	for y := 0; y < 36; y++ {
		for x := 0; x < 64; x++ {
			c := color.RGBA{R: 220, G: 30, B: 30, A: 255}
			if x >= 32 {
				c = color.RGBA{R: 30, G: 220, B: 30, A: 255}
			}
			img.Set(x, y, c)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateTransitionMaskDiff(t *testing.T) {
	oldPath := filepath.Join(t.TempDir(), "old.png")
	newPath := filepath.Join(t.TempDir(), "new.png")
	writeMaskTestImage(t, oldPath)
	writeDiffTestImage(t, newPath)

	out := filepath.Join(t.TempDir(), "diff.png")
	if err := GenerateTransitionMask(newPath, "diff", oldPath, out); err != nil {
		t.Fatalf("GenerateTransitionMask(diff): %v", err)
	}

	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode diff mask: %v", err)
	}

	unchanged := color.GrayModel.Convert(img.At(40, 135)).(color.Gray).Y
	changed := color.GrayModel.Convert(img.At(maskWidth-40, 135)).(color.Gray).Y
	if unchanged <= changed {
		t.Errorf("diff mask deberia revelar antes lo cambiado: unchanged=%d changed=%d", unchanged, changed)
	}
}

func TestGenerateTransitionMaskRejectsUnknownKind(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src.png")
	writeMaskTestImage(t, src)

	if err := GenerateTransitionMask(src, "magia", "", filepath.Join(t.TempDir(), "x.png")); err == nil {
		t.Error("expected error for unknown mask kind")
	}
}
