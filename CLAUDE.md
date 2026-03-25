# CLAUDE.md

## Project

Ngram — a personal second brain built on Obsidian markdown files, powered by a single Go binary (`n`) and Meilisearch. All specs and tickets live in Linear under the "Zero Friction Pentest Knowledge System" project. The canonical bootstrap doc is v3 ("Ngram Implementation Bootstrap — Read This First (v3, Current)").

Note: Some Linear tickets still reference "ZFKS" or "Engram". The canonical name is **Ngram**. Replace all instances during implementation.

## Framework and Dependency Rules

Before using or recommending any framework, library, or dependency:

1. **Always web search** for the latest stable version and current best practices before writing code that depends on it.
2. **Use the latest stable release** of every dependency. Do not rely on memorized version numbers — verify via web search.
3. **Check for breaking changes** between major versions. If migrating or starting fresh, use the current API surface, not deprecated patterns.
4. This applies to all dependencies: Go modules, Docker images, Meilisearch versions, CLI frameworks (Cobra, Viper), TUI libraries (Bubbletea, Lipgloss), and any third-party APIs.

## Tech Stack (from audit v2)

- **Go** for everything. No Python. Single binary.
- **Cobra + Viper** for CLI and config.
- **Bubbletea + Lipgloss** (Charm) for terminal UI.
- **Meilisearch** (Docker) for search. Native hybrid search with embeddings.
- **fsnotify** for file watching.
- **Anthropic Go SDK** (`github.com/anthropics/anthropic-sdk-go`) for all LLM calls. Uses `ANTHROPIC_API_KEY`.
- **Git** auto-commit on every vault change.

## LLM Integration

All AI operations use the Anthropic Go SDK directly. Runner modes:
- cloud (production, requires ANTHROPIC_API_KEY)
- off (skip AI, write raw)
- mock (fixture responses for testing)

Vision support: JPEG, PNG, GIF, WebP natively. HEIC/HEIF converted to JPEG via macOS `sips`.

## Two-Repo Architecture

- `ngram` repo: tool source code (this repo)
- Vault repo: user data, always private, connected via `~/.ngram.yml`

## Code Style

- No overengineering. Minimal abstractions.
- Google developer docs style for any generated notes or documentation.
- Terse, direct code. No unnecessary comments or docstrings on obvious code.
