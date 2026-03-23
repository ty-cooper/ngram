package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ty-cooper/ngram/internal/taxonomy"
)

// noteSchema defines a single atomic note.
var noteSchema = map[string]any{
	"type":     "object",
	"required": []string{"title", "summary", "body", "content_type", "domain", "topic_cluster", "tags"},
	"properties": map[string]any{
		"title":         map[string]any{"type": "string", "description": "Concise descriptive title"},
		"summary":       map[string]any{"type": "string", "description": "One line summary, under 120 characters"},
		"body":          map[string]any{"type": "string", "description": "Structured markdown body with copyable command blocks using {{PLACEHOLDER}} syntax"},
		"content_type":  map[string]any{"type": "string", "enum": []string{"knowledge", "reference", "log", "link", "media"}},
		"domain":        map[string]any{"type": "string", "description": "Knowledge domain"},
		"topic_cluster": map[string]any{"type": "string", "description": "Specific topic within the domain"},
		"tags":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
	"additionalProperties": false,
}

// NoteJSONSchema wraps notes in an array so one input can produce multiple atomic notes.
var NoteJSONSchema = map[string]any{
	"type":     "object",
	"required": []string{"notes"},
	"properties": map[string]any{
		"notes": map[string]any{
			"type":  "array",
			"items": noteSchema,
		},
	},
	"additionalProperties": false,
}

// StructuringSystemPrompt guides Claude on how to structure notes.
// JSON format is enforced by the schema, not the prompt.
const StructuringSystemPrompt = `You split raw input into atomic notes. One concept per note. If the input covers one topic, return one note. If it covers five topics, return five notes.

RULES:
- Each note covers ONE concept, technique, or finding
- Preserve the user's original meaning. Do not add opinions or filler.
- Commands MUST be in fenced code blocks with language identifiers
- Replace specific IPs, hostnames, usernames, passwords in commands with {{PLACEHOLDERS}} like {{TARGET_IP}}, {{USERNAME}}, {{PORT}} so they are reusable
- Keep the original specific values in the explanation text, just genericize the command blocks
- Summary under 120 chars
- Google developer docs style: declarative, present tense, no filler

CONTENT TYPES:
- knowledge: concepts, explanations (quizzed)
- reference: checklists, configs, commands (not quizzed)
- log: engagement output, findings (not quizzed)
- link: URLs (not quizzed)
- media: screenshots (not quizzed)`

// BuildStructuringPrompt creates the user prompt sent to Claude.
// vaultPath is used to discover existing domain/cluster folders for consistency.
func BuildStructuringPrompt(tax *taxonomy.Taxonomy, rawContent string, vaultPath ...string) string {
	var b strings.Builder

	b.WriteString("Split this into atomic notes. One concept per note. Genericize commands with {{PLACEHOLDER}} syntax.\n\n")

	if domains := tax.CanonicalDomainList(); len(domains) > 0 {
		fmt.Fprintf(&b, "CANONICAL DOMAINS: %s\n", strings.Join(domains, ", "))
		b.WriteString("Use one of these if the content matches. Propose a new domain only if none fit.\n\n")
	}
	if tags := tax.CanonicalTagList(); len(tags) > 0 {
		fmt.Fprintf(&b, "CANONICAL TAGS: %s\n", strings.Join(tags, ", "))
		b.WriteString("Use canonical tags when applicable. You may propose new tags.\n\n")
	}

	// Inject existing domain/cluster paths so Claude reuses them.
	if len(vaultPath) > 0 && vaultPath[0] != "" {
		if clusters := discoverClusters(vaultPath[0]); len(clusters) > 0 {
			fmt.Fprintf(&b, "EXISTING CLUSTERS (reuse these, do not create near-duplicates):\n%s\n\n", strings.Join(clusters, "\n"))
		}
	}

	b.WriteString("RAW NOTE:\n")
	b.WriteString(rawContent)

	return b.String()
}

// discoverClusters scans knowledge/{domain}/{cluster}/ directories.
func discoverClusters(vaultPath string) []string {
	knowledgeDir := filepath.Join(vaultPath, "knowledge")
	domains, err := os.ReadDir(knowledgeDir)
	if err != nil {
		return nil
	}
	var clusters []string
	for _, d := range domains {
		if !d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			continue
		}
		subs, err := os.ReadDir(filepath.Join(knowledgeDir, d.Name()))
		if err != nil {
			continue
		}
		for _, s := range subs {
			if s.IsDir() && !strings.HasPrefix(s.Name(), ".") {
				clusters = append(clusters, fmt.Sprintf("  %s / %s", d.Name(), s.Name()))
			}
		}
	}
	return clusters
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
