package search

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

const indexName = "notes"

// Client wraps the Meilisearch SDK for vault search operations.
type Client struct {
	ms    meilisearch.ServiceManager
	index meilisearch.IndexManager
}

// New creates a Meilisearch client. Host defaults to http://127.0.0.1:7700.
func New(host, apiKey string) (*Client, error) {
	if host == "" {
		host = "http://127.0.0.1:7700"
	}
	ms := meilisearch.New(host, meilisearch.WithAPIKey(apiKey))
	idx := ms.Index(indexName)
	return &Client{ms: ms, index: idx}, nil
}

// EnsureIndex creates the notes index if it doesn't exist and configures
// searchable, filterable, and sortable attributes.
func (c *Client) EnsureIndex() error {
	_, err := c.ms.CreateIndex(&meilisearch.IndexConfig{
		Uid:        indexName,
		PrimaryKey: "id",
	})
	if err != nil {
		// Index may already exist; Meilisearch enqueues a task regardless.
	}

	task, err := c.index.UpdateSearchableAttributes(&[]string{
		"title",
		"tags",
		"summary",
		"body",
	})
	if err != nil {
		return fmt.Errorf("set searchable attributes: %w", err)
	}
	if err := c.waitForTask(task.TaskUID); err != nil {
		return err
	}

	// UpdateFilterableAttributes takes *[]interface{}.
	filterAttrs := []interface{}{
		"domain",
		"content_type",
		"box",
		"phase",
		"tags",
		"engagement",
	}
	task, err = c.index.UpdateFilterableAttributes(&filterAttrs)
	if err != nil {
		return fmt.Errorf("set filterable attributes: %w", err)
	}
	if err := c.waitForTask(task.TaskUID); err != nil {
		return err
	}

	task, err = c.index.UpdateSortableAttributes(&[]string{
		"captured",
	})
	if err != nil {
		return fmt.Errorf("set sortable attributes: %w", err)
	}
	return c.waitForTask(task.TaskUID)
}

// IndexNote upserts a single document into the notes index.
func (c *Client) IndexNote(doc NoteDocument) error {
	pk := "id"
	task, err := c.index.AddDocuments([]NoteDocument{doc}, &meilisearch.DocumentOptions{PrimaryKey: &pk})
	if err != nil {
		return fmt.Errorf("index note: %w", err)
	}
	return c.waitForTask(task.TaskUID)
}

// IndexNotes batch upserts documents into the notes index.
func (c *Client) IndexNotes(docs []NoteDocument) error {
	if len(docs) == 0 {
		return nil
	}
	pk := "id"
	task, err := c.index.AddDocuments(docs, &meilisearch.DocumentOptions{PrimaryKey: &pk})
	if err != nil {
		return fmt.Errorf("index notes: %w", err)
	}
	return c.waitForTask(task.TaskUID)
}

// DeleteNote removes a document by ID.
func (c *Client) DeleteNote(id string) error {
	task, err := c.index.DeleteDocument(id, nil)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	return c.waitForTask(task.TaskUID)
}

// SearchOptions configures a search query.
type SearchOptions struct {
	Filter string
	Limit  int64
	Offset int64
}

// SearchResult is a single search hit.
type SearchResult struct {
	Title       string `json:"title"`
	FilePath    string `json:"file_path"`
	ContentType string `json:"content_type"`
	Domain      string `json:"domain"`
	Snippet     string `json:"snippet"`
}

// SearchResponse wraps results with metadata.
type SearchResponse struct {
	Results      []SearchResult `json:"results"`
	TotalHits    int64          `json:"total_hits"`
	ProcessingMs int64          `json:"processing_ms"`
}

// Search queries the notes index with optional filters.
func (c *Client) Search(query string, opts SearchOptions) (*SearchResponse, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = 10
	}

	req := &meilisearch.SearchRequest{
		Limit:                 limit,
		Offset:                opts.Offset,
		AttributesToHighlight: []string{"body", "title", "summary"},
		HighlightPreTag:       "",
		HighlightPostTag:      "",
		AttributesToCrop:      []string{"body:80"},
	}
	if opts.Filter != "" {
		req.Filter = opts.Filter
	}

	resp, err := c.index.Search(query, req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	results := make([]SearchResult, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		r := SearchResult{
			Title:       hitStr(hit, "title"),
			FilePath:    hitStr(hit, "file_path"),
			ContentType: hitStr(hit, "content_type"),
			Domain:      hitStr(hit, "domain"),
		}

		// Use cropped body from _formatted if available.
		if formatted := hitObj(hit, "_formatted"); formatted != nil {
			if s := rawStr(formatted, "body"); s != "" {
				r.Snippet = s
			}
		}
		if r.Snippet == "" {
			r.Snippet = truncate(hitStr(hit, "body"), 120)
		}

		results = append(results, r)
	}

	return &SearchResponse{
		Results:      results,
		TotalHits:    resp.EstimatedTotalHits,
		ProcessingMs: resp.ProcessingTimeMs,
	}, nil
}

// Healthy returns true if Meilisearch is reachable.
func (c *Client) Healthy() bool {
	return c.ms.IsHealthy()
}

func (c *Client) waitForTask(taskUID int64) error {
	task, err := c.ms.WaitForTask(taskUID, 50*time.Millisecond)
	if err != nil {
		return fmt.Errorf("wait for task %d: %w", taskUID, err)
	}
	if task.Status == meilisearch.TaskStatusFailed {
		msg := "unknown error"
		if task.Error.Message != "" {
			msg = task.Error.Message
		}
		return fmt.Errorf("task %d failed: %s", taskUID, msg)
	}
	return nil
}

// hitStr extracts a string value from a Hit (map[string]json.RawMessage).
func hitStr(hit meilisearch.Hit, key string) string {
	raw, ok := hit[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

// hitObj extracts a nested object from a Hit.
func hitObj(hit meilisearch.Hit, key string) map[string]json.RawMessage {
	raw, ok := hit[key]
	if !ok {
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	return obj
}

// rawStr extracts a string from a raw JSON map.
func rawStr(m map[string]json.RawMessage, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
