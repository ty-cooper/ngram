package retention

import (
	"fmt"
	"testing"
	"time"
)

func TestBuildDailyQueue_Priority(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	notes := []Note{
		{Path: "new1.md", ContentType: "knowledge", Domain: "a", Retention: &Retention{State: "new"}, Created: time.Now().Add(-48 * time.Hour)},
		{Path: "overdue.md", ContentType: "knowledge", Domain: "b", Retention: &Retention{State: "reviewing", NextReview: yesterday}},
		{Path: "today.md", ContentType: "knowledge", Domain: "c", Retention: &Retention{State: "reviewing", NextReview: today}},
		{Path: "future.md", ContentType: "knowledge", Domain: "d", Retention: &Retention{State: "reviewing", NextReview: tomorrow}},
		{Path: "reference.md", ContentType: "reference", Domain: "e", Retention: nil},
	}

	queue := BuildDailyQueue(notes)

	// Future and reference should be excluded.
	for _, n := range queue {
		if n.Path == "future.md" {
			t.Error("future note should not be in queue")
		}
		if n.Path == "reference.md" {
			t.Error("reference note should not be in queue")
		}
	}

	// Overdue should be first.
	if len(queue) > 0 && queue[0].Path != "overdue.md" {
		t.Errorf("first in queue should be overdue, got %s", queue[0].Path)
	}
}

func TestBuildDailyQueue_NewCap(t *testing.T) {
	var notes []Note
	for i := 0; i < 10; i++ {
		notes = append(notes, Note{
			Path:        fmt.Sprintf("new%d.md", i),
			ContentType: "knowledge",
			Domain:      fmt.Sprintf("d%d", i),
			Retention:   &Retention{State: "new"},
			Created:     time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}

	queue := BuildDailyQueue(notes)
	if len(queue) > 3 {
		t.Errorf("new notes should be capped at 3, got %d", len(queue))
	}
}

func TestInterleave(t *testing.T) {
	notes := []Note{
		{Domain: "a", TopicCluster: "1"},
		{Domain: "a", TopicCluster: "1"},
		{Domain: "b", TopicCluster: "2"},
		{Domain: "b", TopicCluster: "2"},
	}

	result := Interleave(notes)

	// No two consecutive should have same domain:cluster.
	for i := 1; i < len(result); i++ {
		prevKey := result[i-1].Domain + ":" + result[i-1].TopicCluster
		currKey := result[i].Domain + ":" + result[i].TopicCluster
		if prevKey == currKey {
			t.Errorf("consecutive same domain:cluster at positions %d and %d", i-1, i)
		}
	}
}

func TestInterleave_SingleDomain(t *testing.T) {
	notes := []Note{
		{Domain: "a", TopicCluster: "1"},
		{Domain: "a", TopicCluster: "1"},
	}

	result := Interleave(notes)
	if len(result) != 2 {
		t.Errorf("expected 2 notes, got %d", len(result))
	}
}

func TestScheduleMessages(t *testing.T) {
	notes := []Note{
		{Path: "a.md"}, {Path: "b.md"}, {Path: "c.md"},
	}

	schedule := ScheduleMessages(notes, 8, 22, 30, 180)

	if len(schedule) == 0 {
		t.Fatal("schedule should not be empty")
	}
	if len(schedule) > 3 {
		t.Errorf("schedule should have at most 3 entries, got %d", len(schedule))
	}

	for _, sq := range schedule {
		if sq.NotePath == "" {
			t.Error("empty note path in schedule")
		}
		if sq.SendAt == "" {
			t.Error("empty send time in schedule")
		}
	}
}

func TestScheduleMessages_Empty(t *testing.T) {
	schedule := ScheduleMessages(nil, 8, 22, 30, 180)
	if schedule != nil {
		t.Error("empty queue should return nil schedule")
	}
}
