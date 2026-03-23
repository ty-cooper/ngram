package search

// NoteDocument is the Meilisearch document schema for indexed vault notes.
type NoteDocument struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Summary     string   `json:"summary,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Domain      string   `json:"domain,omitempty"`
	ContentType string   `json:"content_type,omitempty"`
	Box         string   `json:"box,omitempty"`
	Phase       string   `json:"phase,omitempty"`
	Engagement  string   `json:"engagement,omitempty"`
	FilePath    string   `json:"file_path"`
	Captured    int64    `json:"captured"` // unix timestamp
}
