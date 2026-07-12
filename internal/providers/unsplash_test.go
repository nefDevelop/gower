package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnsplashProvider_GetName(t *testing.T) {
	provider := &UnsplashProvider{APIKey: "test"}
	if provider.GetName() != "unsplash" {
		t.Errorf("Expected name 'unsplash', got '%s'", provider.GetName())
	}
}

func TestUnsplashProvider_Search_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Client-ID test-key" {
			t.Errorf("Expected Authorization header, got '%s'", r.Header.Get("Authorization"))
		}

		response := `[
			{
				"id": "abc123",
				"width": 1920,
				"height": 1080,
				"color": "#336699",
				"description": "Test photo",
				"alt_description": "a test photo",
				"urls": {
					"raw": "https://images.unsplash.com/photo-123",
					"regular": "https://images.unsplash.com/photo-123?w=1080",
					"small": "https://images.unsplash.com/photo-123?w=400",
					"thumb": "https://images.unsplash.com/photo-123?w=200"
				},
				"user": {
					"name": "Test User",
					"username": "testuser"
				},
				"links": {
					"html": "https://unsplash.com/photos/abc123"
				}
			}
		]`
		_, _ = fmt.Fprintln(w, response)
	}))
	defer server.Close()

	originalURL := UnsplashBaseURL
	UnsplashBaseURL = server.URL
	defer func() { UnsplashBaseURL = originalURL }()

	provider := &UnsplashProvider{APIKey: "test-key"}
	wallpapers, err := provider.Search(context.Background(), "", SearchOptions{Limit: 5})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(wallpapers) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(wallpapers))
	}
	if wallpapers[0].ID != "us_abc123" {
		t.Errorf("Expected ID us_abc123, got %s", wallpapers[0].ID)
	}
	if wallpapers[0].Source != "unsplash" {
		t.Errorf("Expected Source unsplash, got %s", wallpapers[0].Source)
	}
	if wallpapers[0].Color != "#336699" {
		t.Errorf("Expected Color #336699, got %s", wallpapers[0].Color)
	}
	if wallpapers[0].Dimension != "1920x1080" {
		t.Errorf("Expected Dimension 1920x1080, got %s", wallpapers[0].Dimension)
	}
}

func TestUnsplashProvider_SearchPhotos_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"total": 1,
			"results": [
				{
					"id": "def456",
					"width": 2560,
					"height": 1440,
					"color": "#000000",
					"description": "Search result",
					"urls": {
						"raw": "https://images.unsplash.com/photo-456",
						"regular": "https://images.unsplash.com/photo-456?w=1080",
						"small": "https://images.unsplash.com/photo-456?w=400"
					},
					"user": {"name": "User", "username": "user"},
					"links": {"html": "https://unsplash.com/photos/def456"}
				}
			]
		}`
		_, _ = fmt.Fprintln(w, response)
	}))
	defer server.Close()

	originalURL := UnsplashBaseURL
	UnsplashBaseURL = server.URL
	defer func() { UnsplashBaseURL = originalURL }()

	provider := &UnsplashProvider{APIKey: "test-key"}
	wallpapers, err := provider.Search(context.Background(), "mountains", SearchOptions{Limit: 5})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(wallpapers) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(wallpapers))
	}
	if wallpapers[0].ID != "us_def456" {
		t.Errorf("Expected ID us_def456, got %s", wallpapers[0].ID)
	}
}
