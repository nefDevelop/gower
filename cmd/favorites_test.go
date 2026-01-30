package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gower/pkg/models"
)

func TestFavoritesListEmpty(t *testing.T) {
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
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")

	// Add a favorite
	output, err := executeCommand(rootCmd, "favorites", "add", "test-id-1", "--notes", "My cool wallpaper")
	if err != nil {
		t.Fatalf("Error executing favorites add: %v", err)
	}
	if !strings.Contains(output, "Wallpaper test-id-1 added to favorites.") {
		t.Errorf("Expected 'Wallpaper test-id-1 added to favorites.', got: %s", output)
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
	tmpDir := setupTestHome(t)
	defer os.RemoveAll(tmpDir)

	executeCommand(rootCmd, "config", "init")
	executeCommand(rootCmd, "favorites", "add", "fav-id-1")
	executeCommand(rootCmd, "favorites", "add", "fav-id-2")

	exportFile := filepath.Join(tmpDir, "favorites_backup.json")

	// Export favorites
	output, err := executeCommand(rootCmd, "favorites", "export", "--file", exportFile)
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
	output, err = executeCommand(rootCmd, "favorites", "import", "--file", exportFile)
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

	output, err = executeCommand(rootCmd, "favorites", "import", "--file", newImportFile)
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
