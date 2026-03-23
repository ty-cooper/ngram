# Engineering Standards

Cross-cutting concerns: how the `n` binary calls Claude, how tests work, cost monitoring, and the deterministic linter.

## Claude Code Integration

All AI operations shell out to the `claude` CLI binary. The `n` binary never makes direct Anthropic API HTTP calls.

```go
type ClaudeRunner struct {
    BinaryPath string   // default: "claude" (found via PATH)
    Model      string   // from config
    VaultPath  string   // for CLAUDE.md context
}

func (c *ClaudeRunner) Run(ctx context.Context, prompt string, opts ...RunOption) ([]byte, error) {
    args := []string{
        "-p", prompt,
        "--output-format", "json",
    }
    for _, opt := range opts {
        opt.apply(&args)
    }
    cmd := exec.CommandContext(ctx, c.BinaryPath, args...)
    cmd.Dir = c.VaultPath

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("claude: %w: %s", err, stderr.String())
    }
    return stdout.Bytes(), nil
}
```

### Why This Design

- **Auth handled.** Claude Code manages API keys and session tokens.
- **Retries handled.** Built-in retry logic and rate limit handling.
- **Model management handled.** Latest model automatically.
- **No mock interface needed.** Tests mock at the `exec.Command` level with fixture scripts.

### Usage Across the System

| Component | Call |
|-----------|------|
| Processor (structuring) | `claude -p "Structure this note..." --output-format json` |
| Quality gate | `claude -p "Check this note for quality..." --output-format json` |
| Dedup master agent | `claude -p "Compare this note against these..." --output-format json` |
| Quiz generation | `claude -p "Generate a question from this note..." --output-format json` |
| Quiz grading | `claude -p "Grade this answer..." --output-format json` |
| RAG synthesis | `claude -p "Synthesize an answer from these notes..." --output-format json` |

### MODEL Flag Routing

```
MODEL=cloud  → claude binary (default)
MODEL=local  → ollama binary via same exec pattern
MODEL=off    → skip all AI calls, write raw notes directly
MODEL=mock   → mock-claude test script
```

### Degradation

If `claude` is not in PATH or returns non-zero:

- Processor: notes queue in `_inbox/`
- Quality gate: deterministic linter only
- Quiz generation: template fallback ("Explain [note title] in your own words")
- Quiz grading: self-assessment fallback ("Rate yourself 1-5")
- RAG: Meilisearch results only, no synthesis

`n status` reports: `claude: unavailable (binary not found)` or `claude: error (auth expired)`

## Test Strategy

### Unit Tests (no external services)

- SM-2 algorithm: 30+ scenarios with known inputs/outputs
- Frontmatter parsing: round-trip read/modify/write
- Taxonomy resolution: alias mapping, unknown tag proposal
- Query translation: field:value to Meilisearch filter strings
- Deterministic linter: banned words, em dashes, summary length
- File naming: title to slug conversion

All use `MODEL=mock` with fixture scripts:

```bash
#!/bin/bash
# test-fixtures/mock-claude
echo '{"title":"Raft leader election","domain":"distributed-systems",...}'
```

Set `ClaudeRunner.BinaryPath = "test-fixtures/mock-claude"` in tests. No Go interface mocking needed.

### Integration Tests (need Meilisearch)

Indexing, search, dedup detection, hybrid search, faceted filters.

### End-to-End Tests (need claude binary)

Full pipeline: raw note → structured output → indexed in Meilisearch. Expensive. Run manually or in CI with a budget flag.

## Deterministic Linter

Go function that runs before any Claude call. Catches ~40% of issues at zero cost.

```go
func Lint(note *StructuredNote) []Violation {
    var v []Violation

    if strings.Contains(note.Body, "\u2014") {
        v = append(v, Violation{Rule: "no-em-dash"})
    }

    banned := []string{"essentially", "basically", "simply", "just",
        "actually", "really", "important", "key", "crucial", "significant"}
    for _, word := range banned {
        if containsWord(note.Body, word) {
            v = append(v, Violation{Rule: "no-filler", Word: word})
        }
    }

    starters := []string{"It is ", "There are ", "There is ", "This is ", "Note that "}
    for _, s := range starters {
        if startsAnySentence(note.Body, s) {
            v = append(v, Violation{Rule: "no-weak-starter", Starter: s})
        }
    }

    if len(note.Summary) > 120 {
        v = append(v, Violation{Rule: "summary-too-long", Len: len(note.Summary)})
    }

    return v
}
```

If the linter fails, the note is retried with violation feedback injected into the prompt. Max 2 retries. This saves Claude calls by catching formatting issues before they reach the quality gate.

## Cost Monitoring

Track call counts per invocation to `_meta/api-usage.jsonl`:

```json
{"timestamp": "2026-03-23T10:42:00Z", "component": "processor", "note_id": "a1b2c3d4", "duration_ms": 3200}
```

Daily budget in `~/.ngram.yml`:

```yaml
api:
  max_calls_per_day: 100
  warn_at_percent: 80
```

When the limit is hit, non-critical calls pause (same degradation behavior as binary unavailable).

## Writing Style Requirements

All subagent prompts include these non-negotiable constraints:

```
WRITING RULES (non-negotiable):
- Follow Google Technical Writing standards
- Zero personality. Declarative voice. Present tense. Active voice.
- No hyphens or em dashes anywhere in the output
- No filler words: basically, essentially, actually, interestingly, in order to
- No hedging: might, perhaps, seems, could be
- No conversational tone: "you can", "you'll want to", "let's"
- Minimum words for maximum information transfer
- All commands in fenced code blocks with language identifier
- All code blocks must be directly copyable with no artifacts
- One concept per sentence. One topic per paragraph.
- Do not repeat information present in frontmatter tags.
- If a sentence does not add information, delete it.
```
