package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ty-cooper/ngram/internal/taxonomy"
)

// StructuringSystemPrompt guides Claude on how to structure notes.
const StructuringSystemPrompt = `You split raw input into atomic Zettelkasten notes. One idea per note. If the input covers one topic, return one note. If it covers five topics, return five notes.

ATOMIC NOTE PRINCIPLES:
- ONE idea per note — like fitting on a single index card
- Write clearly enough that a stranger could understand it
- Explain the concept as if teaching it to someone new (Feynman Technique)
- A single sentence is fine. Cap at ~500 words max.
- Imperfect notes are OK — capture the idea, it develops over time

RULES:
- Preserve the user's original meaning. Do not add opinions or filler.
- Commands MUST be in fenced code blocks with the correct language identifier (powershell, bash, cmd, sql, etc.)
- Each code block MUST start with a # [Context] comment identifying the TOOL or SHELL where the command runs. Examples:
  # [Mimikatz] — Mimikatz console
  # [PowerShell] — PowerShell prompt
  # [cmd] — Windows cmd.exe
  # [Bash] — Linux/macOS terminal
  # [msfconsole] — Metasploit console
  # [msfvenom] — msfvenom CLI
  # [sqlcmd] — SQL Server client
  # [BloodHound] — BloodHound UI
  # [CrackMapExec] — CrackMapExec CLI
  # [Kerbrute] — Kerbrute CLI
  NEVER use vague labels like "Attacker", "Target", "Victim", "Host". Always name the specific tool or shell.
  If the command runs in a standard terminal, use the shell name (Bash, PowerShell, cmd).
  This makes every note self-contained — a reader can understand the command without surrounding context.
- Replace specific IPs, hostnames, usernames, passwords in commands with {{PLACEHOLDERS}} like {{TARGET_IP}}, {{USERNAME}}, {{PORT}} so they are reusable
- Keep the original specific values in the explanation text, just genericize the command blocks
- Summary under 120 chars
- Google developer docs style: declarative, present tense, no filler

TAGS (STRICT):
- Maximum 5 tags per note. Most notes need 2-3. Each tag must earn its place.
- HEAVILY prefer existing tags from the ALLOWED TAGS list. Only create a new tag if NO existing tag fits.
- Tags should be specific and useful for retrieval, not generic filler like "security" or "tool".
- Do NOT create near-duplicates of existing tags (e.g. "nmap-scanning" when "nmap" exists).
- NEVER use the domain name as a tag (e.g. if domain is "biology", don't add "biology" as a tag).
- NEVER use system tags: "inbox", "test", "log", "capture-session". These are internal and not for knowledge notes.

DISCARD:
If the input is incoherent gibberish (e.g. "asdfghjk"), test/debug data (e.g. "testing 123"), or truly empty, return {"notes": [], "discard": true, "discard_reason": "..."}.
ANY coherent factual statement is valid knowledge — never discard based on topic or domain. "the plural of zebra is zebrae" is valid. "nmap -sV scans versions" is valid. ALL topics are welcome.

DOMAIN vs TOPIC CLUSTER vs TAGS:
- domain: broad knowledge area (e.g. "biology", "penetration-testing"). One per note.
- topic_cluster: subtopic within the domain (e.g. "neurology" under "biology"). Groups related notes.
- tags: cross-cutting retrieval labels (e.g. "nmap", "fish"). Specific, not generic. Never duplicate the domain.

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
		fmt.Fprintf(&b, "EXISTING DOMAINS: %s\n", strings.Join(domains, ", "))
		b.WriteString("Prefer these. Create a new domain if none fit — all knowledge is welcome.\n\n")
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

	// Remove system/blacklisted tags and domain-as-tag.
	blacklist := map[string]bool{
		"inbox": true, "test": true, "log": true,
		"capture-session": true, "ngramcapture": true,
	}
	domains := tax.CanonicalDomainList()
	for _, d := range domains {
		blacklist[d] = true
	}
	for t := range blacklist {
		delete(allTags, t)
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
