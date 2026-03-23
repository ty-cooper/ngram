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
const StructuringSystemPrompt = `You are a note classifier. You receive raw text and output metadata for it. Do NOT rewrite, rephrase, or expand the input. The body field is the original input cleaned up for formatting only (fix typos, add markdown structure). Preserve the user's words.

RULES:
- body = the original note, cleaned up for formatting only. Do not add content.
- title = short label derived from the content
- summary = one line, under 120 chars
- Classify into domain, topic_cluster, tags, content_type
- Commands go in fenced code blocks

CONTENT TYPES:
- knowledge: concepts, explanations (quizzed)
- reference: checklists, configs, recipes (not quizzed)
- log: command output, findings (not quizzed)
- link: URLs (not quizzed)
- media: screenshots (not quizzed)`

// BuildStructuringPrompt creates the user prompt sent to Claude.
func BuildStructuringPrompt(tax *taxonomy.Taxonomy, rawContent string) string {
	var b strings.Builder

	b.WriteString("Classify this note and clean up formatting. Do not rewrite or rephrase the content.\n\n")

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
