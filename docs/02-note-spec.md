# Atomic Note Specification

Every note the AI pipeline produces conforms to this spec. Search, quizzing, dedup, and retention all depend on this format.

## Principles

- **Atomic.** One concept per note. Two ideas become two notes.
- **Searchable.** Every sentence surfaces when queried for its concept.
- **Deduplicable.** Same concept in different words still matches via Meilisearch hybrid search at >= 0.85 similarity.
- **Self-contained.** Makes sense without reading other notes. Wiki-links add depth, not context.
- **Google developer docs style.** No filler. No LLM mannerisms. No em dashes. Direct, factual, terse.

## File Structure

Every note is a markdown file with four sections. No exceptions.

```markdown
---
[frontmatter]
---

[one-line summary]

[body]

## Links

- [[noteID]] Related note title
```

No "Overview" or "Conclusion" headers. No preamble.

## Frontmatter Schema

```yaml
---
id: "a1b2c3d4"                    # 8-char hex, unique across vault
title: "Raft leader election"      # noun phrase, no verbs
content_type: "knowledge"          # knowledge | reference | log | link | media
created: "2026-03-22T10:42:00Z"
modified: "2026-03-22T10:42:00Z"
source: "DDIA Ch. 9"              # exact origin
source_type: "book"               # book | course | engagement | study | reference
domain: "distributed-systems"     # canonical, from taxonomy
topic_cluster: "consensus"        # sub-topic, from taxonomy
tags: [raft, leader-election, consensus]  # canonical only
url: null                         # populated for content_type: link
related: []                       # populated by auto-linker
retention:                        # null for non-knowledge types
  state: "new"
  ease_factor: 2.5
  interval_days: 0
  repetition_count: 0
  last_reviewed: null
  next_review: null
  total_reviews: 0
  total_correct: 0
  retention_score: 0
  difficulty_rating: null
  streak: 0
  lapse_count: 0
---
```

**`id`**: 8 hex chars generated at creation. Used for cross-references and link stability. File renames don't break links.

**`title`**: Noun phrase describing the concept. "Raft leader election" not "How Raft elects leaders."

**`domain` and `topic_cluster`**: Top-level fields, not nested inside `retention:`. Must resolve against `_meta/taxonomy.yml`.

**`retention`**: Full SM-2 block for `knowledge` notes. `null` for everything else.

## Content Types

| Type | Quizzed? | Examples |
|------|----------|----------|
| `knowledge` | Yes | Nmap scan types, Raft consensus, DDIA chapters |
| `reference` | No | Recipes, config templates, checklists, code snippets |
| `log` | No | Command output captures, engagement timelines |
| `link` | No | Saved articles, videos, bookmarks |
| `media` | No | Screenshots, diagrams |

Auto-detected by the processor. A muffin recipe becomes `reference`. A YouTube URL becomes `link`. Study notes become `knowledge`. The user never specifies this.

## File Naming and Placement

File name: `{id}-{slug}.md` where slug is title lowercased, hyphenated, max 50 chars.

```
knowledge/{domain}/{topic_cluster}/{id}-{slug}.md
boxes/{box}/{phase}/{id}-{slug}.md
tools/{tool-name}/{id}-{slug}.md
```

## Writing Rules

The body follows Google developer documentation style:

- Present tense, active voice
- No em dashes (use commas, periods, or parentheses)
- No filler: "important," "key," "crucial," "essentially," "basically," "simply," "just," "actually"
- No hedging: "might," "could potentially," "it seems like"
- No weak starters: "It is," "There are," "This is," "Note that"
- State facts directly: "The ACK flag bypasses stateless firewalls" not "The ACK flag can potentially be used to bypass stateless firewalls"
- One paragraph per idea, 2-4 sentences each
- Code blocks with language identifiers
- Don't restate the summary in the body

## Summary Line

First line after frontmatter. One sentence, under 120 characters. States the core concept. No formatting. Independently searchable.

Good: `A Raft follower that times out starts an election by incrementing its term, voting for itself, and requesting votes from peers.`

Bad: `This note covers Raft leader election.`

## Tag Taxonomy

Tags are controlled via `_meta/taxonomy.yml`. No freeform tagging.

```yaml
domains:
  pentest:
    aliases: [pentesting, penetration-testing, ptest, red-team, redteam, offsec]
  distributed-systems:
    aliases: [dist-sys, distributed, distributed-computing]
  cooking:
    aliases: [recipes, food, baking]

tags:
  nmap:
    aliases: [nmap-scan, nmap-scanning]
    domain_hint: pentest
  privilege-escalation:
    aliases: [privesc, priv-esc, escalation, pe]
    domain_hint: pentest
  consensus:
    aliases: [consensus-protocol, consensus-algo]
    domain_hint: distributed-systems
```

### Three-Layer Enforcement

**Layer 1: Prompt injection.** The canonical tag list is injected into Claude's structuring prompt. Claude sees the allowed vocabulary and uses it. Catches ~90% of cases.

**Layer 2: Alias resolution.** After Claude returns proposed tags, the processor resolves each through the taxonomy. "pentesting" becomes `pentest`. "fw-bypass" becomes `firewall-bypass`. A Go function, ~40 lines, runs in microseconds.

```go
func (t *Taxonomy) Resolve(raw string) string {
    slug := slugify(raw)
    if _, ok := t.Tags[slug]; ok { return slug }
    for canonical, entry := range t.Tags {
        for _, alias := range entry.Aliases {
            if alias == slug { return canonical }
        }
    }
    return t.ProposeNew(slug)
}
```

**Layer 3: New tag proposals.** Unknown tags are auto-added to taxonomy.yml with `proposed: true`. Usable immediately, flagged for review via `n tags --proposed`.

## Quality Gate

Two parts, run in sequence:

**Part 1: Deterministic linter (Go function, no API call).** Checks em dashes, banned filler words, weak starters, summary length, title format, tag validity, required frontmatter fields. Catches ~40% of issues at zero cost.

**Part 2: Claude quality check (only if Part 1 passes).** Checks atomicity (one concept per note), coherence, depth, link suggestions. Saves API calls by filtering formatting failures before they reach Claude.

Max 2 retries before flagging for manual review.

## Dedup Contract

Meilisearch hybrid search at similarity >= 0.85 flags potential duplicates. The processor routes flagged notes to the dedup agent (see [03-processing-pipeline.md](03-processing-pipeline.md)) for a merge/keep/link decision. No custom embedding store. No cosine similarity code.
