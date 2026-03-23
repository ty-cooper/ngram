package pipeline

import (
	"fmt"
	"strings"

	"github.com/ty-cooper/ngram/internal/taxonomy"
)

// BuildStructuringPrompt creates the prompt sent to Claude for structuring a raw note.
func BuildStructuringPrompt(tax *taxonomy.Taxonomy, rawContent string) string {
	var b strings.Builder

	b.WriteString("You are a knowledge structuring agent for the Ngram system.\n\n")
	b.WriteString("Structure the following raw note into a clean, atomic knowledge note.\n\n")

	b.WriteString("WRITING RULES (non negotiable):\n")
	b.WriteString("- Zero personality. Declarative voice. Present tense. Active voice.\n")
	b.WriteString("- No hyphens or em dashes anywhere in the output.\n")
	b.WriteString("- No filler words: basically, essentially, actually, interestingly, in order to, simply, just, really, important, key, crucial, significant.\n")
	b.WriteString("- No weak starters: \"It is\", \"There are\", \"There is\", \"This is\", \"Note that\".\n")
	b.WriteString("- Minimum words for maximum information transfer.\n")
	b.WriteString("- One concept per sentence. One topic per paragraph.\n")
	b.WriteString("- All commands in fenced code blocks with language identifier.\n")
	b.WriteString("- Summary must be under 120 characters.\n\n")

	// Inject taxonomy.
	if domains := tax.CanonicalDomainList(); len(domains) > 0 {
		fmt.Fprintf(&b, "CANONICAL DOMAINS: %s\n", strings.Join(domains, ", "))
		b.WriteString("Use one of these domains if the content matches. Propose a new domain only if none fit.\n\n")
	}
	if tags := tax.CanonicalTagList(); len(tags) > 0 {
		fmt.Fprintf(&b, "CANONICAL TAGS: %s\n", strings.Join(tags, ", "))
		b.WriteString("Use canonical tags when applicable. You may propose new tags.\n\n")
	}

	b.WriteString("CONTENT TYPE RULES:\n")
	b.WriteString("- knowledge: study notes, concepts, explanations (gets quizzed)\n")
	b.WriteString("- reference: checklists, configs, recipes, bookmarks (not quizzed)\n")
	b.WriteString("- log: engagement logs, command output, findings (not quizzed)\n")
	b.WriteString("- link: saved URLs with description (not quizzed)\n")
	b.WriteString("- media: screenshots, images with description (not quizzed)\n\n")

	b.WriteString("Return ONLY valid JSON matching this schema:\n")
	b.WriteString("```json\n")
	b.WriteString(`{
  "title": "concise descriptive title (generate from content, never empty)",
  "summary": "one line, under 120 chars",
  "body": "structured markdown content",
  "content_type": "knowledge|reference|log|link|media",
  "domain": "domain name",
  "topic_cluster": "specific topic within domain",
  "tags": ["tag1", "tag2"]
}`)
	b.WriteString("\n```\n\n")

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
