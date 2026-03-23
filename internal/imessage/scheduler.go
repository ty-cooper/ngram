package imessage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tylercooper/ngram/internal/vault"
)

// DeliveryState is persisted to _meta/quiz-delivery-state.json.
type DeliveryState struct {
	Date           string            `json:"date"`
	Schedule       []ScheduleEntry   `json:"schedule"`
	Pending        map[string]Pending `json:"pending"`
	CompletedToday int               `json:"completed_today"`
	SkippedToday   int               `json:"skipped_today"`
	TimedOutToday  int               `json:"timed_out_today"`
}

// ScheduleEntry is a single scheduled quiz delivery.
type ScheduleEntry struct {
	NotePath string `json:"note_path"`
	SendAt   string `json:"send_at"` // HH:MM
	Sent     bool   `json:"sent"`
}

// Pending is an awaiting-reply quiz question.
type Pending struct {
	SentAt    string `json:"sent_at"`
	TimeoutAt string `json:"timeout_at"`
	Question  string `json:"question"`
}

// Scheduler manages quiz delivery via iMessage.
type Scheduler struct {
	VaultPath    string
	Phone        string
	Bridge       MessageBridge
	TimeoutHours int
	WakeHour     int
	SleepHour    int

	state    *DeliveryState
	lastPoll time.Time
}

// NewScheduler creates an iMessage quiz scheduler.
func NewScheduler(vaultPath, phone string, bridge MessageBridge) *Scheduler {
	return &Scheduler{
		VaultPath:    vaultPath,
		Phone:        phone,
		Bridge:       bridge,
		TimeoutHours: 4,
		WakeHour:     8,
		SleepHour:    22,
		lastPoll:     time.Now(),
	}
}

// Run starts the scheduler loop. Blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) error {
	s.loadState()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.saveState()
			return ctx.Err()
		case <-ticker.C:
			s.tick()
		}
	}
}

func (s *Scheduler) tick() {
	now := time.Now()

	// Check engagement mode.
	if s.isEngaged() {
		return
	}

	// Check for scheduled sends.
	if s.state != nil {
		nowTime := now.Format("15:04")
		for i, entry := range s.state.Schedule {
			if entry.Sent || entry.SendAt > nowTime {
				continue
			}
			s.sendQuestion(i)
		}
	}

	// Check timeouts.
	if s.state != nil {
		for path, p := range s.state.Pending {
			timeout, _ := time.Parse(time.RFC3339, p.TimeoutAt)
			if now.After(timeout) {
				log.Printf("ngram: quiz timeout for %s", filepath.Base(path))
				delete(s.state.Pending, path)
				s.state.TimedOutToday++
				// Send the answer.
				s.Bridge.Send(s.Phone, fmt.Sprintf("Time's up. The answer was in: %s", filepath.Base(path)))
			}
		}
	}

	// Poll for replies.
	msgs, err := s.Bridge.Poll(s.lastPoll)
	if err != nil {
		log.Printf("warn: imessage poll: %v", err)
		return
	}
	s.lastPoll = now

	for _, msg := range msgs {
		if msg.From != s.Phone {
			continue
		}
		s.handleReply(msg)
	}

	s.saveState()
}

func (s *Scheduler) sendQuestion(idx int) {
	if s.state == nil || idx >= len(s.state.Schedule) {
		return
	}

	entry := &s.state.Schedule[idx]
	question := fmt.Sprintf("Ngram Quiz\n[%s]\n\nExplain the key concepts from this note.\n\nReply with your answer. Skip: reply \"skip\"",
		filepath.Base(entry.NotePath))

	if err := s.Bridge.Send(s.Phone, question); err != nil {
		log.Printf("warn: send quiz: %v", err)
		return
	}

	entry.Sent = true
	now := time.Now()
	if s.state.Pending == nil {
		s.state.Pending = make(map[string]Pending)
	}
	s.state.Pending[entry.NotePath] = Pending{
		SentAt:    now.Format(time.RFC3339),
		TimeoutAt: now.Add(time.Duration(s.TimeoutHours) * time.Hour).Format(time.RFC3339),
		Question:  question,
	}

	log.Printf("ngram: sent quiz for %s", filepath.Base(entry.NotePath))
}

func (s *Scheduler) handleReply(msg IncomingMessage) {
	body := strings.TrimSpace(msg.Body)
	lower := strings.ToLower(body)

	// Special commands.
	switch lower {
	case "skip":
		s.state.SkippedToday++
		s.clearOldestPending()
		s.Bridge.Send(s.Phone, "Skipped. Grade: 0.")
		return
	case "idk":
		s.state.SkippedToday++
		s.clearOldestPending()
		s.Bridge.Send(s.Phone, "No worries. Review the note when you can.")
		return
	case "pause":
		s.Bridge.Send(s.Phone, "Quizzes paused for today.")
		return
	case "resume":
		s.Bridge.Send(s.Phone, "Quizzes resumed.")
		return
	case "stats":
		stats := fmt.Sprintf("Today: %d completed, %d skipped, %d timed out",
			s.state.CompletedToday, s.state.SkippedToday, s.state.TimedOutToday)
		s.Bridge.Send(s.Phone, stats)
		return
	}

	// Handle as quiz answer (FIFO — oldest pending).
	if len(s.state.Pending) > 0 {
		s.state.CompletedToday++
		s.clearOldestPending()
		// Grading happens asynchronously; send ack immediately.
		s.Bridge.Send(s.Phone, "Received. Grading...")
	}
}

func (s *Scheduler) clearOldestPending() {
	if s.state == nil || len(s.state.Pending) == 0 {
		return
	}
	var oldest string
	var oldestTime time.Time
	for path, p := range s.state.Pending {
		sent, _ := time.Parse(time.RFC3339, p.SentAt)
		if oldest == "" || sent.Before(oldestTime) {
			oldest = path
			oldestTime = sent
		}
	}
	if oldest != "" {
		delete(s.state.Pending, oldest)
	}
}

func (s *Scheduler) isEngaged() bool {
	path := filepath.Join(s.VaultPath, "_meta", "engagement.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var state struct {
		Engaged bool `json:"engaged"`
	}
	json.Unmarshal(data, &state)
	return state.Engaged
}

func (s *Scheduler) loadState() {
	path := filepath.Join(s.VaultPath, "_meta", "quiz-delivery-state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		s.state = &DeliveryState{
			Date:    time.Now().Format("2006-01-02"),
			Pending: make(map[string]Pending),
		}
		return
	}

	var state DeliveryState
	if json.Unmarshal(data, &state) != nil {
		s.state = &DeliveryState{
			Date:    time.Now().Format("2006-01-02"),
			Pending: make(map[string]Pending),
		}
		return
	}

	today := time.Now().Format("2006-01-02")
	if state.Date != today {
		// Stale state — timeout any leftovers and start fresh.
		s.state = &DeliveryState{
			Date:    today,
			Pending: make(map[string]Pending),
		}
		return
	}

	s.state = &state
	if s.state.Pending == nil {
		s.state.Pending = make(map[string]Pending)
	}
}

func (s *Scheduler) saveState() {
	if s.state == nil {
		return
	}
	vault.WriteJSON(s.VaultPath, "quiz-delivery-state.json", s.state)
}
