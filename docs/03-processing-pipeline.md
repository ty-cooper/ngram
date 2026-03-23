# Processing Pipeline

The processor goroutine is the core intelligence of the system. It watches `_inbox/` via fsnotify, structures raw notes through Claude, and writes organized output to the vault.

## Directory-Based State Machine

Processing state is determined by which directory a file lives in. No frontmatter flags. A directory listing shows the entire system state.

```
_inbox/        ← raw notes land here (CLI, overlay, pipe, iMessage)
_processing/   ← goroutine moves note here while working (filesystem lock)
knowledge/     ← structured knowledge notes (final destination)
boxes/         ← engagement findings (final destination)
tools/         ← tool references (final destination)
_archive/      ← permanent archive of every raw input
```

**Why directories instead of frontmatter flags:** Zero parsing overhead. `ls _inbox/` shows the backlog. Crash recovery is trivial (anything in `_processing/` on restart = interrupted, retry it). No double-processing because the file is physically moved, not flagged.

## Pipeline Steps

Each note goes through 15 steps:

```
 1. fsnotify detects new file in _inbox/ (debounced 500ms)
 2. Move to _processing/ (filesystem lock, prevents double-processing)
 3. Read raw content + any images (capture bundles have manifest.yml)
 4. Detect capture bundle → process screenshots via Claude vision
 5. Claude API: raw text + images → structured JSON
    - Structures per atomic note spec
    - Proposes: title, domain, topic_cluster, tags, content_type, body, links
    - Canonical tag list from taxonomy.yml injected into prompt
 6. Taxonomy resolution: resolve proposed tags/domain via _meta/taxonomy.yml aliases
 7. Deterministic linter: em dashes, filler, summary length, tags
    - Fail → retry structuring with violation feedback (max 2 retries)
 8. Claude quality gate (only if linter passes): atomicity, coherence, depth
    - Fail → retry (max 2 retries, then flag for manual review)
 9. Dedup check: Meilisearch hybrid search for similarity >= 0.85
    - Match found → route to dedup agent for merge/keep/link decision
10. Write structured note to correct directory
11. Initialize retention frontmatter (knowledge → state: new; others → null)
12. Update Meilisearch index (via indexer goroutine, not inline)
13. Git auto-commit: "ngram: structured {id} from {source}"
14. Archive raw input: move from _processing/ to _archive/
15. Log API usage to _meta/api-usage.jsonl
```

## Capture Bundle Processing

The overlay (Cmd+Shift+N) creates mixed-media sessions: screenshots + text interleaved. These land in `_inbox/` as a directory with a `manifest.yml`.

1. Read manifest.yml for item ordering
2. Send all screenshots to Claude as image inputs (vision extracts text)
3. Send text blocks alongside images with positional context
4. Claude decides segmentation: one note or multiple atomic notes (max 3)
5. Each output note follows the standard pipeline from step 6

No separate OCR library. Claude vision handles all screenshot text extraction.

## Dedup Agent

When Meilisearch returns similar existing notes (>= 0.85), the dedup agent receives the new note AND the top 5 matches. It decides:

**APPEND**: New content belongs in an existing note. Body extended, tags merged, modified timestamp updated, git commit. Retention state preserved (appending doesn't reset quiz schedule).

**NEW**: Genuinely new content. If the raw input contains multiple concepts, the agent identifies up to 3 atomic concepts and routes each to a subagent.

**DUPLICATE**: Already fully captured. Raw input moved to `_archive/` with a log entry in `_meta/dedup-log.jsonl`. No new note created.

Decision matrix:

```
Similarity >= 0.95 AND same domain/context → DUPLICATE
Similarity >= 0.85 AND related content     → APPEND to existing note
Similarity >= 0.85 AND different angle     → NEW (keep both, add links)
Similarity < 0.85                          → NEW (no dedup concern)
```

API cost per dedup decision: 0 embedding calls (Meilisearch handles this), 1 master agent call, 0-3 subagent calls if NEW with multiple concepts. Default to NEW on any error (never lose a note).

## Dual-Mode AI

```
MODEL=cloud  → claude CLI binary (default)
MODEL=local  → ollama binary (client engagements, air-gapped)
MODEL=off    → skip AI, write raw note directly, index in Meilisearch only
MODEL=mock   → test fixture scripts, no API calls
```

Read from `~/.ngram.yml` (global) or `.boxrc` (per-engagement override). When `MODEL=local`, no data leaves the machine.

## API Degradation

When the Claude binary is unreachable:

- Notes queue in `_inbox/` (not moved to `_processing/`)
- `n status` shows: "processor: degraded (API unreachable, 3 notes queued)"
- When API returns, queued notes process in order
- Quality gate falls back to deterministic linter only

## Auto-Linking

Claude identifies references to related concepts and generates `[[noteID]]` wiki-links. The auto-linker also searches Meilisearch for existing notes that should link to the new note and appends backlinks. Links use note IDs (not filenames) so they survive renames.

## Crash Recovery

On startup, the processor goroutine:

1. Checks `_processing/` for orphaned files (crashed mid-processing)
2. Moves them back to `_inbox/` for reprocessing
3. Cleans up orphaned `.tmp` files

## Reprocessing

When structuring prompts improve or taxonomy changes:

```bash
n reprocess optimum              # copies box notes back to _inbox/
n reprocess --all                # copies everything back to _inbox/
n reprocess --low-confidence     # only notes where AI confidence was low
```
