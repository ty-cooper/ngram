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
func ParseStructuredJSON(data []byte) (*StructuredNote, error) {
	data = stripCodeFences(data)

	// Check if this is a Claude Code envelope.
	var envelope struct {
		Type   string          `json:"type"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Type == "result" && len(envelope.Result) > 0 {
		// Result may be a JSON string or a JSON object.
		var resultStr string
		if err := json.Unmarshal(envelope.Result, &resultStr); err == nil {
			// It's a string — parse the inner JSON.
			data = stripCodeFences([]byte(resultStr))
		} else {
			// It's already a JSON object.
			data = []byte(envelope.Result)
		}
	}

	var note StructuredNote
	if err := json.Unmarshal(data, &note); err != nil {
		return nil, fmt.Errorf("parse structured json: %w", err)
	}
	if note.Body == "" && note.Title == "" {
		return nil, fmt.Errorf("structured note: empty response from LLM")
	}
	// If no title, derive from body (first 60 chars).
	if note.Title == "" {
		note.Title = deriveTitle(note.Body)
	}
	// If no body, use the title as body.
	if note.Body == "" {
		note.Body = note.Title
	}
	if note.ContentType == "" {
		note.ContentType = "knowledge"
	}
	return &note, nil
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

// BuildFrontmatter generates the full COO-83 YAML frontmatter for a ProcessedNote.
func BuildFrontmatter(n *ProcessedNote) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %q\n", n.ID)
	fmt.Fprintf(&b, "title: %q\n", n.Title)
	fmt.Fprintf(&b, "content_type: %q\n", n.ContentType)
	fmt.Fprintf(&b, "created: %q\n", n.Created.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "modified: %q\n", n.Created.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "source: %q\n", n.Source)

	if n.Domain != "" {
		fmt.Fprintf(&b, "domain: %q\n", n.Domain)
	}
	if n.TopicCluster != "" {
		fmt.Fprintf(&b, "topic_cluster: %q\n", n.TopicCluster)
	}

	if len(n.Tags) > 0 {
		b.WriteString("tags:\n")
		for _, tag := range n.Tags {
			fmt.Fprintf(&b, "  - %s\n", tag)
		}
	}

	if n.Box != "" {
		fmt.Fprintf(&b, "box: %q\n", n.Box)
	}
	if n.Phase != "" {
		fmt.Fprintf(&b, "phase: %q\n", n.Phase)
	}

	b.WriteString("related: []\n")

	// Retention block per COO-83.
	if n.ContentType == "knowledge" {
		b.WriteString("retention:\n")
		b.WriteString("  state: new\n")
		b.WriteString("  ease_factor: 2.5\n")
		b.WriteString("  interval_days: 0\n")
		b.WriteString("  repetition_count: 0\n")
		b.WriteString("  lapse_count: 0\n")
		b.WriteString("  streak: 0\n")
		b.WriteString("  total_reviews: 0\n")
		b.WriteString("  total_correct: 0\n")
		b.WriteString("  retention_score: 0\n")
		fmt.Fprintf(&b, "  next_review: %q\n", n.Created.Format("2006-01-02"))
		b.WriteString("  last_reviewed: null\n")
	}

	b.WriteString("---\n")
	return b.String()
}

// BuildNoteContent returns the complete markdown: frontmatter + body + properties below page break.
func BuildNoteContent(n *ProcessedNote) string {
	var b strings.Builder
	b.WriteString(BuildFrontmatter(n))
	b.WriteString("\n")
	if n.Summary != "" {
		fmt.Fprintf(&b, "*%s*\n\n", n.Summary)
	}
	b.WriteString(n.Body)
	b.WriteString("\n")

	// Readable properties below page break.
	b.WriteString("\n---\n\n")
	if n.Domain != "" {
		fmt.Fprintf(&b, "**Domain:** %s", n.Domain)
		if n.TopicCluster != "" {
			fmt.Fprintf(&b, " / %s", n.TopicCluster)
		}
		b.WriteString("\n")
	}
	if len(n.Tags) > 0 {
		fmt.Fprintf(&b, "**Tags:** %s\n", strings.Join(n.Tags, ", "))
	}
	fmt.Fprintf(&b, "**Type:** %s\n", n.ContentType)
	if n.Box != "" {
		fmt.Fprintf(&b, "**Box:** %s", n.Box)
		if n.Phase != "" {
			fmt.Fprintf(&b, " / %s", n.Phase)
		}
		b.WriteString("\n")
	}

	return b.String()
}
