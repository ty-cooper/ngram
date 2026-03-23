package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppendJSONL appends a JSON line to a .jsonl file in _meta/.
func AppendJSONL(vaultPath, filename string, entry interface{}) error {
	metaDir := filepath.Join(vaultPath, "_meta")
	if err := EnsureDir(metaDir); err != nil {
		return err
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(metaDir, filename), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(data)
	f.Write([]byte("\n"))
	return nil
}

// WriteJSON writes a JSON file to _meta/.
func WriteJSON(vaultPath, filename string, data interface{}) error {
	metaDir := filepath.Join(vaultPath, "_meta")
	if err := EnsureDir(metaDir); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(metaDir, filename), raw, 0o644)
}

// ReadJSON reads a JSON file from _meta/ into the target.
func ReadJSON(vaultPath, filename string, target interface{}) error {
	path := filepath.Join(vaultPath, "_meta", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// QuizSession is a single quiz session record for quiz-history.jsonl.
type QuizSession struct {
	SessionID       string         `json:"session_id"`
	NotesQuizzed    int            `json:"notes_quizzed"`
	DomainScores    map[string]int `json:"domain_scores"`
	AvgScore        int            `json:"avg_score"`
	DurationSeconds int            `json:"duration_seconds"`
}

// RetentionSnapshot is the daily retention stats for retention-snapshot.json.
type RetentionSnapshot struct {
	Date          string         `json:"date"`
	TotalNotes    int            `json:"total_notes"`
	ByState       map[string]int `json:"by_state"`
	ByDomain      map[string]int `json:"by_domain"`
	DueToday      int            `json:"due_today"`
	Overdue       int            `json:"overdue"`
	AvgScore      int            `json:"avg_score"`
	StreakDays    int            `json:"streak_days"`
}

// StatsCache is precomputed stats for the n stats command.
type StatsCache struct {
	TotalNotes      int                       `json:"total_notes"`
	KnowledgeNotes  int                       `json:"knowledge_notes"`
	DomainBreakdown map[string]DomainStats    `json:"domain_breakdown"`
	DueToday        int                       `json:"due_today"`
	Overdue         int                       `json:"overdue"`
	QuizStreak      int                       `json:"quiz_streak"`
	LastQuiz        string                    `json:"last_quiz"`
}

// DomainStats holds per-domain retention statistics.
type DomainStats struct {
	NoteCount      int `json:"note_count"`
	AvgScore       int `json:"avg_score"`
	DueCount       int `json:"due_count"`
	LearningCount  int `json:"learning_count"`
	ReviewingCount int `json:"reviewing_count"`
	SolidifiedCount int `json:"solidified_count"`
}

// LogQuizSession appends a quiz session to quiz-history.jsonl.
func LogQuizSession(vaultPath string, session QuizSession) error {
	return AppendJSONL(vaultPath, "quiz-history.jsonl", session)
}

// WriteRetentionSnapshot writes the daily retention snapshot.
func WriteRetentionSnapshot(vaultPath string, snap RetentionSnapshot) error {
	return WriteJSON(vaultPath, "retention-snapshot.json", snap)
}

// WriteStatsCache writes the precomputed stats cache.
func WriteStatsCache(vaultPath string, stats StatsCache) error {
	return WriteJSON(vaultPath, "stats-cache.json", stats)
}

// ReadStatsCache reads the precomputed stats cache.
func ReadStatsCache(vaultPath string) (*StatsCache, error) {
	var stats StatsCache
	if err := ReadJSON(vaultPath, "stats-cache.json", &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}
