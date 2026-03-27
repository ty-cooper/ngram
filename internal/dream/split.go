package dream

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"strings"
)

// SplitNote represents one atomic note from a split operation.
type SplitNote struct {
	Title string `json:"title" jsonschema:"description=Title for this atomic note,required=true"`
	Body  string `json:"body" jsonschema:"description=Complete markdown body for this note,required=true"`
}

// SplitResponse is returned by the LLM when splitting oversized notes.
type SplitResponse struct {
	Notes []SplitNote `json:"notes" jsonschema:"description=Atomic notes split from the original. Each must be under 500 words and self-contained.,required=true"`
}

// splitPass finds notes over the word limit and splits them into atomic notes.
func (r *Runner) splitPass(ctx context.Context, results []LintResult) []Action {
	var actions []Action

	for _, result := range results {
		// Only process over-word-limit violations.
		hasWordLimit := false
		for _, v := range result.Violations {
			if v.Rule == "over-word-limit" {
				hasWordLimit = true
				break
			}
		}
		if !hasWordLimit {
			continue
		}

		data, err := os.ReadFile(result.Note.Path)
		if err != nil {
			log.Printf("dream: split — can't read %s: %v", result.Note.ID, err)
			continue
		}

		body := extractBody(string(data))
		if body == "" {
			continue
		}

		prompt := fmt.Sprintf(`This note is too long for an atomic knowledge base (over 500 words). Split it into multiple atomic notes.

RULES:
- Each note must contain ONE concept and be self-contained (a stranger should understand it).
- Each note must be under 500 words.
- Preserve ALL information — nothing should be lost.
- Preserve all code blocks, commands, and technical details exactly.
- Code blocks MUST have a # [Tool/Context] comment as the first line.
- Use {{PLACEHOLDER}} syntax for variable values in commands.
- Each note gets its own title (# Heading) and summary (*italic line*).
- Add a ## Related section at the bottom of each note referencing the other split notes by title.

ORIGINAL NOTE (ID: %s, Title: %s):
%s

Return the split as JSON.`, result.Note.ID, result.Note.Title, body)

		var resp SplitResponse
		if err := r.LLM.Instruct(ctx, prompt, &resp); err != nil {
			log.Printf("dream: split LLM failed for %s: %v", result.Note.ID, err)
			continue
		}

		if len(resp.Notes) < 2 {
			log.Printf("dream: split for %s returned %d notes, skipping", result.Note.ID, len(resp.Notes))
			continue
		}

		actions = append(actions, Action{
			Type:    "split",
			NoteIDs: []string{result.Note.ID},
			Reason:  fmt.Sprintf("note was %d words, split into %d atomic notes", wordCount(body), len(resp.Notes)),
			SplitNotes: resp.Notes,
		})
	}

	return actions
}

func extractBody(content string) string {
	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content[4:], "\n---\n", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[1])
		}
	}
	return strings.TrimSpace(content)
}

func generateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
