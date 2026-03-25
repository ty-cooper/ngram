package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ty-cooper/ngram/internal/taxonomy"
)

// StructuringSystemPrompt guides Claude on how to structure notes.
const StructuringSystemPrompt = `You split raw input into atomic notes. One concept per note. If the input covers one topic, return one note. If it covers five topics, return five notes.

RULES:
- Each note covers ONE concept, technique, or finding
- Preserve the user's original meaning. Do not add opinions or filler.
- Commands MUST be in fenced code blocks with language identifiers
- Replace specific IPs, hostnames, usernames, passwords in commands with {{PLACEHOLDERS}} like {{TARGET_IP}}, {{USERNAME}}, {{PORT}} so they are reusable
- Keep the original specific values in the explanation text, just genericize the command blocks
- Summary under 120 chars
- Google developer docs style: declarative, present tense, no filler

TAGS (STRICT):
- Maximum 5 tags per note. Most notes need 2-3. Each tag must earn its place.
- HEAVILY prefer existing tags from the ALLOWED TAGS list. Only create a new tag if NO existing tag fits.
- Tags should be specific and useful for retrieval, not generic filler like "security" or "tool".
- Do NOT create near-duplicates of existing tags (e.g. "nmap-scanning" when "nmap" exists).

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
// Merges canonical taxonomy tags with discovered vault tags into one authoritative list.
func BuildStructuringPrompt(tax *taxonomy.Taxonomy, rawContent string, vaultPath ...string) string {
	var b strings.Builder

	b.WriteString("Split this into atomic notes. One concept per note. Genericize commands with {{PLACEHOLDER}} syntax.\n\n")

	if domains := tax.CanonicalDomainList(); len(domains) > 0 {
		fmt.Fprintf(&b, "ALLOWED DOMAINS: %s\n", strings.Join(domains, ", "))
		b.WriteString("Use one of these. Only propose a new domain if none fit.\n\n")
	}

	// Build unified tag list: taxonomy canonical + discovered from vault.
	allTags := make(map[string]bool)
	for _, t := range tax.CanonicalTagList() {
		allTags[t] = true
	}
	if len(vaultPath) > 0 && vaultPath[0] != "" {
		for _, t := range discoverTags(vaultPath[0]) {
			allTags[t] = true
		}
	}

	if len(allTags) > 0 {
		tagList := make([]string, 0, len(allTags))
		for t := range allTags {
			tagList = append(tagList, t)
		}
		fmt.Fprintf(&b, "ALLOWED TAGS: %s\n", strings.Join(tagList, ", "))
		b.WriteString("Use tags from this list. Only create a new tag if NONE of these fit. Max 5 tags per note.\n\n")
	} else {
		b.WriteString("No existing tags yet. Create concise, specific tags. Max 5 per note.\n\n")
	}

	b.WriteString("RAW NOTE:\n")
	b.WriteString(rawContent)

	return b.String()
}

// discoverTags scans all knowledge notes (recursively) for existing tags in frontmatter.
func discoverTags(vaultPath string) []string {
	tagSet := make(map[string]bool)
	knowledgeDir := filepath.Join(vaultPath, "knowledge")

	filepath.Walk(knowledgeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
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
		return nil
	})

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
