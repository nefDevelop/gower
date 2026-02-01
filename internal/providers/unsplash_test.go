// internal/providers/unsplash_test.go
package providers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const mockUnsplashRandomResponse = `[
  {
    "id": "mock_id_1",
    "description": "A beautiful landscape",
    "width": 1920,
    "height": 1080,
    "color": "#aabbcc",
    "urls": {
      "full": "https://example.com/full_1.jpg",
      "thumb": "https://example.com/thumb_1.jpg"
    },
    "user": {
      "name": "John Doe"
    }
  }
]`

const mockUnsplashSearchResponse = `{
  "results": [
    {
      "id": "mock_id_2",
      "description": "A cute cat",
      "width": 1080,
      "height": 1920,
      "color": "#ccbbaa",
      "urls": {
        "full": "https://example.com/full_2.jpg",
        "thumb": "https://example.com/thumb_2.jpg"
      },
      "user": {
        "name": "Jane Doe"
      }
    }
  ]
}`

func TestUnsplashProvider_SearchRandom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/photos/random?count=1", r.URL.String())
		assert.Equal(t, "Client-ID test_api_key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, mockUnsplashRandomResponse)
	}))
	defer server.Close()

	// Override the base URL to point to our mock server
	originalBaseURL := unsplashAPIBaseURL
	unsplashAPIBaseURL = server.URL
	defer func() { unsplashAPIBaseURL = originalBaseURL }()

	provider := NewUnsplashProvider("test_api_key")
	wallpapers, err := provider.Search("", SearchOptions{Limit: 1})

	assert.NoError(t, err)
	assert.Len(t, wallpapers, 1)

	wp := wallpapers[0]
	assert.Equal(t, "us_mock_id_1", wp.ID)
	assert.Equal(t, "https://example.com/full_1.jpg", wp.URL)
	assert.Equal(t, "A beautiful landscape", wp.Title)
	assert.Equal(t, "#aabbcc", wp.Color)
}

func TestUnsplashProvider_SearchWithQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search/photos?query=cats&per_page=1", r.URL.String())
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, mockUnsplashSearchResponse)
	}))
	defer server.Close()

	originalBaseURL := unsplashAPIBaseURL
	unsplashAPIBaseURL = server.URL
	defer func() { unsplashAPIBaseURL = originalBaseURL }()

	provider := NewUnsplashProvider("test_api_key")
	wallpapers, err := provider.Search("cats", SearchOptions{Limit: 1})

	assert.NoError(t, err)
	assert.Len(t, wallpapers, 1)

	wp := wallpapers[0]
	assert.Equal(t, "us_mock_id_2", wp.ID)
	assert.Equal(t, "https://example.com/full_2.jpg", wp.URL)
	assert.Equal(t, "A cute cat", wp.Title)
}

func TestUnsplashProvider_NoAPIKey(t *testing.T) {
	provider := NewUnsplashProvider("")
	_, err := provider.Search("cats", SearchOptions{Limit: 1})
	assert.Error(t, err)
	assert.Equal(t, "Unsplash API key is not configured", err.Error())
}
