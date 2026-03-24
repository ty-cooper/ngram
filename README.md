# Ngram

A personal knowledge engine. Capture raw notes from any context. Get back searchable, interlinked atomic notes organized by domain. Quiz yourself through spaced repetition delivered to your phone. Ask questions against your entire knowledge base with sourced answers.

Single Go binary. Meilisearch for search. Claude Code CLI for AI structuring. Obsidian vault as the storage layer.

## Prerequisites

- **Go** 1.22+
- **Docker** (for Meilisearch)
- **Anthropic API key** (`ANTHROPIC_API_KEY` env var)
- **Obsidian** (vault viewer)
- **macOS** (for iMessage quizzes and capture overlay)

## Install

```bash
# CLI
go install github.com/ty-cooper/ngram/cmd/n@latest

# Or build from source
git clone https://github.com/ty-cooper/ngram.git
cd ngram
make build          # builds bin/n
make install        # installs to $GOPATH/bin/n
```

## Setup

```bash
# 1. Set your Anthropic API key
export ANTHROPIC_API_KEY="sk-ant-..."  # add to your shell profile

# 2. Create config
cat > ~/.ngram.yml << 'EOF'
vault_path: ~/path/to/your/vault
model: cloud
imessage:
  phone: "+1XXXXXXXXXX"
schedule:
  wake_hour: 8
  sleep_hour: 22
EOF

# 3. Initialize vault structure
n init

# 4. Start services (Meilisearch + processor daemon)
n up

# 5. Verify everything is running
n status

# 6. Install as system service (survives reboot)
n up --install
```

Open the vault path in Obsidian as a vault.

## Capture Overlay (macOS)

NgramCapture is a SwiftUI menu bar app for global capture via **Cmd+Option+N**.

```bash
make overlay                    # builds bin/NgramCapture.app
open bin/NgramCapture.app       # launch — appears in menu bar
```

Grant these macOS permissions (System Settings → Privacy & Security):
- **Accessibility** → add NgramCapture.app
- **Screen Recording** → add NgramCapture.app

Add to Login Items to start on boot.

After a rebuild (`make overlay`), you need to re-grant permissions since the code signature changes.

## Obsidian Plugin

The ngram-search plugin adds Meilisearch-powered search to Obsidian.

```bash
cd obsidian-plugin && npm install    # first time only
make obsidian                        # builds and installs to vault
```

In Obsidian: Settings → Community Plugins → turn off Restricted Mode → enable **Ngram Search**.

Open via **Cmd+Option+1** or Command Palette (Cmd+P) → "Ngram Search: Search vault". Results render as a single assembled document with all matching note bodies, code blocks, and clickable source links to jump to the original note.

## iMessage Quizzes

Requires macOS with Full Disk Access for the terminal or `n` binary (reads `~/Library/Messages/chat.db`).

System Settings → Privacy & Security → Full Disk Access → add your terminal app (Terminal.app, iTerm2, Alacritty, etc.).

## Capture

```bash
# Instant text note
n found sqli on login param id

# With explicit title
n -t "nmap results" open ports on 445, 139, 80

# Open $EDITOR (vim) to write a full note
n new

# Pipe command output
nmap -sV 10.10.10.8 | n -t "nmap-optimum"

# Run and capture (command string + output)
n run nmap -sV 10.10.10.8

# Engagement scaffolding
n box optimum 10.10.10.8 --os=windows
n phase exploit
```

Notes land in `_inbox/` and are automatically structured by the AI pipeline into `knowledge/` with tags, domain classification, and retention metadata. Screenshots go to `_assets/`.

## Search

```bash
# Plain text (fuzzy, typo-tolerant)
n search proxychains socks setup

# Structured filters
n search domain:pentest tag:nmap
n search state:learning score:<60
n search due:overdue
n search box:optimum phase:exploit

# RAG — synthesized answer with citations
n ask "how do I bypass a stateless firewall?"
n ask "what are the tradeoffs between B-trees and LSM trees?" --domain data-engineering
n ask "what creds do I have for optimum?" --sources-only
```

## Quiz

```bash
# Interactive terminal quiz (system-selected, cross-domain)
n quiz

# Filtered
n quiz --domain pentest
n quiz --weak          # score < 60
n quiz --new           # unreviewed notes
```

Quizzes also arrive via iMessage at random intervals throughout the day. Reply with your answer. Graded in 30 seconds.

## Dream Cycle

Nightly knowledge consolidation. Scans the vault for duplicates, junk notes, and near-synonym clusters. Creates a PR against the vault repo with proposed changes — each change is a separate commit so you can cherry-pick.

```bash
# Preview what would change
n dream --dry-run

# Run and create PR
n dream
```

Passes:
- **Dedup** — finds similar note pairs via Meilisearch, LLM decides merge or keep
- **Quality** — archives notes under 20 chars (junk captures)
- **Clusters** — detects near-synonyms (e.g. "Network Reconnaissance" vs "Network Scanning"), proposes taxonomy merges
- **Nothing** — if the vault is clean, no PR is created

Schedule nightly:

```bash
# crontab (run at 3am daily)
crontab -e
# add: 0 3 * * * /path/to/n dream >> /tmp/ngram-dream.log 2>&1

# or via n up --install (launchd on macOS)
# the daemon scheduler runs n dream automatically at 3am if configured:
# add to ~/.ngram.yml:
#   dream:
#     enabled: true
#     hour: 3
```

## Commands

```
CAPTURE
  n <text>                    instant note
  n -t "title" <text>         note with title
  n new                       open $EDITOR to write a note
  n run <command>             execute + capture (via /bin/sh -c)
  command | n -t "title"      pipe to note
  n amend <text>              append to last captured note
  n edit                      open last note in $EDITOR
  n capture-on                start auto-capture mode
  n capture-off               stop auto-capture mode

ENGAGEMENT
  n box <name> <ip> [--os=]   scaffold target
  n phase <phase>              switch phase
  n engage <name>              pause quizzes, set NGRAM_ENGAGEMENT
  n disengage                  resume quizzes

SEARCH
  n search <query>             vault search (field:value filters)
  n ask "question"             RAG synthesis with citations
  n domains                    list domains with note counts
  n domains <name>             list clusters under domain
  n reindex                    rebuild search index

RETENTION
  n quiz                       interactive quiz session
  n quiz --domain X            domain-filtered quiz
  n quiz --weak                notes with score < 60
  n verify                     check hash chain integrity

SYSTEM
  n init                       initialize vault structure
  n up                         start services (Meilisearch + daemon)
  n down                       stop all services
  n status                     health check + processing backlog
  n up --install               install as system service
  n up --uninstall             remove system service
  n dream                      nightly knowledge consolidation (creates PR)
  n dream --dry-run            preview changes without modifying
  n migrate --source <dir>     batch import existing vault
  n report <box>               generate engagement report
```

## iMessage Commands

When quizzes arrive on your phone, reply with:

| Reply | Action |
|-------|--------|
| *(answer text)* | Grade against source note |
| `Q2 answer` | Target a specific question by ID |
| `skip` | Skip, grade 0 |
| `idk` | Grade 0, system sends the answer |
| `defer` | Remove from today, no grade (comes back tomorrow) |
| `pause` | Pause delivery for today |
| `resume` | Resume delivery |
| `stats` | Today's score summary |
| `missed` | Get answers for today's incorrect questions |
| `review` | Key points from today's lapsed notes |

## Architecture

```
n <text> / NgramCapture overlay / iMessage
                    ↓
               _inbox/*.md
                    ↓
         processor goroutine (fsnotify)
                    ↓
          Anthropic API (structure, tag, classify)
                    ↓
           notes/{id}-{slug}.md (Zettelkasten flat)
                    ↓
         Meilisearch index + git auto-commit
```

Two repos: `ngram` (tool source) and vault (private data, connected via `~/.ngram.yml`).

## Tech Stack

- **Go** single binary, all backend code
- **Cobra + Viper** CLI and config
- **Bubbletea + Lipgloss** terminal UI (quiz TUI)
- **Meilisearch** (Docker) hybrid search with embeddings
- **Anthropic API** (`claude-sonnet-4-20250514`) for AI structuring with JSON prefill
- **SwiftUI** capture overlay (macOS menu bar app)
- **TypeScript** Obsidian search plugin
- **fsnotify** file watching with 500ms debounce
- **go-sqlite3** iMessage chat.db polling
- **Git** auto-commit on every vault change

## License

Private.
