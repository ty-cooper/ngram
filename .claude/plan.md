# COO-64: Meilisearch — Plain Language Vault Search

## Versions
- Meilisearch Docker: `v1.40.0` (latest stable, March 2026)
- meilisearch-go SDK: `v0.36.1` (latest, Feb 2026)

## Scope

Wire up Meilisearch as the search backend. Index structured vault notes. Build `nq` search command.

## Changes

### 1. Update docker-compose.yml
- Bump image to `getmeili/meilisearch:v1.40.0`
- Add `MEILI_MASTER_KEY` env var (optional, for local dev security)

### 2. New package: `internal/search/`

**`client.go`** — Meilisearch client wrapper
- `Client` struct wrapping `meilisearch.ServiceManager`
- `New(host, apiKey string)` constructor
- `EnsureIndex()` — creates/updates the `notes` index with correct settings:
  - Searchable attributes: `title`, `tags`, `body`, `summary`
  - Filterable attributes: `domain`, `content_type`, `box`, `phase`, `tags`, `engagement`
  - Sortable attributes: `captured`
  - Ranking rules: default Meilisearch order (words, typo, proximity, attribute, sort, exactness)
- `IndexNote(doc NoteDocument)` — upsert a single document
- `DeleteNote(id string)` — remove by ID
- `Search(query string, opts SearchOptions) ([]SearchResult, error)` — search with optional filters

**`document.go`** — Note document struct matching Meilisearch schema
```go
type NoteDocument struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Body        string   `json:"body"`
    Summary     string   `json:"summary"`
    Tags        []string `json:"tags"`
    Domain      string   `json:"domain"`
    ContentType string   `json:"content_type"`
    Box         string   `json:"box,omitempty"`
    Phase       string   `json:"phase,omitempty"`
    Engagement  string   `json:"engagement,omitempty"`
    FilePath    string   `json:"file_path"`
    Captured    int64    `json:"captured"` // unix timestamp for sorting
}
```

**`parser.go`** — Parse a structured vault markdown file into a NoteDocument
- Read file, split frontmatter from body
- Parse YAML frontmatter into struct
- Return NoteDocument ready for indexing

**`search.go`** — Search types and result formatting
```go
type SearchOptions struct {
    Filter string   // Meilisearch filter expression
    Limit  int
    Offset int
}

type SearchResult struct {
    Title       string
    FilePath    string
    ContentType string
    Snippet     string // highlighted match
    Domain      string
    Score       float64
}
```

### 3. New package: `internal/search/query/`

**`translate.go`** — Thin query translator (per COO-84 direction)
- Parse `field:value` tokens out of the query string
- Known fields: `domain`, `box`, `phase`, `tag`, `type`, `engagement`
- Everything else is the free-text search query
- Convert field:value pairs to Meilisearch filter syntax
- Example: `nq proxy setup domain:pentest` → query="proxy setup", filter=`domain = "pentest"`

### 4. CLI command: `nq` / `n search`

**`internal/cli/search.go`**
- `searchCmd` added to root as `search` subcommand, aliased as `nq` via symlink or arg detection
- Loads config, creates Meilisearch client
- Parses query through translator
- Formats results to terminal:
  ```
  [1] Proxychains Configuration
      knowledge/pentest/tunneling/a1b2-proxychains.md
      ...set up proxychains4 with SOCKS5 proxy on port 1080...

  [2] SSH Dynamic Port Forwarding
      knowledge/pentest/tunneling/c3d4-ssh-tunnel.md
      ...ssh -D 1080 user@pivot for dynamic SOCKS proxy...

  2 results (12ms)
  ```
- Flags: `--limit N` (default 10), `--json` for machine output

### 5. `n reindex` command

**`internal/cli/reindex.go`**
- Walk all `.md` files in vault (excluding `_inbox/`, `_processing/`, `_archive/`, `_meta/`, `_trash/`)
- Parse each into NoteDocument
- Batch upsert to Meilisearch
- Print progress: `indexed 247 notes (1.2s)`

### 6. Config addition

Add `meilisearch` section to config:
```go
type Config struct {
    VaultPath   string          `mapstructure:"vault_path"`
    Model       string          `mapstructure:"model"`
    Meilisearch MeilisearchConfig `mapstructure:"meilisearch"`
}

type MeilisearchConfig struct {
    Host   string `mapstructure:"host"`   // default "http://127.0.0.1:7700"
    APIKey string `mapstructure:"api_key"` // optional
}
```

### 7. Tests

- `internal/search/parser_test.go` — frontmatter parsing round-trip
- `internal/search/query/translate_test.go` — field:value extraction and filter generation
- Integration test (needs running Meilisearch): index, search, verify results

## File list (new)
- `internal/search/client.go`
- `internal/search/document.go`
- `internal/search/parser.go`
- `internal/search/parser_test.go`
- `internal/search/query/translate.go`
- `internal/search/query/translate_test.go`
- `internal/cli/search.go`
- `internal/cli/reindex.go`

## File list (modified)
- `docker-compose.yml` — bump version
- `go.mod` / `go.sum` — add meilisearch-go dependency
- `internal/config/config.go` — add MeilisearchConfig
- `internal/cli/root.go` — register search and reindex commands
