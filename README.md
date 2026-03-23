# Ngram

A personal knowledge engine. Capture raw notes from any context. Get back searchable, interlinked atomic notes organized by domain. Quiz yourself through spaced repetition delivered to your phone. Ask questions against your entire knowledge base with sourced answers.

Single Go binary. Meilisearch for search. Claude Code CLI for AI structuring. Obsidian vault as the storage layer.

## Install

```bash
go install github.com/tylercooper/ngram/cmd/n@latest
```

## Setup

```bash
# Create config
cat > ~/.ngram.yml << 'EOF'
vault_path: ~/vault
model: cloud
EOF

# Initialize vault
n init

# Start services (Meilisearch + processor daemon)
n up

# Install as system service (survives reboot)
n up --install
```

## Capture

```bash
# Instant text note
n found sqli on login param id

# With explicit title
n -t "nmap results" open ports on 445, 139, 80

# Pipe command output
nmap -sV 10.10.10.8 | n -t "nmap-optimum"

# Run and capture (command string + output)
n run nmap -sV 10.10.10.8

# Engagement scaffolding
n box optimum 10.10.10.8 --os=windows
n phase exploit
```

Notes land in `_inbox/` and are automatically structured by the AI pipeline into the correct vault directory with tags, domain classification, and retention metadata.

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

## Commands

```
CAPTURE
  n <text>                    instant note
  n -t "title" <text>         note with title
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
n <text> → _inbox/ → processor → knowledge/{domain}/{cluster}/
                         ↓
                   Claude Code CLI
                   (structure, tag, classify)
                         ↓
                   Meilisearch index
                         ↓
                   git auto-commit
```

Two repos: `ngram` (tool source) and vault (private data, connected via `~/.ngram.yml`).

## Tech Stack

- **Go** single binary, all code
- **Cobra + Viper** CLI and config
- **Bubbletea + Lipgloss** terminal UI
- **Meilisearch** (Docker) hybrid search
- **Claude Code CLI** (`claude -p`) for all AI operations
- **fsnotify** file watching with 500ms debounce
- **go-sqlite3** iMessage chat.db polling
- **Git** auto-commit on every vault change

## License

Private.
