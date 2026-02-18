package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gower/pkg/models"
)

func resetFavoritesFlags() {
	favPage = 1
	favLimit = 10
	favNotes = ""
	favColor = ""
	favForce = false
	favAll = false
	favFile = ""
}

func TestFavoritesListEmpty(t *testing.T) {
	resetFavoritesFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	output, err := executeCommand(rootCmd, "favorites", "list")
	if err != nil {
		t.Fatalf("Error executing favorites list: %v", err)
	}

	if !strings.Contains(output, "No favorite wallpapers yet.") {
		t.Errorf("Expected 'No favorite wallpapers yet.', got: %s", output)
	}
}

func TestFavoritesAddAndList(t *testing.T) {
	resetFavoritesFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Add a favorite
	output, err := executeCommand(rootCmd, "favorites", "add", "test-id-1", "--notes", "My cool wallpaper")
	if err != nil {
		t.Fatalf("Error executing favorites add: %v", err)
	}
	if !strings.Contains(output, "Wallpaper test-id-1 added to favorites list.") {
		t.Errorf("Expected 'Wallpaper test-id-1 added to favorites list.', got: %s", output)
	}

	// List favorites
	output, err = executeCommand(rootCmd, "favorites", "list")
	if err != nil {
		t.Fatalf("Error executing favorites list: %v", err)
	}
	if !strings.Contains(output, "ID: test-id-1") {
		t.Errorf("Expected 'ID: test-id-1' in list output, got: %s", output)
	}
	if !strings.Contains(output, "Notes: My cool wallpaper") {
		t.Errorf("Expected notes in list output, got: %s", output)
	}

	// Try adding the same favorite again
	output, err = executeCommand(rootCmd, "favorites", "add", "test-id-1")
	if err != nil {
		t.Fatalf("Error executing favorites add (duplicate): %v", err)
	}
	if !strings.Contains(output, "Wallpaper test-id-1 is already in favorites.") {
		t.Errorf("Expected 'Wallpaper test-id-1 is already in favorites.', got: %s", output)
	}
}

func TestFavoritesRemove(t *testing.T) {
	resetFavoritesFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")
	executeCommand(rootCmd, "favorites", "add", "test-id-1")
	executeCommand(rootCmd, "favorites", "add", "test-id-2")

	// Remove a favorite
	output, err := executeCommand(rootCmd, "favorites", "remove", "test-id-1")
	if err != nil {
		t.Fatalf("Error executing favorites remove: %v", err)
	}
	if !strings.Contains(output, "Wallpaper test-id-1 removed from favorites.") {
		t.Errorf("Expected 'Wallpaper test-id-1 removed from favorites.', got: %s", output)
	}

	// List to confirm removal
	output, err = executeCommand(rootCmd, "favorites", "list")
	if err != nil {
		t.Fatalf("Error executing favorites list: %v", err)
	}
	if strings.Contains(output, "ID: test-id-1") {
		t.Errorf("Expected 'ID: test-id-1' to be removed, but it's still in output: %s", output)
	}
	if !strings.Contains(output, "ID: test-id-2") {
		t.Errorf("Expected 'ID: test-id-2' to remain in output: %s", output)
	}

	// Try removing a non-existent favorite
	output, err = executeCommand(rootCmd, "favorites", "remove", "non-existent-id")
	if err != nil {
		t.Fatalf("Error executing favorites remove (non-existent): %v", err)
	}
	if !strings.Contains(output, "Wallpaper non-existent-id not found in favorites.") {
		t.Errorf("Expected 'Wallpaper non-existent-id not found in favorites.', got: %s", output)
	}

	// Try removing with force
	output, err = executeCommand(rootCmd, "favorites", "remove", "non-existent-id", "--force")
	if err != nil {
		t.Fatalf("Error executing favorites remove with force: %v", err)
	}
	if strings.Contains(output, "not found") {
		t.Errorf("Expected no output with force, got: %s", output)
	}
}

func TestFavoritesExportAndImport(t *testing.T) {
	resetFavoritesFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")
	executeCommand(rootCmd, "favorites", "add", "fav-id-1")
	executeCommand(rootCmd, "favorites", "add", "fav-id-2")

	exportFile := filepath.Join(tmpDir, "favorites_backup.json")

	// Export favorites
	output, err := executeCommand(rootCmd, "export", "favorites", "--file", exportFile)
	if err != nil {
		t.Fatalf("Error executing favorites export: %v", err)
	}
	if !strings.Contains(output, "Favorites exported to") {
		t.Errorf("Expected 'Favorites exported to', got: %s", output)
	}
	if _, err := os.Stat(exportFile); os.IsNotExist(err) {
		t.Errorf("Export file was not created")
	}

	// Clear current favorites
	executeCommand(rootCmd, "favorites", "remove", "fav-id-1")
	executeCommand(rootCmd, "favorites", "remove", "fav-id-2")

	// Import favorites
	output, err = executeCommand(rootCmd, "import", "favorites", "--file", exportFile)
	if err != nil {
		t.Fatalf("Error executing favorites import: %v", err)
	}
	if !strings.Contains(output, "Favorites imported successfully") {
		t.Errorf("Expected 'Favorites imported successfully', got: %s", output)
	}

	// List to confirm import
	output, err = executeCommand(rootCmd, "favorites", "list")
	if err != nil {
		t.Fatalf("Error executing favorites list: %v", err)
	}
	if !strings.Contains(output, "ID: fav-id-1") || !strings.Contains(output, "ID: fav-id-2") {
		t.Errorf("Expected imported favorites in list, got: %s", output)
	}

	// Test import with existing favorites and a new one
	executeCommand(rootCmd, "favorites", "add", "fav-id-3") // Add a new one
	// Create a new import file with fav-id-1 and fav-id-4
	newImportFavs := []FavoriteWallpaper{
		{Wallpaper: models.Wallpaper{ID: "fav-id-1", URL: "url-1", Source: "src-1"}},
		{Wallpaper: models.Wallpaper{ID: "fav-id-4", URL: "url-4", Source: "src-4"}},
	}
	newImportFile := filepath.Join(tmpDir, "favorites_new_import.json")
	data, _ := json.MarshalIndent(newImportFavs, "", "  ")
	os.WriteFile(newImportFile, data, 0644)

	_, err = executeCommand(rootCmd, "import", "favorites", "--file", newImportFile)
	if err != nil {
		t.Fatalf("Error executing favorites import: %v", err)
	}
	output, err = executeCommand(rootCmd, "favorites", "list")
	if err != nil {
		t.Fatalf("Error executing favorites list (after import): %v", err)
	}
	if !strings.Contains(output, "ID: fav-id-1") || !strings.Contains(output, "ID: fav-id-4") {
		t.Errorf("Expected imported favorites, got: %s", output)
	}
	if strings.Contains(output, "ID: fav-id-3") {
		t.Errorf("Expected fav-id-3 to be overwritten, but it is present")
	}
}

func TestFavoritesListColor(t *testing.T) {
	resetFavoritesFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Manually save favorites with colors to test filtering
	favs := []FavoriteWallpaper{
		{Wallpaper: models.Wallpaper{ID: "red-wp", Color: "#FF0000"}, Notes: "Red"},
		{Wallpaper: models.Wallpaper{ID: "blue-wp", Color: "#0000FF"}, Notes: "Blue"},
	}
	if err := saveFavorites(favs); err != nil {
		t.Fatalf("Failed to save favorites: %v", err)
	}

	// Manually create a dynamic palette for the test
	colorsPath := filepath.Join(tmpDir, ".config", "gower", "data", "colors.json")
	paletteJSON := `{"favorites_palette": ["#FF0000", "#0000FF"]}`
	if err := os.WriteFile(colorsPath, []byte(paletteJSON), 0644); err != nil {
		t.Fatalf("Failed to write colors.json for test: %v", err)
	}

	// Filter by Red
	output, err := executeCommand(rootCmd, "favorites", "list", "--color", "FF0000")
	if err != nil {
		t.Fatalf("Error executing favorites list --color: %v", err)
	}

	if !strings.Contains(output, "ID: red-wp") {
		t.Errorf("Expected red-wp in output, got: %s", output)
	}
	if strings.Contains(output, "ID: blue-wp") {
		t.Errorf("Did not expect blue-wp in output, got: %s", output)
	}

	// Filter by a color close to red
	output, err = executeCommand(rootCmd, "favorites", "list", "--color", "EE1111")
	if err != nil {
		t.Fatalf("Error executing favorites list --color: %v", err)
	}

	if !strings.Contains(output, "ID: red-wp") {
		t.Errorf("Expected red-wp in output when filtering by near-red, got: %s", output)
	}
	if strings.Contains(output, "ID: blue-wp") {
		t.Errorf("Did not expect blue-wp in output when filtering by near-red, got: %s", output)
	}
}

func TestFavoritesAddWithPersistence(t *testing.T) {
	resetFavoritesFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Setup wallpapers dir
	wallpapersDir := filepath.Join(tmpDir, "Wallpapers")
	os.MkdirAll(wallpapersDir, 0755)

	// Enable persistence
	executeCommand(rootCmd, "config", "set", "behavior.save_favorites_to_folder=true")
	executeCommand(rootCmd, "config", "set", "paths.wallpapers="+wallpapersDir)

	// Create a dummy local file to add as favorite
	localFile := filepath.Join(tmpDir, "source.jpg")
	os.WriteFile(localFile, []byte("dummy image content"), 0644)

	// Add to feed manually so favorites add can find it and use the local path
	feedPath := filepath.Join(tmpDir, ".config", "gower", "data", "feed.json")
	feed := []models.Wallpaper{
		{ID: "local-fav", URL: localFile, Source: "local"},
	}
	data, _ := json.Marshal(feed)
	os.WriteFile(feedPath, data, 0644)

	// Add to favorites
	executeCommand(rootCmd, "favorites", "add", "local-fav")

	// Verify file exists in wallpapersDir
	destPath := filepath.Join(wallpapersDir, "local-fav.jpg")

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("Favorite was not copied to wallpapers folder")
	}
}

func TestFavoritesGetColors(t *testing.T) {
	resetFavoritesFlags()
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Manually create colors.json with favorite palette entries
	colorsPath := filepath.Join(tmpDir, ".config", "gower", "data", "colors.json")
	expectedColors := []string{"#FF0000", "#00FF00", "#0000FF"}
	paletteJSON := `{"feed_palette": [], "favorites_palette": ["#FF0000", "#00FF00", "#0000FF"]}`
	if err := os.WriteFile(colorsPath, []byte(paletteJSON), 0644); err != nil {
		t.Fatalf("Failed to write colors.json for test: %v", err)
	}

	// Test plain output
	output, err := executeCommand(rootCmd, "favorites", "get", "colors")
	if err != nil {
		t.Fatalf("Error executing favorites get colors: %v", err)
	}

	for _, color := range expectedColors {
		if !strings.Contains(output, color) {
			t.Errorf("Expected color %s in output, got: %s", color, output)
		}
	}

	// Test JSON output
	config.JSONOutput = true
	output, err = executeCommand(rootCmd, "favorites", "get", "colors")
	config.JSONOutput = false // Reset for other tests
	if err != nil {
		t.Fatalf("Error executing favorites get colors with JSON output: %v", err)
	}

	var actualColors []string
	if err := json.Unmarshal([]byte(output), &actualColors); err != nil {
		t.Fatalf("Failed to unmarshal JSON output: %v", err)
	}

	if len(actualColors) != len(expectedColors) {
		t.Errorf("Expected %d colors, got %d", len(expectedColors), len(actualColors))
	}
	for i, color := range expectedColors {
		if actualColors[i] != color {
			t.Errorf("Expected color %s at index %d, got %s", color, i, actualColors[i])
		}
	}
}
