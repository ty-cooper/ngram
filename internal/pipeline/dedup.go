package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ty-cooper/ngram/internal/llm"
	"github.com/ty-cooper/ngram/internal/search"
	"github.com/ty-cooper/ngram/internal/vault"
)

const dedupThreshold = 0.75 // Lowered for hybrid (semantic + keyword) score distribution

// DedupDecision is the master agent's decision.
type DedupDecision struct {
	Decision       string   `json:"decision"`        // "append", "new", "duplicate"
	TargetNoteID   string   `json:"target_note_id"`  // for append/duplicate
	Reason         string   `json:"reason"`
	AppendContent  string   `json:"append_content"`  // for append
	LinkSuggestions []string `json:"link_suggestions"`
}

// DedupResult is what the dedup check returns to the processor.
type DedupResult struct {
	Action   string // "proceed", "appended", "duplicate"
	Reason   string
	TargetID string // ID of existing note (for append/duplicate)
}

// Deduplicator checks new notes against existing vault content.
type Deduplicator struct {
	VaultPath    string
	SearchClient *search.Client
	Runner       *llm.Runner
}

// Check runs the dedup pipeline for a structured note.
// Returns a DedupResult indicating what action was taken.
// On any error, defaults to "proceed" (never lose a note).
func (d *Deduplicator) Check(ctx context.Context, note *StructuredNote, procPath string) *DedupResult {
	if d.SearchClient == nil {
		return &DedupResult{Action: "proceed", Reason: "no search client"}
	}

	// Build search query from note content.
	query := note.Title + " " + note.Summary
	if len(query) < 20 {
		query += " " + truncateStr(note.Body, 200)
	}

	similar, err := d.SearchClient.FindSimilar(query, 5)
	if err != nil {
		log.Printf("warn: dedup search failed: %v", err)
		return &DedupResult{Action: "proceed", Reason: "search failed"}
	}

	// Check if any result crosses the threshold.
	var aboveThreshold []search.SimilarNote
	for _, s := range similar {
		if s.Score >= dedupThreshold {
			aboveThreshold = append(aboveThreshold, s)
		}
	}

	if len(aboveThreshold) == 0 {
		return &DedupResult{Action: "proceed", Reason: "no similar notes above threshold"}
	}

	// Call master agent for decision.
	decision, err := d.callMasterAgent(ctx, note, aboveThreshold)
	if err != nil {
		log.Printf("warn: dedup master agent failed: %v, defaulting to NEW", err)
		return &DedupResult{Action: "proceed", Reason: "master agent failed"}
	}

	switch decision.Decision {
	case "duplicate":
		d.handleDuplicate(note, decision, procPath)
		return &DedupResult{
			Action:   "duplicate",
			Reason:   decision.Reason,
			TargetID: decision.TargetNoteID,
		}

	case "append":
		if err := d.handleAppend(note, decision); err != nil {
			log.Printf("warn: append failed: %v, defaulting to NEW", err)
			return &DedupResult{Action: "proceed", Reason: "append failed"}
		}
		return &DedupResult{
			Action:   "appended",
			Reason:   decision.Reason,
			TargetID: decision.TargetNoteID,
		}

	default:
		return &DedupResult{Action: "proceed", Reason: decision.Reason}
	}
}

func (d *Deduplicator) callMasterAgent(ctx context.Context, note *StructuredNote, similar []search.SimilarNote) (*DedupDecision, error) {
	prompt := buildDedupPrompt(note, similar)

	out, err := d.Runner.Run(ctx, prompt)
	if err != nil {
		return nil, err
	}

	out = stripCodeFences(out)

	var decision DedupDecision
	if err := json.Unmarshal(out, &decision); err != nil {
		return nil, fmt.Errorf("parse dedup decision: %w", err)
	}

	if decision.Decision == "" {
		return nil, fmt.Errorf("empty decision from master agent")
	}

	return &decision, nil
}

func buildDedupPrompt(note *StructuredNote, similar []search.SimilarNote) string {
	var b strings.Builder

	b.WriteString("You are a knowledge deduplication agent for the Ngram knowledge system.\n\n")
	b.WriteString("NEW NOTE:\n")
	fmt.Fprintf(&b, "Title: %s\nDomain: %s\nSummary: %s\n\n%s\n\n", note.Title, note.Domain, note.Summary, note.Body)

	b.WriteString("EXISTING SIMILAR NOTES:\n\n")
	for i, s := range similar {
		fmt.Fprintf(&b, "--- Note %d (ID: %s, Score: %.2f) ---\n", i+1, s.ID, s.Score)
		fmt.Fprintf(&b, "Title: %s\nDomain: %s\nSummary: %s\n\n%s\n\n", s.Title, s.Domain, s.Summary, s.Body)
	}

	b.WriteString(`Decide ONE of:

APPEND — the new content adds detail to an existing note. Specify which note (by ID) and what content to add.

NEW — the content is genuinely new, not covered by existing notes.

DUPLICATE — the content is already fully captured in an existing note. Specify which note it duplicates and why.

Decision matrix:
- Similarity >= 0.90 AND same domain/context → DUPLICATE
- Similarity >= 0.75 AND related content → APPEND to existing note
- Similarity >= 0.75 AND different angle → NEW (keep both)

Return ONLY valid JSON:
{"decision": "append|new|duplicate", "target_note_id": "id", "reason": "...", "append_content": "...", "link_suggestions": ["id1"]}
`)

	return b.String()
}

func (d *Deduplicator) handleDuplicate(note *StructuredNote, decision *DedupDecision, procPath string) {
	// Log to dedup-log.jsonl.
	metaDir := filepath.Join(d.VaultPath, "_meta")
	vault.EnsureDir(metaDir)

	entry := map[string]interface{}{
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"raw_input":    note.Title,
		"duplicate_of": decision.TargetNoteID,
		"reason":       decision.Reason,
	}
	data, _ := json.Marshal(entry)

	f, err := os.OpenFile(filepath.Join(metaDir, "dedup-log.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		defer f.Close()
		f.Write(data)
		f.Write([]byte("\n"))
	}

	log.Printf("ngram: duplicate of %s — %s", decision.TargetNoteID, decision.Reason)
}

func (d *Deduplicator) handleAppend(note *StructuredNote, decision *DedupDecision) error {
	if decision.TargetNoteID == "" {
		return fmt.Errorf("no target note ID for append")
	}

	// Find the target note file.
	targetPath, err := d.findNoteByID(decision.TargetNoteID)
	if err != nil {
		return err
	}

	// Read existing content.
	existing, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("read target: %w", err)
	}

	content := string(existing)
	appendText := decision.AppendContent
	if appendText == "" {
		appendText = note.Body
	}

	// Append new content before any Links section.
	if idx := strings.Index(content, "\n## Links"); idx >= 0 {
		content = content[:idx] + "\n\n" + appendText + content[idx:]
	} else {
		content = strings.TrimRight(content, "\n") + "\n\n" + appendText + "\n"
	}

	// Update modified timestamp in frontmatter.
	content = updateModifiedTimestamp(content)

	// Write back atomically.
	if err := atomicWrite(targetPath, content); err != nil {
		return fmt.Errorf("write appended: %w", err)
	}

	// Re-index the updated note.
	if d.SearchClient != nil {
		doc, err := search.ParseNoteFile(targetPath, d.VaultPath)
		if err == nil {
			d.SearchClient.IndexNote(*doc)
		}
	}

	log.Printf("ngram: appended to %s — %s", decision.TargetNoteID, decision.Reason)
	return nil
}

func (d *Deduplicator) findNoteByID(id string) (string, error) {
	var found string
	filepath.Walk(d.VaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, id+"-") || base == id+".md" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("note %s not found in vault", id)
	}
	return found, nil
}

func updateModifiedTimestamp(content string) string {
	lines := strings.Split(content, "\n")
	now := time.Now().UTC().Format(time.RFC3339)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "modified:") {
			lines[i] = fmt.Sprintf("modified: %q", now)
			break
		}
	}
	return strings.Join(lines, "\n")
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
