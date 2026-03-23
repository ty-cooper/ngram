package pipeline

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// StructuredNote is the JSON schema Claude returns after structuring.
type StructuredNote struct {
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Body         string   `json:"body"`
	ContentType  string   `json:"content_type"`
	Domain       string   `json:"domain"`
	TopicCluster string   `json:"topic_cluster"`
	Tags         []string `json:"tags"`
}

// ProcessedNote is the fully resolved note ready for writing.
type ProcessedNote struct {
	StructuredNote
	ID      string
	Source  string
	Box     string
	Phase   string
	Created time.Time
}

// GenerateID returns a random 8-char hex string.
func GenerateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ParseStructuredJSON parses Claude's JSON response into a StructuredNote.
// Handles both raw JSON and the Claude Code envelope {"type":"result","result":"..."}.
// ParseStructuredNotes parses the {"notes": [...]} response from the API.
// Returns one or more atomic notes.
func ParseStructuredNotes(data []byte) ([]*StructuredNote, error) {
	data = stripCodeFences(data)

	var wrapper struct {
		Notes []StructuredNote `json:"notes"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("parse structured json: %w", err)
	}
	if len(wrapper.Notes) == 0 {
		return nil, fmt.Errorf("structured notes: empty response from LLM")
	}

	var result []*StructuredNote
	for i := range wrapper.Notes {
		note := &wrapper.Notes[i]
		if note.Title == "" {
			note.Title = deriveTitle(note.Body)
		}
		if note.Body == "" {
			note.Body = note.Title
		}
		if note.ContentType == "" {
			note.ContentType = "knowledge"
		}
		result = append(result, note)
	}
	return result, nil
}

// ParseStructuredJSON parses a single note. Kept for backward compat with tests.
func ParseStructuredJSON(data []byte) (*StructuredNote, error) {
	notes, err := ParseStructuredNotes(data)
	if err != nil {
		// Fall back to single-note parse for old format.
		data = stripCodeFences(data)
		var note StructuredNote
		if err2 := json.Unmarshal(data, &note); err2 != nil {
			return nil, fmt.Errorf("parse structured json: %w", err2)
		}
		if note.Title == "" {
			note.Title = deriveTitle(note.Body)
		}
		if note.ContentType == "" {
			note.ContentType = "knowledge"
		}
		return &note, nil
	}
	return notes[0], nil
}

// deriveTitle creates a title from body content.
func deriveTitle(body string) string {
	// Take first line or first 60 chars.
	line := body
	if idx := strings.IndexByte(body, '\n'); idx > 0 {
		line = body[:idx]
	}
	line = strings.TrimSpace(line)
	if len(line) > 60 {
		// Cut at last space before 60.
		if i := strings.LastIndex(line[:60], " "); i > 0 {
			line = line[:i]
		} else {
			line = line[:60]
		}
	}
	return line
}

// BuildFrontmatter generates Zettelkasten YAML frontmatter.
func BuildFrontmatter(n *ProcessedNote) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", n.ID)
	fmt.Fprintf(&b, "title: %q\n", n.Title)
	fmt.Fprintf(&b, "created: %s\n", n.Created.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "type: %s\n", n.ContentType)

	if len(n.Tags) > 0 {
		b.WriteString("tags:\n")
		for _, tag := range n.Tags {
			fmt.Fprintf(&b, "  - %s\n", tag)
		}
	}

	if n.Box != "" {
		fmt.Fprintf(&b, "box: %s\n", n.Box)
	}
	if n.Phase != "" {
		fmt.Fprintf(&b, "phase: %s\n", n.Phase)
	}

	// Retention for quizzable notes.
	if n.ContentType == "knowledge" {
		b.WriteString("retention:\n")
		b.WriteString("  state: new\n")
		b.WriteString("  ease_factor: 2.5\n")
		b.WriteString("  interval_days: 0\n")
		fmt.Fprintf(&b, "  next_review: %s\n", n.Created.Format("2006-01-02"))
	}

	b.WriteString("---\n")
	return b.String()
}

// BuildNoteContent returns Zettelkasten markdown: frontmatter + title + summary + body + tags.
func BuildNoteContent(n *ProcessedNote) string {
	var b strings.Builder
	b.WriteString(BuildFrontmatter(n))
	b.WriteString("\n")

	// Title as H1.
	fmt.Fprintf(&b, "# %s\n\n", n.Title)

	if n.Summary != "" {
		fmt.Fprintf(&b, "*%s*\n\n", n.Summary)
	}

	b.WriteString(n.Body)
	b.WriteString("\n")

	// Tags and links below page break.
	b.WriteString("\n---\n\n")
	if len(n.Tags) > 0 {
		var tagLinks []string
		for _, tag := range n.Tags {
			tagLinks = append(tagLinks, fmt.Sprintf("#%s", tag))
		}
		fmt.Fprintf(&b, "%s\n", strings.Join(tagLinks, " "))
	}

	return b.String()
}
