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
- **Claude Code CLI** (`claude -p`) for all LLM calls via os/exec. No direct Anthropic API HTTP calls.
- **Git** auto-commit on every vault change.

## LLM Integration

All AI operations shell out to the `claude` CLI binary. LLMClient interface with implementations:
- ClaudeCodeClient (production, MODEL=cloud)
- OllamaClient (MODEL=local, HTTP to localhost:11434)
- MockClient (MODEL=mock, fixture scripts)
- ResilientClient (circuit breaker wrapper)

No ANTHROPIC_API_KEY management. Claude Code handles its own auth.

## Two-Repo Architecture

- `ngram` repo: tool source code (this repo)
- Vault repo: user data, always private, connected via `~/.ngram.yml`

## Code Style

- No overengineering. Minimal abstractions.
- Google developer docs style for any generated notes or documentation.
- Terse, direct code. No unnecessary comments or docstrings on obvious code.
