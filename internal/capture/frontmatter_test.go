package capture

import (
	"strings"
	"testing"
	"time"

	"github.com/ty-cooper/ngram/internal/config"
)

func TestBuildFrontmatter_Terminal(t *testing.T) {
	meta := NoteMetadata{
		Title:  "test note",
		Source: "terminal",
		Time:   time.Date(2026, 3, 22, 10, 42, 0, 0, time.UTC),
	}

	got := BuildFrontmatter(meta)

	if !strings.HasPrefix(got, "---\n") {
		t.Error("frontmatter should start with ---")
	}
	if !strings.HasSuffix(got, "---\n") {
		t.Error("frontmatter should end with ---")
	}
	if !strings.Contains(got, `source: "terminal"`) {
		t.Error("should contain source: terminal")
	}
	if !strings.Contains(got, `title: "test note"`) {
		t.Error("should contain title")
	}
	if strings.Contains(got, "box:") {
		t.Error("should not contain box when no boxrc")
	}
}

func TestBuildFrontmatter_WithBoxRC(t *testing.T) {
	meta := NoteMetadata{
		Title:  "sqli found",
		Source: "terminal",
		Time:   time.Date(2026, 3, 22, 10, 42, 0, 0, time.UTC),
		BoxCtx: &config.BoxContext{
			Box:   "optimum",
			IP:    "10.10.10.8",
			Phase: "exploit",
		},
	}

	got := BuildFrontmatter(meta)

	if !strings.Contains(got, `box: "optimum"`) {
		t.Error("should contain box")
	}
	if !strings.Contains(got, `ip: "10.10.10.8"`) {
		t.Error("should contain ip")
	}
	if !strings.Contains(got, `phase: "exploit"`) {
		t.Error("should contain phase")
	}
}

func TestBuildFrontmatter_CommandCapture(t *testing.T) {
	meta := NoteMetadata{
		Title:   "nmap scan",
		Source:  "command-capture",
		Command: "nmap -sV 10.10.10.8",
		Time:    time.Date(2026, 3, 22, 10, 42, 0, 0, time.UTC),
	}

	got := BuildFrontmatter(meta)

	if !strings.Contains(got, `command: "nmap -sV 10.10.10.8"`) {
		t.Error("should contain command")
	}
	if !strings.Contains(got, `source: "command-capture"`) {
		t.Error("should contain source: command-capture")
	}
}
