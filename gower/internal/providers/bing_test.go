package providers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBingProvider_Search(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request parameters
		if r.URL.Path != "/HPImageArchive.aspx" {
			t.Errorf("Expected path /HPImageArchive.aspx, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("format") != "js" {
			t.Errorf("Expected format=js, got %s", r.URL.Query().Get("format"))
		}
		if r.URL.Query().Get("idx") != "0" { // opts.Page-1, default page is 1
			t.Errorf("Expected idx=0, got %s", r.URL.Query().Get("idx"))
		}
		if r.URL.Query().Get("n") != "1" { // We request only one image
			t.Errorf("Expected n=1, got %s", r.URL.Query().Get("n"))
		}
		if r.URL.Query().Get("mkt") != "en-US" { // From provider config
			t.Errorf("Expected mkt=en-US, got %s", r.URL.Query().Get("mkt"))
		}

		// Mock response
		response := `{
			"images": [
				{
					"hsh": "12345",
					"url": "/th?id=OHR.TestImage_EN-US123.jpg_1920x1080.jpg",
					"urlbase": "/th?id=OHR.TestImage_EN-US123.jpg",
					"copyright": "Test Image (© Bing)",
					"drk": 0
				}
			]
		}`
		_, _ = fmt.Fprintln(w, response)
	}))
	defer server.Close()

	// Override BingBaseURL to point to our mock server
	originalBingBaseURL := BingBaseURL
	BingBaseURL = server.URL + "/HPImageArchive.aspx"
	defer func() { BingBaseURL = originalBingBaseURL }()

	provider := NewBingProvider("en-US")
	wallpapers, err := provider.Search("", SearchOptions{Page: 1}) // Request page 1 (idx=0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(wallpapers) != 1 {
		t.Errorf("Expected 1 result, got %d", len(wallpapers))
	}
	if wallpapers[0].ID != "bing_12345" {
		t.Errorf("Expected ID bing_12345, got %s", wallpapers[0].ID)
	}
	expectedURL := "https://www.bing.com/th?id=OHR.TestImage_EN-US123.jpg_1920x1080.jpg"
	if wallpapers[0].URL != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, wallpapers[0].URL)
	}
	if wallpapers[0].Dimension != "1920x1080" {
		t.Errorf("Expected Dimension 1920x1080, got %s", wallpapers[0].Dimension)
	}
	if wallpapers[0].Source != "bing" {
		t.Errorf("Expected Source bing, got %s", wallpapers[0].Source)
	}
}

func TestBingProvider_GetCategories(t *testing.T) {
	provider := NewBingProvider("en-US")
	categories := provider.GetCategories()
	if len(categories) != 0 { // New Bing provider returns empty categories
		t.Errorf("Expected 0 categories, got %v", categories)
	}
}
