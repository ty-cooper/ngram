# Architecture

## What Ngram Is

A personal knowledge engine built on Obsidian markdown files, powered by a single Go binary (`n`) and Meilisearch. Raw thoughts go in from any context. Claude structures them into atomic notes. A spaced repetition engine quizzes you via iMessage throughout the day. A RAG layer answers natural language questions against everything you've ever learned. All data stays as markdown on disk, version-controlled by git.

## Tech Stack

| Component | Technology | Why |
|-----------|-----------|-----|
| Language | Go | Single static binary. No runtime deps. Instant startup. CLI + daemon in one process. |
| CLI framework | Cobra + Viper | Cobra for subcommands and shell completions. Viper merges config from `~/.ngram.yml`, env vars, and flags. |
| File watching | fsnotify | OS-native filesystem events (inotify/kqueue/FSEvents). No polling. |
| Terminal UI | Bubbletea + Lipgloss | Charm stack for interactive quiz sessions and dashboards. |
| Search | Meilisearch (Docker) | Hybrid keyword + semantic search. Native embeddings. Fuzzy, typo-tolerant. Sub-50ms. |
| Embeddings | Ollama nomic-embed-text (local) or OpenAI (cloud) | Configured inside Meilisearch. No custom embedding store. |
| LLM | Claude Code CLI (`claude -p`) | All AI calls shell out to the `claude` binary via `os/exec`. No direct Anthropic API HTTP calls. Claude Code handles auth, retries, model selection. |
| Screenshot text | Claude vision | Screenshots sent as images to Claude. No OCR library. |
| iMessage | BlueBubbles REST API (production) / AppleScript (MVP) | Quiz delivery to your phone. Swappable via interface. |
| Capture overlay | Hammerspoon (MVP) / Swift NSPanel (production) | Global hotkeys from any window. |
| Notes viewer | Obsidian | Markdown rendering, graph view, plugins. |
| Version control | Git | Auto-commit on every vault change. Hourly push to private GitHub repo. |
| Data format | Markdown + YAML frontmatter | Human-readable, portable, grep-friendly. |

## Two-Repo Architecture

**`ngram` repo (this repo):** Go source code for the `n` binary, Docker Compose config for Meilisearch, Hammerspoon scripts, structuring prompts, quality gate prompts, seed taxonomy, note templates, build scripts.

**Vault repo (always private):** User data. `_inbox/`, `_archive/`, `_meta/`, `knowledge/`, `boxes/`, `tools/`. This is what Obsidian opens. This is what gets hourly git pushes.

The `n` binary reads the vault path from config:

```yaml
# ~/.ngram.yml
vault_path: ~/vault
model: cloud        # cloud | local | off
```

Or via env var: `NGRAM_VAULT_PATH`.

This separation means the tool is open-sourceable while vault data stays private. A new machine setup is:

```bash
go install github.com/yourusername/ngram/cmd/n@latest
git clone git@github.com:yourusername/my-vault.git ~/vault
echo "vault_path: ~/vault" > ~/.ngram.yml
n up
```

## How Components Connect

```
┌──────────────────────────────────────────────────────────────┐
│                         YOUR BRAIN                            │
└──────────────────────────┬───────────────────────────────────┘
                           │
          ┌────────────────▼─────────────────────────┐
          │           CAPTURE LAYER                    │
          │  Cmd+Shift+N overlay (mixed media)         │
          │  n <text> (terminal)                       │
          │  n run <command> (execute + capture)       │
          │  command | n -t "title" (pipe)              │
          │  note: via iMessage (phone)                │
          └────────────────┬─────────────────────────┘
                           │ writes to _inbox/
          ┌────────────────▼─────────────────────────┐
          │      AI PROCESSING (Go goroutine + Claude) │
          │  Structure → Lint → Quality gate → Dedup   │
          │  Taxonomy resolution from _meta/taxonomy.yml│
          │  Screenshot text via Claude vision          │
          └────────────────┬─────────────────────────┘
                           │ atomic notes
          ┌────────────────┴─────────────────────────┐
          │                    │                       │
  ┌───────▼────────┐  ┌──────▼──────────┐            │
  │ OBSIDIAN VAULT  │  │  MEILISEARCH    │            │
  │ knowledge/      │  │  hybrid search  │            │
  │ boxes/          │  │  keyword+semantic│            │
  │ git auto-commit │  │  faceted filters│            │
  └───────┬────────┘  └──────┬──────────┘            │
          │                   │                       │
          └─────────┬─────────┘                       │
                    │                                  │
  ┌─────────────────▼──────────────────────────────┐  │
  │         RETRIEVAL + RETENTION                   │  │
  │  nq (structured search)                         │  │
  │  n ask (RAG with citations)                     │  │
  │  SM-2 spaced repetition → iMessage delivery     │  │
  └─────────────────────────────────────────────────┘
```

**Running processes: 3.** The `n` binary (CLI + goroutines), the Meilisearch Docker container, and the hotkey overlay.

## Key Design Decisions

**Go for everything.** One binary. CLI commands, daemon goroutines, file watcher, service orchestrator all live in the same process. No Python. No microservices.

**Claude Code CLI, not direct API.** The `n` binary shells out to `claude -p` for every AI operation. Claude Code handles auth, retries, rate limits, and model selection. This eliminates HTTP client code, API key management, token counting, and retry logic from the codebase. Testing mocks at the `exec.Command` level with fixture scripts.

**No custom embedding store.** Meilisearch generates, stores, and indexes vectors natively via its hybrid search feature. No separate vector DB. No custom cosine similarity. Dedup detection is a Meilisearch search call.

**No OCR library.** Claude vision handles screenshot text extraction. Screenshots are sent as images in the structuring API call.

**Goroutines, not processes.** `n up` spawns goroutines with a shared `context.Context`. `n down` cancels the context. Clean shutdown propagates to all goroutines. No PID files. No custom supervisor.

**Directory-based pipeline, not frontmatter flags.** Processing state is determined by which directory a file is in (`_inbox/` → `_processing/` → `knowledge/`), not by parsing YAML fields. A directory listing tells you the entire system state.

**Controlled taxonomy, not freeform tags.** Tags are resolved against `_meta/taxonomy.yml` with aliases. Prevents drift where "pentesting", "penetration-testing", "ptest", and "red-team" become separate tags.

**Content types gate quiz eligibility.** Only `content_type: knowledge` notes enter the spaced repetition queue. Recipes (`reference`), command captures (`log`), bookmarks (`link`), and screenshots (`media`) are searchable but never quizzed.

**Nothing is ever hard-deleted.** Raw inputs archived in `_archive/`. Deleted notes go to `_trash/`. `n gc` is manual and requires confirmation. Git auto-commits after every operation.

**Meilisearch is a rebuildable cache.** `n reindex` reconstructs the entire search index from vault markdown files. The vault is the source of truth.

## Data Protection

- Git auto-commit after every vault modification
- Raw inputs archived permanently in `_archive/`
- Atomic file writes (temp file + `os.Rename`)
- Soft deletes only (`_trash/` with timestamps)
- Append-only logs for quiz history, ask history, API usage
- Hourly git push to private GitHub repo
- Pre-flight checks before bulk operations
- API cost monitoring with daily budget cap
