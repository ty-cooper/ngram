package dream

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const stateFile = "_meta/dream-state.json"

// dreamState maps note ID → last reviewed timestamp.
type dreamState map[string]time.Time

func loadState(vaultPath string) dreamState {
	path := filepath.Join(vaultPath, stateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return make(dreamState)
	}
	var s dreamState
	if err := json.Unmarshal(data, &s); err != nil {
		return make(dreamState)
	}
	return s
}

func saveState(vaultPath string, s dreamState) error {
	path := filepath.Join(vaultPath, stateFile)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// needsReview returns true if the note has been modified since dream last reviewed it.
func (s dreamState) needsReview(id string, modified time.Time) bool {
	reviewed, ok := s[id]
	if !ok {
		return true
	}
	// Notes without a modified field (zero time) are always reviewed.
	if modified.IsZero() {
		return true
	}
	return modified.After(reviewed)
}
