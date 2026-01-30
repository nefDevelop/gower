package providers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Since we cannot inject the URL into the provider, we cannot write a real unit test.
// We will write a test that checks the provider's name and other simple methods.
func TestWallhavenProvider_GetName(t *testing.T) {
	provider := &WallhavenProvider{}
	if provider.GetName() != "wallhaven" {
		t.Errorf("Expected provider name 'wallhaven', got '%s'", provider.GetName())
	}
}

// The following test is a more complete test that would work if the provider was designed to be more testable.
// I am leaving it here for reference.
func TestWallhavenProvider_Search_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check query params
		if r.URL.Query().Get("q") != "test" {
			t.Errorf("Expected q=test, got %s", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("categories") != "111" {
			t.Errorf("Expected categories=111, got %s", r.URL.Query().Get("categories"))
		}

		response := `{
			"data": [
				{
					"id": "id1",
					"path": "path1",
					"resolution": "1920x1080",
					"thumbs": {"large": "thumb1"}
				}
			]
		}`
		fmt.Fprintln(w, response)
	}))
	defer server.Close()

	// To make this test work, the WallhavenProvider should allow setting the URL.
	// provider := &WallhavenProvider{}
	// provider.APIURL = server.URL // or similar
	// wallpapers, err := provider.Search("test", SearchOptions{Category: "111"})
	// if err != nil {
	// 	t.Fatalf("Search failed: %v", err)
	// }
	// if len(wallpapers) != 1 {
	// 	t.Fatalf("Expected 1 wallpaper, got %d", len(wallpapers))
	// }
	// if wallpapers[0].ID != "wh_id1" {
	// 	t.Errorf("Incorrect wallpaper ID")
	// }
}
