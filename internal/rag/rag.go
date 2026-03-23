package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ty-cooper/ngram/internal/llm"
	"github.com/ty-cooper/ngram/internal/search"
	"github.com/ty-cooper/ngram/internal/vault"
)

// Engine handles RAG queries: search → retrieve → synthesize.
type Engine struct {
	SearchClient *search.Client
	Runner       *llm.Runner
	VaultPath    string
}

// AskOptions configures an ask query.
type AskOptions struct {
	Domain      string
	SourcesOnly bool
	Verbose     bool
	Limit       int
}

// AskResult is the response from an ask query.
type AskResult struct {
	Answer      string
	Sources     []Source
	GapDetected bool
}

// Source is a cited note.
type Source struct {
	ID       string
	Title    string
	FilePath string
}

// Ask runs a RAG query: search vault, synthesize answer with citations.
func (e *Engine) Ask(ctx context.Context, question string, opts AskOptions) (*AskResult, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = 10
	}

	// Build search filter.
	filter := ""
	if opts.Domain != "" {
		filter = fmt.Sprintf(`domain = "%s"`, opts.Domain)
	}

	// Search vault.
	resp, err := e.SearchClient.Search(question, search.SearchOptions{
		Filter: filter,
		Limit:  int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	// Knowledge gap detection.
	if len(resp.Results) == 0 {
		e.logGap(question, opts.Domain)
		return &AskResult{
			Answer:      "No relevant notes found in the vault.",
			GapDetected: true,
		}, nil
	}

	// Build sources list.
	sources := make([]Source, len(resp.Results))
	for i, r := range resp.Results {
		sources[i] = Source{
			ID:       extractID(r.FilePath),
			Title:    r.Title,
			FilePath: r.FilePath,
		}
	}

	// Sources-only mode.
	if opts.SourcesOnly {
		return &AskResult{Sources: sources}, nil
	}

	// Retrieve full note content for top results.
	noteContents := e.retrieveNotes(resp.Results)

	// Synthesize with Claude.
	answer, err := e.synthesize(ctx, question, noteContents, sources)
	if err != nil {
		// API degradation: fall back to sources-only.
		log.Printf("warn: LLM unavailable for synthesis: %v", err)
		return &AskResult{
			Answer:  "LLM unavailable. Showing search results only.",
			Sources: sources,
		}, nil
	}

	// Check if results were too few — trigger planner fallback.
	if len(resp.Results) < 3 && !opts.SourcesOnly {
		// Planner fallback: decompose and re-search.
		plannerResult, err := e.plannerFallback(ctx, question, opts, noteContents, sources)
		if err == nil && plannerResult != nil {
			return plannerResult, nil
		}
		// If planner fails, return what we have.
	}

	return &AskResult{
		Answer:  answer,
		Sources: sources,
	}, nil
}

func (e *Engine) synthesize(ctx context.Context, question string, noteContents []noteContent, sources []Source) (string, error) {
	prompt := buildSynthesisPrompt(question, noteContents)
	out, err := e.Runner.Run(ctx, prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (e *Engine) plannerFallback(ctx context.Context, question string, opts AskOptions, existingNotes []noteContent, existingSources []Source) (*AskResult, error) {
	// Ask Claude to decompose the question into targeted queries.
	planPrompt := fmt.Sprintf(`The following question returned fewer than 3 results from the knowledge vault:

Question: %s

Generate 2-3 alternative search queries that might find relevant notes. Return ONLY a JSON array of strings.`, question)

	out, err := e.Runner.Run(ctx, planPrompt)
	if err != nil {
		return nil, err
	}

	var queries []string
	if err := json.Unmarshal(stripCodeFences(out), &queries); err != nil {
		return nil, err
	}

	// Run additional searches.
	allNotes := make(map[string]noteContent)
	allSources := make(map[string]Source)

	for _, n := range existingNotes {
		allNotes[n.id] = n
	}
	for _, s := range existingSources {
		allSources[s.ID] = s
	}

	filter := ""
	if opts.Domain != "" {
		filter = fmt.Sprintf(`domain = "%s"`, opts.Domain)
	}

	for _, q := range queries {
		resp, err := e.SearchClient.Search(q, search.SearchOptions{
			Filter: filter,
			Limit:  5,
		})
		if err != nil {
			continue
		}
		for _, r := range resp.Results {
			id := extractID(r.FilePath)
			if _, exists := allNotes[id]; !exists {
				allSources[id] = Source{ID: id, Title: r.Title, FilePath: r.FilePath}
			}
		}
		retrieved := e.retrieveNotes(resp.Results)
		for _, n := range retrieved {
			allNotes[n.id] = n
		}
	}

	// Flatten.
	var notes []noteContent
	var sources []Source
	for _, n := range allNotes {
		notes = append(notes, n)
	}
	for _, s := range allSources {
		sources = append(sources, s)
	}

	answer, err := e.synthesize(ctx, question, notes, sources)
	if err != nil {
		return nil, err
	}

	return &AskResult{Answer: answer, Sources: sources}, nil
}

type noteContent struct {
	id    string
	title string
	body  string
}

func (e *Engine) retrieveNotes(results []search.SearchResult) []noteContent {
	var notes []noteContent
	for _, r := range results {
		path := filepath.Join(e.VaultPath, r.FilePath)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		id := extractID(r.FilePath)
		notes = append(notes, noteContent{
			id:    id,
			title: r.Title,
			body:  string(data),
		})
	}
	return notes
}

func buildSynthesisPrompt(question string, notes []noteContent) string {
	var b strings.Builder
	b.WriteString("You are a knowledge synthesis agent for the Ngram system.\n\n")
	b.WriteString("Answer the following question using ONLY the provided notes.\n")
	b.WriteString("Every factual claim MUST include an inline citation [noteID].\n")
	b.WriteString("After the answer, include a Sources section listing each cited note.\n\n")
	fmt.Fprintf(&b, "QUESTION: %s\n\n", question)
	b.WriteString("NOTES:\n\n")

	for _, n := range notes {
		fmt.Fprintf(&b, "--- [%s] %s ---\n%s\n\n", n.id, n.title, truncate(n.body, 2000))
	}

	b.WriteString("Respond with a clear, concise answer with [noteID] citations inline.")
	return b.String()
}

func (e *Engine) logGap(question, domain string) {
	entry := map[string]interface{}{
		"question":        question,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"domain_searched": domain,
		"results_found":   0,
	}
	vault.AppendJSONL(e.VaultPath, "knowledge-gaps.jsonl", entry)
}

func extractID(filePath string) string {
	base := filepath.Base(filePath)
	base = strings.TrimSuffix(base, ".md")
	if idx := strings.Index(base, "-"); idx > 0 {
		return base[:idx]
	}
	return base
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func stripCodeFences(data []byte) []byte {
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		if strings.HasSuffix(s, "```") {
			s = s[:len(s)-3]
		}
		s = strings.TrimSpace(s)
	}
	return []byte(s)
}
