package search

// CommandDocument is the Meilisearch document schema for indexed code blocks.
// Extracted from notes during reindex by parsing fenced code blocks with # [Tool] comments.
type CommandDocument struct {
	ID           string `json:"id"`                     // parent_note_id + "-cmd-N"
	ParentNoteID string `json:"parent_note_id"`         // note this command lives in
	ParentTitle  string `json:"parent_title"`           // title of parent note
	Tool         string `json:"tool"`                   // from # [Tool] comment
	Language     string `json:"language,omitempty"`      // code fence language (powershell, bash, etc)
	Command      string `json:"command"`                // the actual command text
	Description  string `json:"description"`            // # [Tool] rest of comment line
	Phase        string `json:"phase,omitempty"`        // inherited from parent note
	Domain       string `json:"domain,omitempty"`       // inherited from parent note
	Tags         []string `json:"tags,omitempty"`       // inherited from parent note
	FilePath     string `json:"file_path"`              // parent note file path
}

// NoteDocument is the Meilisearch document schema for indexed vault notes.
type NoteDocument struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Body           string   `json:"body"`
	Summary        string   `json:"summary"`
	Tags           []string `json:"tags"`
	Domain         string   `json:"domain,omitempty"`
	TopicCluster   string   `json:"topic_cluster,omitempty"`
	ContentType    string   `json:"content_type"`
	Source         string   `json:"source,omitempty"`
	SourceType     string   `json:"source_type,omitempty"`
	Box            string   `json:"box,omitempty"`
	Phase          string   `json:"phase,omitempty"`
	Engagement     string   `json:"engagement,omitempty"`
	FilePath       string   `json:"file_path"`
	SourceCommand  string   `json:"source_command,omitempty"`
	Tool           string   `json:"tool,omitempty"`
	SessionID      string   `json:"session_id,omitempty"`
	Captured       int64    `json:"captured"`
	RetentionState string   `json:"retention_state,omitempty"`
	RetentionScore int      `json:"retention_score,omitempty"`
	NextReview     string   `json:"next_review,omitempty"`
	LapseCount     int      `json:"lapse_count,omitempty"`
	Streak         int      `json:"streak,omitempty"`
	Created        string   `json:"created,omitempty"`
	Modified       string   `json:"modified,omitempty"`
}
