# CLI Reference

The `n` binary handles capture, search, quizzing, taxonomy, and service management. Built with Cobra + Viper.

## Configuration

```yaml
# ~/.ngram.yml
vault_path: ~/vault
model: cloud                       # cloud | local | off
imessage:
  phone: "+1XXXXXXXXXX"
  bridge: bluebubbles
  bluebubbles_url: "http://localhost:1234"
schedule:
  wake_hour: 8
  sleep_hour: 22
api:
  daily_budget_usd: 5.00
embeddings:
  openai_api_key: "sk-..."        # or set OPENAI_API_KEY env var
```

API keys: `ANTHROPIC_API_KEY` for LLM calls, `OPENAI_API_KEY` for embeddings. Env vars or config file.

## Context Engine: `.boxrc`

The CLI reads `.boxrc` from cwd, then walks parent dirs. All notes auto-tagged with box, phase, IP.

```
BOX=optimum
IP=10.10.10.8
PHASE=exploit
ENGAGEMENT=htb-2026
MODEL=cloud
```

## Commands

### Capture

```bash
n <text>                          # instant note to _inbox/
n -t "title" <text>               # note with explicit title
n run <command>                   # execute command, capture command + output
command | n -t "title"            # pipe output to note
```

`n <text>` writes markdown to `_inbox/{timestamp}-{slug}.md` with frontmatter from `.boxrc` context.

`n run` uses `os/exec`, captures stdout+stderr, writes the command string in frontmatter.

### Engagement

```bash
n box <name> <ip> [--os=]         # scaffold target (full folder structure + .boxrc)
n phase <phase>                   # switch active phase in .boxrc
n engage <name>                   # pause quizzes, engagement mode
n disengage                       # resume quizzes
```

### Search

```bash
nq <query>                        # search vault (plain text + field:value filters)
n ask "question"                  # RAG query with sourced answer
n ask "q" --domain X              # domain-restricted RAG
n ask "q" --sources-only          # show matching notes only
```

### Retention

```bash
n quiz                            # manual terminal quiz session
n quiz --domain X                 # domain-filtered quiz
n quiz --weak                     # notes with score < 60
n stats                           # retention dashboard
n due                             # today's quiz schedule
```

### Taxonomy

```bash
n tags                            # list canonical tags
n tags --proposed                 # unreviewed new tags
n tags audit                      # spot drift
n tags merge old new              # merge tag variant
n tags alias "x" "y"              # add alias
n tags rename old new             # rename across vault
```

### System

```bash
n up                              # start all services
n up --foreground                 # run in foreground (for launchd/systemd)
n up --install                    # install as system service + start
n up --uninstall                  # remove system service
n down                            # stop all services
n status                          # service health + goroutine status
n restart                         # restart all
n logs [service]                  # tail logs
n reindex                         # rebuild Meilisearch index
n gc                              # clean _trash/ (manual, requires confirm)
n report <box>                    # generate engagement report
```

## Vault Structure

```
vault/
├── _inbox/                       raw captures land here
├── _archive/                     permanent raw input archive
├── _processing/                  in-flight processing
├── _trash/                       soft-deleted notes
├── _meta/                        system metadata (git-tracked)
│   ├── taxonomy.yml              canonical tags + aliases
│   ├── topic-clusters.yml        domain/cluster definitions
│   ├── quiz-history.jsonl        append-only quiz log
│   ├── ask-history.jsonl         append-only RAG query log
│   ├── api-usage.jsonl           API call tracking
│   ├── knowledge-gaps.jsonl      unanswered topics
│   ├── quiz-delivery.yml         schedule config
│   ├── quiz-delivery-state.json  daily schedule state
│   ├── heartbeat.json            process health
│   ├── retention-snapshot.json   daily retention stats
│   ├── stats-cache.json          precomputed stats for n stats
│   ├── dedup-log.jsonl           dedup decisions
│   └── dream-state.json         per-note dream review timestamps
├── knowledge/                    domain-agnostic knowledge base
│   └── {domain}/{cluster}/
├── boxes/                        engagement targets
│   └── {box-name}/
│       ├── _recon/
│       ├── _enum/
│       ├── _exploit/
│       ├── _post/
│       └── _loot/
├── tools/
├── creds/                        encrypted
├── reports/
└── _templates/
```
