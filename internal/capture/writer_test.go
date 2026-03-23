package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ty-cooper/ngram/internal/config"
)

func TestWriteNote(t *testing.T) {
	vault := t.TempDir()

	meta := NoteMetadata{
		Title:  "test note",
		Source: "terminal",
		Time:   time.Date(2026, 3, 22, 10, 42, 0, 0, time.UTC),
	}

	result, err := WriteNote(vault, "hello world", meta)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(result.RelPath, "_inbox/") {
		t.Errorf("RelPath = %q, want _inbox/ prefix", result.RelPath)
	}
	if !strings.Contains(result.RelPath, "test-note.md") {
		t.Errorf("RelPath = %q, want to contain test-note.md", result.RelPath)
	}

	content, err := os.ReadFile(result.AbsPath)
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.HasPrefix(s, "---\n") {
		t.Error("file should start with frontmatter")
	}
	if !strings.Contains(s, "hello world") {
		t.Error("file should contain body text")
	}
	if !strings.Contains(s, `source: "terminal"`) {
		t.Error("file should contain source")
	}
}

func TestWriteNote_WithBoxCtx(t *testing.T) {
	vault := t.TempDir()

	meta := NoteMetadata{
		Title:  "sqli found",
		Source: "terminal",
		Time:   time.Now(),
		BoxCtx: &config.BoxContext{Box: "optimum", Phase: "exploit"},
	}

	result, err := WriteNote(vault, "sqli on login", meta)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(result.AbsPath)
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.Contains(s, `box: "optimum"`) {
		t.Error("should contain box in frontmatter")
	}
}

func TestConfirmation(t *testing.T) {
	got := Confirmation("_inbox/123-test.md", &config.BoxContext{Box: "optimum", Phase: "exploit"})
	if got != "✓ _inbox/123-test.md [optimum/exploit]" {
		t.Errorf("got %q", got)
	}

	got = Confirmation("_inbox/123-test.md", nil)
	if got != "✓ _inbox/123-test.md" {
		t.Errorf("got %q", got)
	}
}

func TestWriteNote_CreatesInbox(t *testing.T) {
	vault := t.TempDir()
	inbox := filepath.Join(vault, "_inbox")

	// inbox doesn't exist yet
	if _, err := os.Stat(inbox); !os.IsNotExist(err) {
		t.Fatal("inbox should not exist yet")
	}

	meta := NoteMetadata{Title: "test", Source: "terminal", Time: time.Now()}
	_, err := WriteNote(vault, "body", meta)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(inbox); err != nil {
		t.Error("inbox should have been created")
	}
}
