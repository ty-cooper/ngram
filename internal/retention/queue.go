package retention

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
)

// Note is a minimal representation for queue building.
type Note struct {
	Path         string
	ContentType  string
	Domain       string
	TopicCluster string
	Retention    *Retention
	Created      time.Time
}

// ScheduledQuiz is a note scheduled for delivery at a specific time.
type ScheduledQuiz struct {
	NotePath string `json:"note_path"`
	SendAt   string `json:"send_at"` // HH:MM
	Sent     bool   `json:"sent"`
}

// BuildDailyQueue selects and orders notes for today's quiz session.
func BuildDailyQueue(notes []Note) []Note {
	today := time.Now().Truncate(24 * time.Hour)
	todayStr := today.Format("2006-01-02")

	var overdue, dueToday, newNotes, lapsed []Note

	for _, n := range notes {
		if n.ContentType != "knowledge" || n.Retention == nil {
			continue
		}

		switch {
		case n.Retention.NextReview != "" && n.Retention.NextReview < todayStr:
			overdue = append(overdue, n)
		case n.Retention.NextReview == todayStr:
			dueToday = append(dueToday, n)
		case n.Retention.State == "new":
			newNotes = append(newNotes, n)
		case n.Retention.LapseCount >= 2 && n.Retention.State == "learning":
			lapsed = append(lapsed, n)
		}
	}

	// Sort overdue: most overdue first.
	sort.Slice(overdue, func(i, j int) bool {
		return overdue[i].Retention.NextReview < overdue[j].Retention.NextReview
	})

	// Sort new: oldest first.
	sort.Slice(newNotes, func(i, j int) bool {
		return newNotes[i].Created.Before(newNotes[j].Created)
	})

	// Cap new notes at 3 per day.
	cap := 3
	if len(newNotes) < cap {
		cap = len(newNotes)
	}
	newNotes = newNotes[:cap]

	// Sort lapsed: most lapsed first.
	sort.Slice(lapsed, func(i, j int) bool {
		return lapsed[i].Retention.LapseCount > lapsed[j].Retention.LapseCount
	})

	// Assemble and deduplicate.
	var queue []Note
	queue = append(queue, overdue...)
	queue = append(queue, dueToday...)
	queue = append(queue, newNotes...)
	queue = append(queue, lapsed...)
	queue = deduplicate(queue)

	return Interleave(queue)
}

// Interleave reorders notes so no two consecutive share the same domain:cluster.
func Interleave(notes []Note) []Note {
	if len(notes) <= 1 {
		return notes
	}

	buckets := make(map[string][]Note)
	var keys []string
	for _, n := range notes {
		key := n.Domain + ":" + n.TopicCluster
		if _, exists := buckets[key]; !exists {
			keys = append(keys, key)
		}
		buckets[key] = append(buckets[key], n)
	}

	var result []Note
	for len(buckets) > 0 {
		for _, key := range keys {
			bucket, exists := buckets[key]
			if !exists || len(bucket) == 0 {
				continue
			}
			result = append(result, bucket[0])
			buckets[key] = bucket[1:]
			if len(buckets[key]) == 0 {
				delete(buckets, key)
			}
		}
	}
	return result
}

// ScheduleMessages distributes quiz notes across waking hours.
func ScheduleMessages(queue []Note, wakeHour, sleepHour, minGap, maxGap int) []ScheduledQuiz {
	if len(queue) == 0 {
		return nil
	}

	availableMinutes := (sleepHour - wakeHour) * 60
	baseInterval := float64(availableMinutes) / float64(len(queue))
	currentMinute := float64(wakeHour * 60)
	var schedule []ScheduledQuiz

	for i, note := range queue {
		var sendAt float64
		if i == 0 {
			sendAt = currentMinute + float64(rand.Intn(availableMinutes/4+1))
		} else {
			jitter := (rand.Float64() - 0.5) * 0.8 * baseInterval
			gap := math.Max(float64(minGap), math.Min(float64(maxGap), baseInterval+jitter))
			sendAt = currentMinute + gap
		}
		if int(sendAt) >= sleepHour*60 {
			break
		}

		schedule = append(schedule, ScheduledQuiz{
			NotePath: note.Path,
			SendAt:   formatTime(int(sendAt)),
			Sent:     false,
		})
		currentMinute = sendAt
	}
	return schedule
}

func formatTime(totalMinutes int) string {
	h := totalMinutes / 60
	m := totalMinutes % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

func deduplicate(notes []Note) []Note {
	seen := make(map[string]bool)
	var result []Note
	for _, n := range notes {
		if !seen[n.Path] {
			seen[n.Path] = true
			result = append(result, n)
		}
	}
	return result
}
