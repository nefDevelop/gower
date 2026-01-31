// cmd/state.go
package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gower/internal/utils"
)

// State represents the persistent state of the application.
type State struct {
	CurrentWallpaperID  string `json:"current_wallpaper_id"`
	PreviousWallpaperID string `json:"previous_wallpaper_id"`
}

// stateFilePath returns the path to the state file.
func stateFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gower", "state.json"), nil
}

// loadState reads the application state from the state file.
// If the file doesn't exist, it returns a new empty State.
func loadState() (*State, error) {
	path, err := stateFilePath()
	if err != nil {
		return nil, err
	}

	state := &State{}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, return a fresh state
		return state, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// If the file is empty, return a fresh state
	if len(data) == 0 {
		return state, nil
	}

	if err := json.Unmarshal(data, state); err != nil {
		utils.Log.Warn("State file is corrupt, starting with a fresh state. Error: %v", err)
		// Return a fresh state if unmarshalling fails
		return &State{}, nil
	}

	return state, nil
}

// saveState writes the application state to the state file.
func (s *State) saveState() error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
