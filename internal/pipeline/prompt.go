package pipeline

import (
	"fmt"
	"strings"

	"github.com/ty-cooper/ngram/internal/taxonomy"
)

// NoteJSONSchema is the schema enforced by Anthropic's OutputConfig.
// The API guarantees every response matches this exactly.
var NoteJSONSchema = map[string]any{
	"type":     "object",
	"required": []string{"title", "summary", "body", "content_type", "domain", "topic_cluster", "tags"},
	"properties": map[string]any{
		"title":         map[string]any{"type": "string", "description": "Concise descriptive title"},
		"summary":       map[string]any{"type": "string", "description": "One line summary, under 120 characters"},
		"body":          map[string]any{"type": "string", "description": "Structured markdown content in Google developer docs style"},
		"content_type":  map[string]any{"type": "string", "enum": []string{"knowledge", "reference", "log", "link", "media"}},
		"domain":        map[string]any{"type": "string", "description": "Knowledge domain"},
		"topic_cluster": map[string]any{"type": "string", "description": "Specific topic within the domain"},
		"tags":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
	"additionalProperties": false,
}

// StructuringSystemPrompt guides Claude on how to structure notes.
// JSON format is enforced by the schema, not the prompt.
const StructuringSystemPrompt = `You are a note structuring engine. You receive raw text or screenshots and produce a structured knowledge note.

Make sense of what the user was attempting to say. Add missing context ONLY if that context is crucial to understanding the note. Keep the note atomic.

WRITING RULES for body content (non negotiable):
- Google developer documentation style
- Declarative voice. Present tense. Active voice
- No em dashes anywhere
- No filler: basically, essentially, actually, interestingly, simply, just, really, important, key, crucial, significant
- No weak starters: "It is", "There are", "This is", "Note that"
- Minimum words for maximum information transfer
- One concept per sentence. One topic per paragraph
- All commands in fenced code blocks with language identifier
- Summary must be under 120 characters

CONTENT TYPE RULES:
- knowledge: study notes, concepts, explanations (gets quizzed)
- reference: checklists, configs, recipes, bookmarks (not quizzed)
- log: engagement logs, command output, findings (not quizzed)
- link: saved URLs with description (not quizzed)
- media: screenshots, images with description (not quizzed)

If the input is trivial (a single word, typo, or test), still produce a valid note with your best interpretation.`

// BuildStructuringPrompt creates the user prompt sent to Claude.
func BuildStructuringPrompt(tax *taxonomy.Taxonomy, rawContent string) string {
	var b strings.Builder

	b.WriteString("Structure this raw note. Add missing context ONLY if crucial. Keep it atomic.\n\n")

	if domains := tax.CanonicalDomainList(); len(domains) > 0 {
		fmt.Fprintf(&b, "CANONICAL DOMAINS: %s\n", strings.Join(domains, ", "))
		b.WriteString("Use one of these if the content matches. Propose a new domain only if none fit.\n\n")
	}
	if tags := tax.CanonicalTagList(); len(tags) > 0 {
		fmt.Fprintf(&b, "CANONICAL TAGS: %s\n", strings.Join(tags, ", "))
		b.WriteString("Use canonical tags when applicable. You may propose new tags.\n\n")
	}

	b.WriteString("RAW NOTE:\n")
	b.WriteString(rawContent)

	return b.String()
}

// BuildRetryPrompt creates a prompt that includes previous violations.
func BuildRetryPrompt(tax *taxonomy.Taxonomy, rawContent string, violations []Violation, previous *StructuredNote) string {
	base := BuildStructuringPrompt(tax, rawContent)

	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\nYOUR PREVIOUS OUTPUT HAD THESE VIOLATIONS:\n")
	b.WriteString(FormatViolations(violations))
	b.WriteString("\n\nFix all violations.")

	return b.String()
}
