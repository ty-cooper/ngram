package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Verify all directories exist.
	for _, d := range vaultDirs {
		path := filepath.Join(dir, d)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("missing dir %s: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", d)
		}
	}

	// Verify seed files exist.
	seedFiles := []string{
		"_meta/taxonomy.yml",
		"_meta/topic-clusters.yml",
		"_templates/knowledge-note.md",
		"_templates/box.md",
	}
	for _, f := range seedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("missing seed file %s", f)
		}
	}
}

func TestInit_Idempotent(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init 1: %v", err)
	}

	// Write custom content to taxonomy.
	taxPath := filepath.Join(dir, "_meta", "taxonomy.yml")
	os.WriteFile(taxPath, []byte("custom content"), 0o644)

	// Init again should not overwrite.
	if err := Init(dir); err != nil {
		t.Fatalf("Init 2: %v", err)
	}

	data, _ := os.ReadFile(taxPath)
	if string(data) != "custom content" {
		t.Error("Init overwrote existing taxonomy.yml")
	}
}
