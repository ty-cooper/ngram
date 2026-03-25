package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ty-cooper/ngram/internal/taxonomy"
)

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

DISCARD:
If the input is empty, junk, test data, or contains no extractable knowledge, return {"notes": [], "discard": true, "discard_reason": "..."}. Do not force-create notes from garbage.

CONTENT TYPES:
- knowledge: concepts, explanations (quizzed)
- reference: checklists, configs, commands (not quizzed)
- log: engagement output, findings (not quizzed)
- link: URLs (not quizzed)
- media: screenshots (not quizzed)

TOOL-PARSED OUTPUT:
If the input contains a "## Parsed Output" section, it was pre-structured by a tool parser. Preserve its tables and structure in the body. Use the parsed data to inform the title, summary, and tags. The "## Raw Output" section contains the original tool output for reference — include key details but don't reproduce the entire raw dump.`

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

	// Inject existing tags from vault notes so Claude reuses them.
	if len(vaultPath) > 0 && vaultPath[0] != "" {
		if existing := discoverTags(vaultPath[0]); len(existing) > 0 {
			fmt.Fprintf(&b, "EXISTING TAGS (reuse these, do not create near-duplicates):\n%s\n\n", strings.Join(existing, ", "))
		}
	}

	b.WriteString("RAW NOTE:\n")
	b.WriteString(rawContent)

	return b.String()
}

// discoverTags scans knowledge/ for existing tags in frontmatter.
func discoverTags(vaultPath string) []string {
	notesDir := filepath.Join(vaultPath, "knowledge")
	entries, err := os.ReadDir(notesDir)
	if err != nil {
		return nil
	}
	tagSet := make(map[string]bool)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(notesDir, e.Name()))
		if err != nil {
			continue
		}
		inFM := false
		inTags := false
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "---" {
				if inFM {
					break
				}
				inFM = true
				continue
			}
			if !inFM {
				continue
			}
			if trimmed == "tags:" {
				inTags = true
				continue
			}
			if inTags && strings.HasPrefix(trimmed, "- ") {
				tag := strings.TrimPrefix(trimmed, "- ")
				tagSet[tag] = true
			} else if inTags {
				inTags = false
			}
		}
	}
	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}
	return tags
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
