package core

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"

	"gower/internal/utils"
)

const (
	maskWidth    = 480
	maskHeight   = 270
	maskClusters = 6
	maskIters    = 8
	// maskVersion invalida cachés antiguas cuando cambia el algoritmo.
	maskVersion = "v3"
)

// IsContentMask indica si la máscara se genera a partir del contenido de la
// imagen (en vez de ser una de las máscaras geométricas de roserlay).
func IsContentMask(kind string) bool {
	switch kind {
	case "color", "edges", "luminance", "hue", "saliency", "diff":
		return true
	default:
		return false
	}
}

type maskPixel struct {
	r, g, b float64
	luma    float64
}

// GenerateTransitionMaskFile genera (o reutiliza) una máscara en
// cache/masks y devuelve su ruta. Para kind="diff" se necesita oldPath.
func GenerateTransitionMaskFile(srcPath, kind, oldPath string) (string, error) {
	if !IsContentMask(kind) {
		return "", fmt.Errorf("unsupported content mask: %s", kind)
	}

	appDir, err := GetAppDir()
	if err != nil {
		return "", err
	}
	outDir := filepath.Join(appDir, "cache", "masks")

	key := utils.SafeFilename(filepath.Base(srcPath)) + "-" + kind
	if kind == "diff" {
		if oldPath == "" {
			return "", fmt.Errorf("diff mask requires the previous wallpaper")
		}
		key += "-" + utils.SafeFilename(filepath.Base(oldPath))
	}
	outPath := filepath.Join(outDir, key+"-"+maskVersion+".png")

	srcInfo, srcErr := os.Stat(srcPath)
	oldInfo, oldErr := os.Stat(oldPath)
	if srcErr == nil && (kind != "diff" || oldErr == nil) {
		if outInfo, err := os.Stat(outPath); err == nil && outInfo.Size() > 0 &&
			outInfo.ModTime().After(srcInfo.ModTime()) &&
			(kind != "diff" || outInfo.ModTime().After(oldInfo.ModTime())) {
			return outPath, nil
		}
	}

	if err := GenerateTransitionMask(srcPath, kind, oldPath, outPath); err != nil {
		return "", err
	}
	return outPath, nil
}

// GenerateTransitionMask escribe una máscara en escala de grises (negro =
// primero, blanco = último) para que roserlay la use con --mask-file.
func GenerateTransitionMask(srcPath, kind, oldPath, outPath string) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode %s: %w", srcPath, err)
	}

	points, gray, hue := sampleMaskPixels(img)
	mw, mh := maskWidth, maskHeight

	var oldPoints []maskPixel
	if kind == "diff" {
		if oldPath == "" {
			return fmt.Errorf("diff mask requires the previous wallpaper")
		}
		oldFile, err := os.Open(oldPath)
		if err != nil {
			return fmt.Errorf("failed to open previous wallpaper %s: %w", oldPath, err)
		}
		defer func() { _ = oldFile.Close() }()
		oldImg, _, err := image.Decode(oldFile)
		if err != nil {
			return fmt.Errorf("failed to decode previous wallpaper %s: %w", oldPath, err)
		}
		oldPoints, _, _ = sampleMaskPixels(oldImg)
	}

	var mask []uint8
	switch kind {
	case "luminance":
		mask = blurMask(gray, mw, mh, 3)
	case "hue":
		mask = blurMask(hue, mw, mh, 3)
	case "color":
		// k-means deja fronteras duras entre clusters: se difuminan para que
		// el revelado no parezca posterizado.
		mask = blurMask(clusterColorMask(points), mw, mh, 5)
	case "edges":
		mask = blurMask(edgeMask(gray, mw, mh), mw, mh, 2)
	case "saliency":
		mask = saliencyMask(gray, mw, mh)
	case "diff":
		mask = diffMask(points, oldPoints)
	default:
		return fmt.Errorf("unsupported content mask: %s", kind)
	}

	return writeGrayPNG(outPath, mw, mh, mask)
}

// sampleMaskPixels recorta al centro con aspecto 16:9 (igual que roserlay hace
// con el wallpaper) y muestrea a la resolución de la máscara.
func sampleMaskPixels(img image.Image) ([]maskPixel, []uint8, []uint8) {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil, nil, nil
	}

	// Recorte central a 16:9.
	targetAspect := float64(maskWidth) / float64(maskHeight)
	srcAspect := float64(srcW) / float64(srcH)
	cropW, cropH := srcW, srcH
	if srcAspect > targetAspect {
		cropW = int(float64(srcH) * targetAspect)
	} else {
		cropH = int(float64(srcW) / targetAspect)
	}
	if cropW < 1 {
		cropW = 1
	}
	if cropH < 1 {
		cropH = 1
	}
	offX := bounds.Min.X + (srcW-cropW)/2
	offY := bounds.Min.Y + (srcH-cropH)/2

	points := make([]maskPixel, 0, maskWidth*maskHeight)
	gray := make([]uint8, 0, maskWidth*maskHeight)
	hue := make([]uint8, 0, maskWidth*maskHeight)

	for y := 0; y < maskHeight; y++ {
		sy := offY + y*cropH/maskHeight
		for x := 0; x < maskWidth; x++ {
			sx := offX + x*cropW/maskWidth
			r16, g16, b16, _ := img.At(sx, sy).RGBA()
			r := float64(r16 >> 8)
			g := float64(g16 >> 8)
			b := float64(b16 >> 8)

			luma := 0.299*r + 0.587*g + 0.114*b
			points = append(points, maskPixel{r: r, g: g, b: b, luma: luma})
			gray = append(gray, uint8(luma))

			hue = append(hue, rgbToHue(uint8(r), uint8(g), uint8(b)))
		}
	}

	return points, gray, hue
}

// clusterColorMask agrupa los píxeles por color (k-means RGB) y ordena los
// clusters por luminancia: los oscuros se revelan primero y los claros al final.
func clusterColorMask(points []maskPixel) []uint8 {
	n := len(points)
	if n == 0 {
		return nil
	}

	k := maskClusters
	if n < k {
		k = n
	}

	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool {
		return points[order[a]].luma < points[order[b]].luma
	})

	centroids := make([]maskPixel, k)
	for i := 0; i < k; i++ {
		centroids[i] = points[order[i*n/k]]
	}

	assign := make([]int, n)
	for iter := 0; iter < maskIters; iter++ {
		sums := make([]maskPixel, k)
		counts := make([]int, k)

		for i, p := range points {
			best := 0
			bestDist := math.MaxFloat64
			for c, cen := range centroids {
				dr := p.r - cen.r
				dg := p.g - cen.g
				db := p.b - cen.b
				dist := dr*dr + dg*dg + db*db
				if dist < bestDist {
					bestDist = dist
					best = c
				}
			}
			assign[i] = best
			sums[best].r += p.r
			sums[best].g += p.g
			sums[best].b += p.b
			counts[best]++
		}

		for c := 0; c < k; c++ {
			if counts[c] == 0 {
				continue
			}
			inv := 1.0 / float64(counts[c])
			centroids[c] = maskPixel{
				r:    sums[c].r * inv,
				g:    sums[c].g * inv,
				b:    sums[c].b * inv,
				luma: (0.299*sums[c].r + 0.587*sums[c].g + 0.114*sums[c].b) * inv,
			}
		}
	}

	clusterOrder := make([]int, k)
	for i := range clusterOrder {
		clusterOrder[i] = i
	}
	sort.Slice(clusterOrder, func(a, b int) bool {
		return centroids[clusterOrder[a]].luma < centroids[clusterOrder[b]].luma
	})

	rank := make([]float64, k)
	for pos, c := range clusterOrder {
		if k == 1 {
			rank[c] = 0
		} else {
			rank[c] = float64(pos) / float64(k-1)
		}
	}

	mask := make([]uint8, n)
	for i, c := range assign {
		mask[i] = uint8(rank[c]*255.0 + 0.5)
	}
	return mask
}

func rgbToHue(r, g, b uint8) uint8 {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	if max == min {
		return 0
	}
	d := max - min

	var h float64
	switch max {
	case rf:
		h = math.Mod((gf-bf)/d, 6)
	case gf:
		h = (bf-rf)/d + 2
	default:
		h = (rf-gf)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return uint8(h/360.0*255.0 + 0.5)
}

// edgeMask aplica Sobel sobre la luminancia y normaliza la magnitud.
func edgeMask(gray []uint8, w, h int) []uint8 {
	mask := make([]uint8, len(gray))
	if w < 3 || h < 3 {
		return mask
	}

	magnitudes := make([]float64, len(gray))
	maxMag := 0.0
	at := func(x, y int) float64 { return float64(gray[y*w+x]) }

	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			gx := -at(x-1, y-1) + at(x+1, y-1) -
				2*at(x-1, y) + 2*at(x+1, y) -
				at(x-1, y+1) + at(x+1, y+1)
			gy := -at(x-1, y-1) - 2*at(x, y-1) - at(x+1, y-1) +
				at(x-1, y+1) + 2*at(x, y+1) + at(x+1, y+1)
			mag := math.Hypot(gx, gy)
			magnitudes[y*w+x] = mag
			if mag > maxMag {
				maxMag = mag
			}
		}
	}

	if maxMag <= 0 {
		return mask
	}
	for i, m := range magnitudes {
		if m <= 0 {
			continue
		}
		mask[i] = uint8(math.Sqrt(m/maxMag)*255.0 + 0.5)
	}
	return mask
}

// saliencyMask resalta zonas de alto contraste y centro de la imagen. El
// resultado se invierte para que el sujeto (alta saliencia) sea oscuro y se
// revele primero.
func saliencyMask(gray []uint8, w, h int) []uint8 {
	out := make([]uint8, len(gray))
	if w < 3 || h < 3 || len(gray) != w*h {
		return out
	}

	blurred := blurMask(gray, w, h, 8)
	vals := make([]float64, len(gray))
	maxVal := 0.0

	cx := float64(w-1) / 2
	cy := float64(h-1) / 2
	maxDist := math.Hypot(cx, cy)
	if maxDist == 0 {
		return out
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			contrast := math.Abs(float64(gray[i]) - float64(blurred[i]))
			dist := math.Hypot(float64(x)-cx, float64(y)-cy) / maxDist
			center := 1.0 - math.Min(dist, 1.0)
			v := contrast * (0.35 + 0.65*center)
			vals[i] = v
			if v > maxVal {
				maxVal = v
			}
		}
	}

	if maxVal <= 0 {
		return out
	}
	for i, v := range vals {
		s := v / maxVal
		out[i] = uint8((1.0-s)*255.0 + 0.5)
	}
	return blurMask(out, w, h, 3)
}

// diffMask compara luminancia y crominancia (YCbCr) del fondo viejo y el nuevo.
// Las zonas que más cambian quedan oscuras para revelarse primero.
func diffMask(a, b []maskPixel) []uint8 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	out := make([]uint8, n)
	if n == 0 {
		return out
	}

	vals := make([]float64, n)
	maxVal := 0.0
	for i := 0; i < n; i++ {
		y1, cb1, cr1 := color.RGBToYCbCr(uint8(a[i].r), uint8(a[i].g), uint8(a[i].b))
		y2, cb2, cr2 := color.RGBToYCbCr(uint8(b[i].r), uint8(b[i].g), uint8(b[i].b))

		dl := math.Abs(float64(y1) - float64(y2))
		dc := math.Hypot(float64(cb1)-float64(cb2), float64(cr1)-float64(cr2))
		d := 0.65*dl + 0.35*dc

		vals[i] = d
		if d > maxVal {
			maxVal = d
		}
	}

	if maxVal <= 0 {
		return out
	}
	for i, v := range vals {
		norm := math.Sqrt(v / maxVal)
		out[i] = uint8((1.0-norm)*255.0 + 0.5)
	}
	return blurMask(out, maskWidth, maskHeight, 3)
}

// blurMask aplica un box blur separable para suavizar las fronteras de la
// máscara. Una sola pasada horizontal + vertical basta para quitar el aspecto
// posterizado sin destruir la estructura.
func blurMask(src []uint8, w, h, radius int) []uint8 {
	if radius <= 0 || w <= 0 || h <= 0 || len(src) != w*h {
		return src
	}

	clamp := func(v, lo, hi int) int {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}

	window := 2*radius + 1
	tmp := make([]uint8, len(src))
	out := make([]uint8, len(src))

	for y := 0; y < h; y++ {
		row := y * w
		sum := 0
		for x := -radius; x <= radius; x++ {
			sum += int(src[row+clamp(x, 0, w-1)])
		}
		for x := 0; x < w; x++ {
			tmp[row+x] = uint8(sum / window)
			sum += int(src[row+clamp(x+radius+1, 0, w-1)])
			sum -= int(src[row+clamp(x-radius, 0, w-1)])
		}
	}

	for x := 0; x < w; x++ {
		sum := 0
		for y := -radius; y <= radius; y++ {
			sum += int(tmp[clamp(y, 0, h-1)*w+x])
		}
		for y := 0; y < h; y++ {
			out[y*w+x] = uint8(sum / window)
			sum += int(tmp[clamp(y+radius+1, 0, h-1)*w+x])
			sum -= int(tmp[clamp(y-radius, 0, h-1)*w+x])
		}
	}

	return out
}

func writeGrayPNG(path string, w, h int, pix []uint8) error {
	if len(pix) != w*h {
		return fmt.Errorf("mask size mismatch: got %d, want %d", len(pix), w*h)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	img := image.NewGray(image.Rect(0, 0, w, h))
	copy(img.Pix, pix)

	tmp, err := os.CreateTemp(filepath.Dir(path), ".mask-*.png")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := png.Encode(tmp, img); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
