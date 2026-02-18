package providers

import (
	"fmt"
	"gower/pkg/models"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenericProvider_Search(t *testing.T) {
	// Mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{"results": [{"id": "123", "url": "http://example.com/img.jpg"}]}`)
	}))
	defer ts.Close()

	config := models.GenericProviderConfig{
		Name:    "test_provider",
		Enabled: true,
		APIURL:  ts.URL,
		ResponseMapping: models.ResponseMapping{
			ResultsPath: "results",
			IDPath:      "id",
			URLPath:     "url",
		},
	}

	provider := &GenericProvider{Config: config}

	results, err := provider.Search("test", SearchOptions{})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].ID != "test_provider-123" {
		t.Errorf("Expected ID test_provider-123, got %s", results[0].ID)
	}
}
