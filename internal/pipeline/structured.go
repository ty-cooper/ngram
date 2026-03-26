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
	Title        string   `json:"title" jsonschema:"description=Concise descriptive title,required=true"`
	Summary      string   `json:"summary" jsonschema:"description=One line summary under 120 characters,required=true"`
	Body         string   `json:"body" jsonschema:"description=Structured markdown body with copyable command blocks using PLACEHOLDER syntax,required=true"`
	ContentType  string   `json:"content_type" jsonschema:"description=Note type,enum=knowledge,enum=reference,enum=log,enum=link,enum=media,required=true"`
	Domain       string   `json:"domain,omitempty" jsonschema:"description=Knowledge domain"`
	TopicCluster string   `json:"topic_cluster,omitempty" jsonschema:"description=Topic cluster within domain"`
	Tags         []string `json:"tags" jsonschema:"description=Tags for organization. Max 5. Prefer existing tags.,required=true,maxItems=5"`
}

// StructuredNotesResponse is the top-level response from the structuring LLM call.
type StructuredNotesResponse struct {
	Notes         []StructuredNote `json:"notes" jsonschema:"description=Atomic notes extracted from input"`
	Discard       bool             `json:"discard,omitempty" jsonschema:"description=True if input is empty or junk with no extractable knowledge"`
	DiscardReason string           `json:"discard_reason,omitempty" jsonschema:"description=Why input was discarded"`
}

// EvidenceChain tracks provenance for a note.
type EvidenceChain struct {
	SourceCommand string
	CapturedAt    string
	SessionID     string
	ParentNoteID  string
	Screenshots   []string
	SourceFile    string
	Tool          string
}

// ProcessedNote is the fully resolved note ready for writing.
type ProcessedNote struct {
	StructuredNote
	ID       string
	Source   string
	Box      string
	Phase    string
	Created  time.Time
	Evidence EvidenceChain
	Related  []RelatedLink // Bidirectional links to other notes
	From     string        // Parent note ID (lineage tracking)
}

// RelatedLink is a connection to another note.
type RelatedLink struct {
	ID    string
	Title string
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
// ErrDiscard is returned when the LLM marks input as junk.
var ErrDiscard = fmt.Errorf("input discarded by LLM")

// ValidateResponse checks a StructuredNotesResponse for discard or empty results.
// Returns validated note pointers or an error.
func ValidateResponse(resp *StructuredNotesResponse) ([]*StructuredNote, error) {
	if resp.Discard {
		reason := resp.DiscardReason
		if reason == "" {
			reason = "no extractable knowledge"
		}
		return nil, fmt.Errorf("%w: %s", ErrDiscard, reason)
	}
	if len(resp.Notes) == 0 {
		return nil, fmt.Errorf("structured notes: empty response from LLM")
	}

	var result []*StructuredNote
	for i := range resp.Notes {
		note := &resp.Notes[i]
		if note.Title == "" {
			note.Title = deriveTitle(note.Body)
		}
		if note.Body == "" {
			note.Body = note.Title
		}
		if note.ContentType == "" {
			note.ContentType = "knowledge"
		}
		// Hard cap at 5 tags.
		if len(note.Tags) > 5 {
			note.Tags = note.Tags[:5]
		}
		result = append(result, note)
	}
	return result, nil
}

// ParseStructuredNotes parses raw JSON bytes into validated notes.
// Kept for backward compat with code that receives raw []byte.
func ParseStructuredNotes(data []byte) ([]*StructuredNote, error) {
	data = stripCodeFences(data)

	var resp StructuredNotesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse structured json: %w", err)
	}
	return ValidateResponse(&resp)
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

// BuildFrontmatter generates minimal top frontmatter (timestamps only — for pipeline).
func BuildFrontmatter(n *ProcessedNote) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", n.ID)
	fmt.Fprintf(&b, "created: %s\n", n.Created.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "modified: %s\n", n.Created.UTC().Format(time.RFC3339))
	b.WriteString("---\n")
	return b.String()
}

// BuildNoteContent returns Zettelkasten markdown: minimal frontmatter, content first, metadata footer.
func BuildNoteContent(n *ProcessedNote) string {
	var b strings.Builder

	// --- Minimal frontmatter (machine-readable) ---
	b.WriteString(BuildFrontmatter(n))
	b.WriteString("\n")

	// --- Content (the idea) ---
	fmt.Fprintf(&b, "# %s\n\n", n.Title)

	if n.Summary != "" {
		fmt.Fprintf(&b, "*%s*\n\n", n.Summary)
	}

	b.WriteString(n.Body)
	b.WriteString("\n")

	// --- Related notes ---
	if len(n.Related) > 0 {
		b.WriteString("\n## Related\n\n")
		for _, r := range n.Related {
			if r.Title != "" {
				fmt.Fprintf(&b, "- [[%s]] %s\n", r.ID, r.Title)
			} else {
				fmt.Fprintf(&b, "- [[%s]]\n", r.ID)
			}
		}
	}

	// --- Evidence/References ---
	if n.Evidence.SourceCommand != "" || n.Evidence.Tool != "" || n.Evidence.SessionID != "" || len(n.Evidence.Screenshots) > 0 {
		b.WriteString("\n## References\n\n")
		if n.Evidence.SourceCommand != "" {
			fmt.Fprintf(&b, "- Command: `%s`\n", n.Evidence.SourceCommand)
		}
		if n.Evidence.Tool != "" {
			fmt.Fprintf(&b, "- Tool: %s\n", n.Evidence.Tool)
		}
		if n.Evidence.CapturedAt != "" {
			fmt.Fprintf(&b, "- Captured: %s\n", n.Evidence.CapturedAt)
		}
		if n.Evidence.SessionID != "" {
			fmt.Fprintf(&b, "- Session: %s\n", n.Evidence.SessionID)
		}
		for _, ss := range n.Evidence.Screenshots {
			fmt.Fprintf(&b, "- ![[%s]]\n", ss)
		}
	}

	// --- Metadata footer ---
	b.WriteString("\n---\n")
	fmt.Fprintf(&b, "id: %s\n", n.ID)
	fmt.Fprintf(&b, "type: %s\n", n.ContentType)
	if n.Domain != "" {
		fmt.Fprintf(&b, "domain: %s\n", n.Domain)
	}
	if n.TopicCluster != "" {
		fmt.Fprintf(&b, "topic_cluster: %s\n", n.TopicCluster)
	}
	if n.Box != "" {
		fmt.Fprintf(&b, "box: %s\n", n.Box)
	}
	if n.Phase != "" {
		fmt.Fprintf(&b, "phase: %s\n", n.Phase)
	}
	if n.From != "" {
		fmt.Fprintf(&b, "from: %s\n", n.From)
	}
	if len(n.Tags) > 0 {
		var tagLinks []string
		for _, tag := range n.Tags {
			tagLinks = append(tagLinks, "#"+tag)
		}
		fmt.Fprintf(&b, "tags: %s\n", strings.Join(tagLinks, " "))
	}
	if n.ContentType == "knowledge" {
		b.WriteString("retention: new | ef=2.5 | interval=0\n")
	}

	return b.String()
}
