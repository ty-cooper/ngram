package pipeline

import (
	"fmt"
	"strings"

	"github.com/ty-cooper/ngram/internal/taxonomy"
)

// StructuredNoteSchema is the JSON schema passed to claude --json-schema.
const StructuredNoteSchema = `{
  "type": "object",
  "required": ["title", "summary", "body", "content_type", "domain", "topic_cluster", "tags"],
  "properties": {
    "title": {
      "type": "string",
      "description": "Concise descriptive title generated from the content"
    },
    "summary": {
      "type": "string",
      "description": "One line summary, under 120 characters"
    },
    "body": {
      "type": "string",
      "description": "Structured markdown content"
    },
    "content_type": {
      "type": "string",
      "enum": ["knowledge", "reference", "log", "link", "media"]
    },
    "domain": {
      "type": "string",
      "description": "Knowledge domain"
    },
    "topic_cluster": {
      "type": "string",
      "description": "Specific topic within the domain"
    },
    "tags": {
      "type": "array",
      "items": { "type": "string" }
    }
  },
  "additionalProperties": false
}`

// StructuringSystemPrompt is passed via --system-prompt to hard-enforce JSON output.
const StructuringSystemPrompt = `You are a note structuring engine. You receive raw text or screenshot descriptions and output a single JSON object. You NEVER output anything except valid JSON. No commentary. No explanation. No preamble. No markdown fences. No trailing text. Your entire response must be parseable by JSON.parse().

The JSON object must have these fields:
{
  "title": "concise descriptive title",
  "summary": "one line, under 120 chars",
  "body": "structured markdown content",
  "content_type": "knowledge|reference|log|link|media",
  "domain": "knowledge domain",
  "topic_cluster": "specific topic within domain",
  "tags": ["tag1", "tag2"]
}

WRITING RULES for body content:
- Google developer documentation style
- Declarative voice. Present tense. Active voice
- No em dashes
- No filler words: basically, essentially, actually, interestingly, simply, just, really
- No weak starters: "It is", "There are", "This is", "Note that"
- Minimum words for maximum information transfer
- All commands in fenced code blocks with language identifier

CONTENT TYPE RULES:
- knowledge: study notes, concepts, explanations (gets quizzed)
- reference: checklists, configs, recipes, bookmarks (not quizzed)
- log: engagement logs, command output, findings (not quizzed)
- link: saved URLs with description (not quizzed)
- media: screenshots, images with description (not quizzed)

If the input is trivial (a single word, typo, or test), still produce valid JSON with your best interpretation.`

// BuildStructuringPrompt creates the user prompt sent to Claude.
// The system prompt (StructuringSystemPrompt) enforces JSON output format.
func BuildStructuringPrompt(tax *taxonomy.Taxonomy, rawContent string) string {
	var b strings.Builder

	b.WriteString("Structure this raw note into a JSON object. Add missing context ONLY if crucial. Keep it atomic.\n\n")

	// Inject taxonomy.
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
	b.WriteString("\n\nFix all violations and return corrected JSON.")

	return b.String()
}
