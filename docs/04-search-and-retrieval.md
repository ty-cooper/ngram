# Search and Retrieval

Two interfaces for getting knowledge out of the vault: structured search (`nq`) and RAG (`n ask`).

## Meilisearch Setup

Meilisearch runs locally in Docker, bound to localhost only.

```yaml
# docker-compose.yml
services:
  meilisearch:
    image: getmeili/meilisearch:latest
    ports:
      - "127.0.0.1:7700:7700"
    volumes:
      - ./meili_data:/meili_data
    environment:
      - MEILI_NO_ANALYTICS=true
```

All data stays local. The indexer goroutine watches `knowledge/`, `boxes/`, `tools/` via fsnotify and upserts documents on file change.

### Index Schema

```go
type MeilisearchDocument struct {
    ID             string   `json:"id"`
    Title          string   `json:"title"`
    Summary        string   `json:"summary"`
    Body           string   `json:"body"`
    Tags           []string `json:"tags"`
    Path           string   `json:"path"`
    ContentType    string   `json:"content_type"`
    Domain         string   `json:"domain"`
    TopicCluster   string   `json:"topic_cluster"`
    Source         string   `json:"source"`
    SourceType     string   `json:"source_type"`
    RetentionState string   `json:"retention_state"`
    RetentionScore int      `json:"retention_score"`
    NextReview     string   `json:"next_review"`
    LapseCount     int      `json:"lapse_count"`
    Streak         int      `json:"streak"`
    Created        string   `json:"created"`
    Modified       string   `json:"modified"`
}
```

### Meilisearch Configuration

```go
filterableAttributes := []string{
    "tags", "content_type", "domain", "topic_cluster",
    "source", "source_type", "retention_state",
    "retention_score", "next_review", "lapse_count",
    "streak", "created", "modified",
}

sortableAttributes := []string{
    "next_review", "retention_score", "lapse_count",
    "created", "modified",
}

searchableAttributes := []string{
    "title", "summary", "tags", "body",
}
```

Searchable attributes are weighted: title > summary > tags > body.

### Hybrid Search

Meilisearch combines keyword matching and semantic similarity in one request. `FindSimilar()` uses `semanticRatio: 0.7` (70% semantic, 30% keyword) when embeddings are configured. This allows conceptually similar notes (e.g. "nmap -sV" and "nmap -sS") to match even when they share few keywords.

Embedder configuration is automatic during `n up` and `n dream`:
- `model: cloud` — OpenAI `text-embedding-3-small`, requires `OPENAI_API_KEY` env var or `embeddings.openai_api_key` in `~/.ngram.yml`
- `model: off` / no API key — falls back to keyword-only search (no embeddings)

The document template sent to the embedder: `"A {{doc.content_type}} note titled {{doc.title}}. {{doc.summary}} {{doc.body}}"` (max 2000 bytes).

This eliminates the need for a custom embedding store, cosine similarity code, or separate vector DB.

## Structured Search (`nq`)

The `nq` command translates field:value syntax into Meilisearch native filters. This is a thin query translator, not a custom parser.

```bash
nq proxychains socks setup                    # fuzzy full-text
nq domain:pentest tag:nmap firewall            # structured filters
nq content_type:reference domain:cooking       # all recipes
nq content_type:link cluster:videos            # saved video links
nq due:overdue                                 # notes past review date
nq score:<60 content_type:knowledge            # weak knowledge
```

### Query Translation

Field prefixes map directly to Meilisearch filter attributes:

| Prefix | Meilisearch Filter |
|--------|--------------------|
| `domain:X` | `domain = "X"` |
| `tag:X` | `tags = "X"` |
| `content_type:X` | `content_type = "X"` |
| `cluster:X` | `topic_cluster = "X"` |
| `state:X` | `retention_state = "X"` |
| `score:<N` | `retention_score < N` |
| `due:overdue` | `next_review < today` |

Remaining text (no prefix) becomes the search query string. Meilisearch handles fuzzy matching and typo tolerance natively.

## RAG (`n ask`)

Natural language questions answered with source citations.

```bash
n ask "how do I bypass a stateless firewall?"
n ask "what recipes do I have?"
n ask "question" --domain pentest     # domain-restricted
n ask "question" --sources-only       # show matching notes, no synthesis
```

### Default Path

1. Single hybrid search call to Meilisearch (keyword + semantic)
2. Feed top results to Claude for synthesis
3. Every claim cited with `[noteID]`

This is one search round-trip and one LLM call. No query planner needed for most questions.

### Fallback Path

If hybrid search returns < 3 results, escalate to a query planner that generates multiple targeted searches. This handles complex multi-faceted questions where a single query isn't enough.

### Knowledge Gap Detection

When Claude can't answer from existing notes, the gap is logged to `_meta/knowledge-gaps.jsonl`:

```json
{"timestamp": "...", "question": "...", "domain": "pentest", "gap": "No notes on DNS tunneling"}
```

### API Degradation

When Claude is unreachable, `n ask` returns raw Meilisearch results without synthesis. Search still works. Synthesis doesn't.

## Access Points

Search is available from:

- **Terminal**: `nq` command
- **Overlay**: Cmd+Shift+K from any window
- **Obsidian**: Cmd+Shift+F overridden to search Meilisearch
- **iMessage**: `ask: <question>` text command

All hit the same Meilisearch instance on localhost:7700.
