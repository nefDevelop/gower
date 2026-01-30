package providers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gower/pkg/models"

	"github.com/tidwall/gjson"
)

// GenericProvider is a provider that can be configured via config file.
type GenericProvider struct {
	Config models.GenericProviderConfig
}

// GetName returns the name of the provider.
func (p *GenericProvider) GetName() string {
	return p.Config.Name
}

// Search searches for wallpapers using the configured API.
func (p *GenericProvider) Search(query string, options SearchOptions) ([]models.Wallpaper, error) {
	if !p.Config.Enabled {
		return nil, fmt.Errorf("provider %s is not enabled", p.Config.Name)
	}

	// Build URL
	url := strings.ReplaceAll(p.Config.APIURL, "{query}", query)
	url = strings.ReplaceAll(url, "{apikey}", p.Config.APIKey)

	// Make HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code from %s: %d", url, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Load parser mapping from file
	homeDir, _ := os.UserHomeDir()
	mappingFile := filepath.Join(homeDir, ".gower", "data", "parser", p.Config.Name+".json")

	var mapping models.ResponseMapping
	mappingData, err := ioutil.ReadFile(mappingFile)
	if err == nil {
		json.Unmarshal(mappingData, &mapping)
	} else {
		// Fallback to config if file missing (backward compatibility)
		mapping = p.Config.ResponseMapping
	}

	// Parse JSON response
	var wallpapers []models.Wallpaper
	results := gjson.Get(string(body), mapping.ResultsPath)

	if !results.Exists() {
		return nil, fmt.Errorf("results path not found in response: %s", mapping.ResultsPath)
	}

	results.ForEach(func(key, value gjson.Result) bool {
		id := gjson.Get(value.Raw, mapping.IDPath).String()
		imageURL := gjson.Get(value.Raw, mapping.URLPath).String()
		dimension := gjson.Get(value.Raw, mapping.DimensionPath).String()

		if id != "" && imageURL != "" {
			wallpapers = append(wallpapers, models.Wallpaper{
				ID:        fmt.Sprintf("%s-%s", p.GetName(), id),
				URL:       imageURL,
				Dimension: dimension,
				Source:    p.GetName(),
			})
		}
		return true // continue iterating
	})

	return wallpapers, nil
}
