package search

// NoteDocument is the Meilisearch document schema for indexed vault notes.
type NoteDocument struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Body           string   `json:"body"`
	Summary        string   `json:"summary,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Domain         string   `json:"domain,omitempty"`
	TopicCluster   string   `json:"topic_cluster,omitempty"`
	ContentType    string   `json:"content_type,omitempty"`
	Source         string   `json:"source,omitempty"`
	SourceType     string   `json:"source_type,omitempty"`
	Box            string   `json:"box,omitempty"`
	Phase          string   `json:"phase,omitempty"`
	Engagement     string   `json:"engagement,omitempty"`
	FilePath       string   `json:"file_path"`
	Captured       int64    `json:"captured"`
	RetentionState string   `json:"retention_state,omitempty"`
	RetentionScore int      `json:"retention_score,omitempty"`
	NextReview     string   `json:"next_review,omitempty"`
	LapseCount     int      `json:"lapse_count,omitempty"`
	Streak         int      `json:"streak,omitempty"`
	Created        string   `json:"created,omitempty"`
	Modified       string   `json:"modified,omitempty"`
}
