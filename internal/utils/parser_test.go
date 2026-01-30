package utils

import (
	"strings"
	"testing"
)

func TestParseResolution(t *testing.T) {
	tests := []struct {
		input  string
		w, h   int
		hasErr bool
	}{
		{"1920x1080", 1920, 1080, false},
		{"800x600", 800, 600, false},
		{"invalid", 0, 0, true},
		{"100x", 0, 0, true},
		{"x100", 0, 0, true},
		{"axb", 0, 0, true},
	}

	for _, tc := range tests {
		w, h, err := ParseResolution(tc.input)
		if tc.hasErr {
			if err == nil {
				t.Errorf("ParseResolution(%q) expected error, got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseResolution(%q) unexpected error: %v", tc.input, err)
			}
			if w != tc.w || h != tc.h {
				t.Errorf("ParseResolution(%q) = %dx%d; want %dx%d", tc.input, w, h, tc.w, tc.h)
			}
		}
	}
}

func TestParseAspectRatio(t *testing.T) {
	tests := []struct {
		input  string
		want   float64
		hasErr bool
	}{
		{"16:9", 16.0 / 9.0, false},
		{"4:3", 4.0 / 3.0, false},
		{"21:9", 21.0 / 9.0, false},
		{"invalid", 0, true},
		{"16:", 0, true},
	}

	for _, tc := range tests {
		got, err := ParseAspectRatio(tc.input)
		if tc.hasErr {
			if err == nil {
				t.Errorf("ParseAspectRatio(%q) expected error, got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseAspectRatio(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseAspectRatio(%q) = %f; want %f", tc.input, got, tc.want)
			}
		}
	}
}

func TestBuildSearchQuery(t *testing.T) {
	flags := map[string]interface{}{
		"min-width":    1920,
		"aspect-ratio": "16:9",
	}
	query := BuildSearchQuery(flags)
	if !strings.Contains(query, "min_width=1920") {
		t.Errorf("Query missing min_width: %s", query)
	}
	if !strings.Contains(query, "aspect_ratio=16:9") {
		t.Errorf("Query missing aspect_ratio: %s", query)
	}
}
