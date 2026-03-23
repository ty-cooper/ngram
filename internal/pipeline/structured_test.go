package pipeline

import (
	"strings"
	"testing"
	"time"
)

func TestParseStructuredJSON(t *testing.T) {
	data := []byte(`{
		"title": "Raft Leader Election",
		"summary": "How Raft elects a leader.",
		"body": "When a follower timeout fires, it becomes a candidate.",
		"content_type": "knowledge",
		"domain": "distributed-systems",
		"topic_cluster": "consensus",
		"tags": ["raft", "consensus"]
	}`)

	note, err := ParseStructuredJSON(data)
	if err != nil {
		t.Fatalf("ParseStructuredJSON: %v", err)
	}
	if note.Title != "Raft Leader Election" {
		t.Errorf("Title = %q", note.Title)
	}
	if note.ContentType != "knowledge" {
		t.Errorf("ContentType = %q", note.ContentType)
	}
	if len(note.Tags) != 2 {
		t.Errorf("Tags = %v", note.Tags)
	}
}

func TestParseStructuredJSON_MissingTitle(t *testing.T) {
	data := []byte(`{"body": "some content"}`)
	_, err := ParseStructuredJSON(data)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestParseStructuredJSON_DefaultContentType(t *testing.T) {
	data := []byte(`{"title": "Test", "body": "Content"}`)
	note, err := ParseStructuredJSON(data)
	if err != nil {
		t.Fatalf("ParseStructuredJSON: %v", err)
	}
	if note.ContentType != "knowledge" {
		t.Errorf("expected default content_type 'knowledge', got %q", note.ContentType)
	}
}

func TestBuildFrontmatter_Knowledge(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title:        "Test Note",
			ContentType:  "knowledge",
			Domain:       "pentest",
			TopicCluster: "recon",
			Tags:         []string{"nmap", "scanning"},
			Summary:      "A test note.",
			Body:         "Body content.",
		},
		ID:      "a1b2c3d4",
		Source:  "terminal",
		Created: time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
	}

	fm := BuildFrontmatter(note)

	checks := []string{
		`id: "a1b2c3d4"`,
		`title: "Test Note"`,
		`content_type: "knowledge"`,
		`domain: "pentest"`,
		`topic_cluster: "recon"`,
		"  - nmap",
		"  - scanning",
		"retention:",
		"  state: new",
		"  ease_factor: 2.5",
		"related: []",
	}
	for _, c := range checks {
		if !strings.Contains(fm, c) {
			t.Errorf("frontmatter missing %q\n\ngot:\n%s", c, fm)
		}
	}
}

func TestBuildFrontmatter_NonKnowledge(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title:       "Useful Link",
			ContentType: "link",
			Body:        "https://example.com",
		},
		ID:      "e5f6a7b8",
		Source:  "terminal",
		Created: time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC),
	}

	fm := BuildFrontmatter(note)

	if strings.Contains(fm, "retention:") {
		t.Error("non-knowledge notes should not have retention block")
	}
}

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if len(id) != 8 {
		t.Errorf("expected 8-char ID, got %q (len %d)", id, len(id))
	}

	// Should be unique.
	id2 := GenerateID()
	if id == id2 {
		t.Error("two generated IDs should not be equal")
	}
}

func TestBuildNoteContent(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title:       "Test",
			ContentType: "knowledge",
			Summary:     "A summary.",
			Body:        "The body.",
		},
		ID:      "abcd1234",
		Source:  "terminal",
		Created: time.Now(),
	}

	content := BuildNoteContent(note)
	if !strings.Contains(content, "---\n") {
		t.Error("missing frontmatter delimiters")
	}
	if !strings.Contains(content, "*A summary.*") {
		t.Error("missing summary line")
	}
	if !strings.Contains(content, "The body.") {
		t.Error("missing body")
	}
}
