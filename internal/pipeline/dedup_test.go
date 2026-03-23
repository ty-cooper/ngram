package pipeline

import (
	"strings"
	"testing"

	"github.com/ty-cooper/ngram/internal/search"
)

func TestBuildDedupPrompt(t *testing.T) {
	note := &StructuredNote{
		Title:   "Raft Leader Election",
		Domain:  "distributed-systems",
		Summary: "How Raft elects leaders.",
		Body:    "Election timeout fires, candidate requests votes.",
	}

	similar := []search.SimilarNote{
		{
			ID:      "existing1",
			Title:   "Raft Consensus",
			Domain:  "distributed-systems",
			Summary: "Raft consensus overview.",
			Body:    "Raft is a consensus algorithm.",
			Score:   0.92,
		},
	}

	prompt := buildDedupPrompt(note, similar)

	if !containsString(prompt, "Raft Leader Election") {
		t.Error("prompt missing new note title")
	}
	if !containsString(prompt, "existing1") {
		t.Error("prompt missing similar note ID")
	}
	if !containsString(prompt, "0.92") {
		t.Error("prompt missing similarity score")
	}
	if !containsString(prompt, "DUPLICATE") {
		t.Error("prompt missing decision options")
	}
}

func TestUpdateModifiedTimestamp(t *testing.T) {
	content := "---\nid: \"abc\"\nmodified: \"2026-01-01T00:00:00Z\"\n---\nBody."
	updated := updateModifiedTimestamp(content)
	if containsString(updated, "2026-01-01") {
		t.Error("modified timestamp should have been updated")
	}
}

func TestTruncateStr(t *testing.T) {
	if got := truncateStr("hello world", 5); got != "hello" {
		t.Errorf("truncateStr = %q", got)
	}
	if got := truncateStr("hi", 5); got != "hi" {
		t.Errorf("truncateStr = %q", got)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(s, substr)
}
