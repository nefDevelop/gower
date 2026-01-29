// internal/utils/parser.go
package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseResolution(res string) (width, height int, err error) {
	parts := strings.Split(res, "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid resolution format")
	}
	var errW, errH error
	width, errW = strconv.Atoi(parts[0])
	height, errH = strconv.Atoi(parts[1])
	if errW != nil || errH != nil {
		return 0, 0, fmt.Errorf("invalid resolution values")
	}
	return width, height, nil
}

func ParseAspectRatio(ratio string) (float64, error) {
	parts := strings.Split(ratio, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid aspect ratio")
	}
	wInt, errW := strconv.Atoi(parts[0])
	hInt, errH := strconv.Atoi(parts[1])
	if errW != nil || errH != nil {
		return 0, fmt.Errorf("invalid aspect ratio values")
	}
	w := float64(wInt)
	h := float64(hInt)
	return w / h, nil
}

func BuildSearchQuery(flags map[string]interface{}) string {
	var query []string

	if minWidth, ok := flags["min-width"].(int); ok && minWidth > 0 {
		query = append(query, fmt.Sprintf("min_width=%d", minWidth))
	}

	if aspect, ok := flags["aspect-ratio"].(string); ok && aspect != "" {
		query = append(query, fmt.Sprintf("aspect_ratio=%s", aspect))
	}

	return strings.Join(query, "&")
}
