// cmd/state_test.go
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupStateTest creates a temporary directory for testing.
func setupStateTest(t *testing.T) string {
	tempDir := setupTestHome(t)

	// Create the .config/gower directory inside our tempDir
	gowerDir := filepath.Join(tempDir, ".config", "gower")
	err := os.MkdirAll(gowerDir, 0755)
	assert.NoError(t, err)

	// Monkey patch the stateFilePath function during tests
	originalStateFilePath := stateFilePath
	stateFilePath = func() (string, error) {
		return filepath.Join(gowerDir, "state.json"), nil
	}

	t.Cleanup(func() {
		stateFilePath = originalStateFilePath
	})

	return tempDir
}

func TestSaveAndLoadState(t *testing.T) {
	setupStateTest(t)

	// 1. Create a new state
	initialState := &State{
		CurrentWallpaperID:  "wall_123",
		PreviousWallpaperID: "wall_abc",
		CurrentWallpapers:   []string{"wall_123", "wall_456"},
		PreviousWallpapers:  []string{"wall_abc", "wall_def"},
	}

	// 2. Save the state
	err := saveState(initialState)
	assert.NoError(t, err)

	// 3. Load the state back
	loadedState, err := loadState()
	assert.NoError(t, err)
	assert.NotNil(t, loadedState)

	// 4. Verify the contents
	assert.Equal(t, "wall_123", loadedState.CurrentWallpaperID)
	assert.Equal(t, "wall_abc", loadedState.PreviousWallpaperID)
	assert.Equal(t, []string{"wall_123", "wall_456"}, loadedState.CurrentWallpapers)
	assert.Equal(t, []string{"wall_abc", "wall_def"}, loadedState.PreviousWallpapers)
}

func TestLoadState_NonExistent(t *testing.T) {
	setupStateTest(t)

	// Load state without saving one first
	state, err := loadState()
	assert.NoError(t, err)
	assert.NotNil(t, state)

	// Should be an empty state
	assert.Equal(t, "", state.CurrentWallpaperID)
	assert.Equal(t, "", state.PreviousWallpaperID)
	assert.Empty(t, state.CurrentWallpapers)
}

func TestLoadState_Corrupt(t *testing.T) {
	setupStateTest(t)

	// Create a corrupt state file
	stateFile, err := stateFilePath()
	assert.NoError(t, err)
	err = os.WriteFile(stateFile, []byte("{ not json "), 0644)
	assert.NoError(t, err)

	// Try to load it
	state, err := loadState()
	assert.NoError(t, err) // Should not error, but return a fresh state
	assert.NotNil(t, state)

	// Should be a fresh empty state
	assert.Equal(t, "", state.CurrentWallpaperID)
	assert.Equal(t, "", state.PreviousWallpaperID)
	assert.Empty(t, state.CurrentWallpapers)
}
