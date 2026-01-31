// cmd/state_test.go
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupStateTest creates a temporary directory for testing.
func setupStateTest(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "gower-state-test")
	assert.NoError(t, err)

	// Override the user home dir to control where the state file is created.
	// We can't directly override os.UserHomeDir, so we'll create the .gower
	// directory inside our tempDir and adjust the path function for tests
	// (or rely on a test-specific helper). For this test, we'll just
	// manually create the file inside the expected structure.
	gowerDir := filepath.Join(tempDir, ".gower")
	err = os.MkdirAll(gowerDir, 0755)
	assert.NoError(t, err)

	// Monkey patch the stateFilePath function during tests
	originalStateFilePath := stateFilePath
	stateFilePath = func() (string, error) {
		return filepath.Join(gowerDir, "state.json"), nil
	}

	cleanup := func() {
		stateFilePath = originalStateFilePath
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestSaveAndLoadState(t *testing.T) {
	_, cleanup := setupStateTest(t)
	defer cleanup()

	// 1. Create a new state
	initialState := &State{
		CurrentWallpaperID:  "wall_123",
		PreviousWallpaperID: "wall_abc",
	}

	// 2. Save the state
	err := initialState.saveState()
	assert.NoError(t, err)

	// 3. Load the state back
	loadedState, err := loadState()
	assert.NoError(t, err)
	assert.NotNil(t, loadedState)

	// 4. Verify the contents
	assert.Equal(t, "wall_123", loadedState.CurrentWallpaperID)
	assert.Equal(t, "wall_abc", loadedState.PreviousWallpaperID)
}

func TestLoadState_NonExistent(t *testing.T) {
	_, cleanup := setupStateTest(t)
	defer cleanup()

	// Load state without saving one first
	state, err := loadState()
	assert.NoError(t, err)
	assert.NotNil(t, state)

	// Should be an empty state
	assert.Equal(t, "", state.CurrentWallpaperID)
	assert.Equal(t, "", state.PreviousWallpaperID)
}

func TestLoadState_Corrupt(t *testing.T) {
	tempDir, cleanup := setupStateTest(t)
	defer cleanup()

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
}
