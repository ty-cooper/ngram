package llm

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBudget_UnderLimit(t *testing.T) {
	b := &Budget{MaxCallsPerDay: 10, VaultPath: t.TempDir()}
	if err := b.Check(); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestBudget_ExceedsLimit(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "_meta"), 0o755)

	// Write enough log entries for today.
	today := time.Now().Format("2006-01-02")
	var lines string
	for i := 0; i < 5; i++ {
		lines += `{"timestamp":"` + today + `T10:00:00Z","component":"processor","note_id":"test","duration_ms":100}` + "\n"
	}
	os.WriteFile(filepath.Join(dir, "_meta", "api-usage.jsonl"), []byte(lines), 0o644)

	b := &Budget{MaxCallsPerDay: 5, VaultPath: dir}
	if err := b.Check(); err == nil {
		t.Error("expected budget exceeded error")
	}
}

func TestBudget_NoLimit(t *testing.T) {
	b := &Budget{MaxCallsPerDay: 0, VaultPath: t.TempDir()}
	// Should never error with no limit.
	for i := 0; i < 1000; i++ {
		b.Increment()
	}
	if err := b.Check(); err != nil {
		t.Fatalf("Check with no limit: %v", err)
	}
}

func TestBudget_Increment(t *testing.T) {
	b := &Budget{MaxCallsPerDay: 10, VaultPath: t.TempDir()}
	b.Increment()
	b.Increment()
	if b.Count() != 2 {
		t.Errorf("Count = %d, want 2", b.Count())
	}
}
