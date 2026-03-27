package search

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

const (
	indexName    = "notes"
	cmdIndexName = "commands"
)

// EmbedderConfig holds settings for configuring Meilisearch's built-in embedder.
type EmbedderConfig struct {
	Source string // "openAi" or "" (disabled)
	Model  string
	APIKey string
}

// Client wraps the Meilisearch SDK for vault search operations.
type Client struct {
	ms             meilisearch.ServiceManager
	index          meilisearch.IndexManager
	cmdIndex       meilisearch.IndexManager
	hybridEnabled  bool
}

// New creates a Meilisearch client. Host defaults to http://127.0.0.1:7700.
func New(host, apiKey string) (*Client, error) {
	if host == "" {
		host = "http://127.0.0.1:7700"
	}
	ms := meilisearch.New(host, meilisearch.WithAPIKey(apiKey))
	idx := ms.Index(indexName)
	cmdIdx := ms.Index(cmdIndexName)
	return &Client{ms: ms, index: idx, cmdIndex: cmdIdx}, nil
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
		"topic_cluster",
		"box",
		"phase",
		"tags",
		"engagement",
		"retention_state",
		"retention_score",
		"next_review",
		"lapse_count",
		"streak",
		"created",
		"modified",
		"source_command",
		"tool",
		"session_id",
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
		"next_review",
		"retention_score",
		"lapse_count",
		"created",
		"modified",
	})
	if err != nil {
		return fmt.Errorf("set sortable attributes: %w", err)
	}
	return c.waitForTask(task.TaskUID)
}

// EnsureCommandsIndex creates the commands index with appropriate settings.
func (c *Client) EnsureCommandsIndex() error {
	_, err := c.ms.CreateIndex(&meilisearch.IndexConfig{
		Uid:        cmdIndexName,
		PrimaryKey: "id",
	})
	if err != nil {
		// May already exist.
	}

	task, err := c.cmdIndex.UpdateSearchableAttributes(&[]string{
		"command",
		"description",
		"tool",
		"parent_title",
	})
	if err != nil {
		return fmt.Errorf("set command searchable attributes: %w", err)
	}
	if err := c.waitForTask(task.TaskUID); err != nil {
		return err
	}

	filterAttrs := []interface{}{
		"tool",
		"language",
		"phase",
		"domain",
		"tags",
		"parent_note_id",
	}
	task, err = c.cmdIndex.UpdateFilterableAttributes(&filterAttrs)
	if err != nil {
		return fmt.Errorf("set command filterable attributes: %w", err)
	}
	if err := c.waitForTask(task.TaskUID); err != nil {
		return err
	}

	// Enable facets for the info modal.
	task, err = c.cmdIndex.UpdateSortableAttributes(&[]string{
		"tool",
		"phase",
	})
	if err != nil {
		return fmt.Errorf("set command sortable attributes: %w", err)
	}
	return c.waitForTask(task.TaskUID)
}

// IndexCommands upserts a batch of command documents.
func (c *Client) IndexCommands(docs []CommandDocument) error {
	if len(docs) == 0 {
		return nil
	}
	pk := "id"
	task, err := c.cmdIndex.AddDocuments(docs, &meilisearch.DocumentOptions{PrimaryKey: &pk})
	if err != nil {
		return fmt.Errorf("index commands: %w", err)
	}
	return c.waitForTask(task.TaskUID)
}

// SearchCommands queries the commands index with optional filters.
func (c *Client) SearchCommands(query string, filters []string, limit int) ([]CommandDocument, error) {
	if limit <= 0 {
		limit = 20
	}

	req := &meilisearch.SearchRequest{
		Limit: int64(limit),
	}

	if len(filters) > 0 {
		combined := ""
		for i, f := range filters {
			if i > 0 {
				combined += " AND "
			}
			combined += f
		}
		req.Filter = combined
	}

	resp, err := c.cmdIndex.Search(query, req)
	if err != nil {
		return nil, fmt.Errorf("search commands: %w", err)
	}

	var results []CommandDocument
	for _, hit := range resp.Hits {
		raw, err := json.Marshal(hit)
		if err != nil {
			continue
		}
		var doc CommandDocument
		if err := json.Unmarshal(raw, &doc); err != nil {
			continue
		}
		results = append(results, doc)
	}
	return results, nil
}

// parseFacets extracts facet values from a raw JSON FacetDistribution response.
func parseFacets(raw json.RawMessage, fields []string) map[string][]string {
	result := make(map[string][]string)
	if len(raw) == 0 {
		return result
	}
	var dist map[string]map[string]int
	if err := json.Unmarshal(raw, &dist); err != nil {
		return result
	}
	for _, field := range fields {
		if counts, ok := dist[field]; ok {
			for val := range counts {
				result[field] = append(result[field], val)
			}
		}
	}
	return result
}

// CommandFacets returns distinct values for faceted fields in the commands index.
func (c *Client) CommandFacets() (map[string][]string, error) {
	fields := []string{"tool", "phase", "domain", "tags", "language"}
	resp, err := c.cmdIndex.Search("", &meilisearch.SearchRequest{
		Facets: fields,
		Limit:  0,
	})
	if err != nil {
		return nil, err
	}
	return parseFacets(resp.FacetDistribution, fields), nil
}

// NotesFacets returns distinct values for faceted fields in the notes index.
func (c *Client) NotesFacets() (map[string][]string, error) {
	fields := []string{"domain", "phase", "tags", "tool", "content_type"}
	resp, err := c.index.Search("", &meilisearch.SearchRequest{
		Facets: fields,
		Limit:  0,
	})
	if err != nil {
		return nil, err
	}
	return parseFacets(resp.FacetDistribution, fields), nil
}

// ClearCommandsIndex removes all documents from the commands index.
func (c *Client) ClearCommandsIndex() error {
	task, err := c.cmdIndex.DeleteAllDocuments(nil)
	if err != nil {
		return err
	}
	return c.waitForTask(task.TaskUID)
}

// ConfigureEmbedder sets up Meilisearch's built-in embedder for hybrid search.
// If cfg.Source is empty, embeddings are disabled and search falls back to keyword-only.
func (c *Client) ConfigureEmbedder(cfg EmbedderConfig) error {
	if cfg.Source == "" {
		return nil
	}

	embedder := meilisearch.Embedder{
		Source: meilisearch.EmbedderSource(cfg.Source),
		Model:  cfg.Model,
		DocumentTemplate: "A {{doc.content_type}} note titled {{doc.title}}. " +
			"{{doc.summary}} {{doc.body}}",
		DocumentTemplateMaxBytes: 2000,
	}
	if cfg.APIKey != "" {
		embedder.APIKey = cfg.APIKey
	}

	task, err := c.index.UpdateEmbedders(map[string]meilisearch.Embedder{
		"default": embedder,
	})
	if err != nil {
		return fmt.Errorf("configure embedder: %w", err)
	}
	if err := c.waitForTask(task.TaskUID); err != nil {
		return err
	}
	c.hybridEnabled = true
	return nil
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

// SimilarNote is a search hit with a ranking score for dedup comparison.
type SimilarNote struct {
	ID         string
	Title      string
	Body       string
	Summary    string
	Domain     string
	Box        string
	Engagement string
	FilePath   string
	Score      float64
}

// FindSimilar returns the top N most similar notes to the given query text.
// Uses hybrid search (keyword + semantic) when embeddings are configured,
// falls back to keyword-only otherwise.
func (c *Client) FindSimilar(query string, limit int64) ([]SimilarNote, error) {
	if limit == 0 {
		limit = 5
	}

	req := &meilisearch.SearchRequest{
		Limit:                limit,
		ShowRankingScore:     true,
		AttributesToRetrieve: []string{"id", "title", "body", "summary", "domain", "file_path"},
	}
	if c.hybridEnabled {
		req.Hybrid = &meilisearch.SearchRequestHybrid{
			SemanticRatio: 0.7,
			Embedder:      "default",
		}
	}

	resp, err := c.index.Search(query, req)
	if err != nil {
		return nil, fmt.Errorf("find similar: %w", err)
	}

	results := make([]SimilarNote, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		score := hitFloat(hit, "_rankingScore")
		results = append(results, SimilarNote{
			ID:       hitStr(hit, "id"),
			Title:    hitStr(hit, "title"),
			Body:     hitStr(hit, "body"),
			Summary:  hitStr(hit, "summary"),
			Domain:   hitStr(hit, "domain"),
			FilePath: hitStr(hit, "file_path"),
			Score:    score,
		})
	}

	return results, nil
}

// FindSimilarFiltered returns similar notes with a Meilisearch filter applied.
// Use filter like `box != "current-box"` for cross-engagement recall.
func (c *Client) FindSimilarFiltered(query string, limit int64, filter string) ([]SimilarNote, error) {
	if limit == 0 {
		limit = 5
	}

	req := &meilisearch.SearchRequest{
		Limit:                limit,
		ShowRankingScore:     true,
		AttributesToRetrieve: []string{"id", "title", "body", "summary", "domain", "box", "engagement", "file_path"},
	}
	if filter != "" {
		req.Filter = filter
	}
	if c.hybridEnabled {
		req.Hybrid = &meilisearch.SearchRequestHybrid{
			SemanticRatio: 0.7,
			Embedder:      "default",
		}
	}

	resp, err := c.index.Search(query, req)
	if err != nil {
		return nil, fmt.Errorf("find similar filtered: %w", err)
	}

	results := make([]SimilarNote, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		score := hitFloat(hit, "_rankingScore")
		results = append(results, SimilarNote{
			ID:         hitStr(hit, "id"),
			Title:      hitStr(hit, "title"),
			Body:       hitStr(hit, "body"),
			Summary:    hitStr(hit, "summary"),
			Domain:     hitStr(hit, "domain"),
			Box:        hitStr(hit, "box"),
			Engagement: hitStr(hit, "engagement"),
			FilePath:   hitStr(hit, "file_path"),
			Score:      score,
		})
	}

	return results, nil
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

// hitFloat extracts a float64 value from a Hit.
func hitFloat(hit meilisearch.Hit, key string) float64 {
	raw, ok := hit[key]
	if !ok {
		return 0
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return 0
	}
	return f
}

// ListAllIDs returns all document IDs in the index for reconciliation.
func (c *Client) ListAllIDs() (map[string]bool, error) {
	ids := make(map[string]bool)
	var offset int64
	for {
		req := &meilisearch.DocumentsQuery{
			Limit:  1000,
			Offset: offset,
			Fields: []string{"id"},
		}
		var result meilisearch.DocumentsResult
		if err := c.index.GetDocuments(req, &result); err != nil {
			return nil, fmt.Errorf("get documents: %w", err)
		}
		for _, doc := range result.Results {
			raw, ok := doc["id"]
			if !ok {
				continue
			}
			var s string
			if err := json.Unmarshal(raw, &s); err == nil {
				ids[s] = true
			}
		}
		if int64(len(result.Results)) < 1000 {
			break
		}
		offset += 1000
	}
	return ids, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
