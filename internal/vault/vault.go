package vault

import (
	"os"
	"path/filepath"
)

// InboxDir returns the absolute path to _inbox/, creating it if needed.
func InboxDir(vaultPath string) (string, error) {
	return ensureDir(filepath.Join(vaultPath, "_inbox"))
}

// BoxDir returns the absolute path to boxes/{name}/.
func BoxDir(vaultPath, name string) string {
	return filepath.Join(vaultPath, "boxes", name)
}

// EnsureDir creates a directory and all parents if they don't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func ensureDir(path string) (string, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}
