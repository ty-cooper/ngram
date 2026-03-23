package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendJSONL(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "_meta"), 0o755)

	entry1 := map[string]string{"key": "value1"}
	entry2 := map[string]string{"key": "value2"}

	if err := AppendJSONL(dir, "test.jsonl", entry1); err != nil {
		t.Fatalf("AppendJSONL 1: %v", err)
	}
	if err := AppendJSONL(dir, "test.jsonl", entry2); err != nil {
		t.Fatalf("AppendJSONL 2: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "_meta", "test.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestWriteAndReadJSON(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "_meta"), 0o755)

	stats := StatsCache{
		TotalNotes:     42,
		KnowledgeNotes: 30,
		DueToday:       5,
	}

	if err := WriteStatsCache(dir, stats); err != nil {
		t.Fatalf("WriteStatsCache: %v", err)
	}

	got, err := ReadStatsCache(dir)
	if err != nil {
		t.Fatalf("ReadStatsCache: %v", err)
	}
	if got.TotalNotes != 42 {
		t.Errorf("TotalNotes = %d, want 42", got.TotalNotes)
	}
	if got.DueToday != 5 {
		t.Errorf("DueToday = %d, want 5", got.DueToday)
	}
}

func TestLogQuizSession(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "_meta"), 0o755)

	session := QuizSession{
		SessionID:       "2026-03-22T14:30:00Z",
		NotesQuizzed:    11,
		DomainScores:    map[string]int{"pentest": 85, "dist-sys": 78},
		AvgScore:        75,
		DurationSeconds: 900,
	}

	if err := LogQuizSession(dir, session); err != nil {
		t.Fatalf("LogQuizSession: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "_meta", "quiz-history.jsonl"))
	var parsed QuizSession
	if err := json.Unmarshal(data[:len(data)-1], &parsed); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.NotesQuizzed != 11 {
		t.Errorf("NotesQuizzed = %d", parsed.NotesQuizzed)
	}
}
