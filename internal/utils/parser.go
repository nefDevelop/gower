// internal/utils/parser.go
package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseResolution acepta resoluciones de la forma "1920x1080".
func ParseResolution(res string) (width, height int, err error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(res)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid resolution format")
	}
	width, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil {
		return 0, 0, fmt.Errorf("invalid resolution values")
	}
	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("resolution values must be positive")
	}
	return width, height, nil
}

// ParseAspectRatio acepta una proporción "16:9" y devuelve su valor decimal.
func ParseAspectRatio(ratio string) (float64, error) {
	parts := strings.Split(strings.TrimSpace(ratio), ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid aspect ratio")
	}
	wInt, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	hInt, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil {
		return 0, fmt.Errorf("invalid aspect ratio values")
	}
	if wInt <= 0 || hInt <= 0 {
		return 0, fmt.Errorf("aspect ratio values must be positive")
	}
	return float64(wInt) / float64(hInt), nil
}
