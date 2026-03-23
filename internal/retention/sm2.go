package retention

import (
	"math"
	"time"
)

// Grade scale (SM-2 standard):
// 0 = complete blackout
// 1 = wrong, recognized topic when shown answer
// 2 = wrong, correct answer felt familiar
// 3 = correct with significant difficulty
// 4 = correct with some hesitation
// 5 = perfect recall, immediate

// Retention holds the spaced repetition state for a note.
type Retention struct {
	State           string  `yaml:"state"`            // new, learning, reviewing, solidified
	EaseFactor      float64 `yaml:"ease_factor"`
	IntervalDays    int     `yaml:"interval_days"`
	RepetitionCount int     `yaml:"repetition_count"`
	LastReviewed    string  `yaml:"last_reviewed"`    // RFC3339 or null
	NextReview      string  `yaml:"next_review"`      // 2006-01-02
	TotalReviews    int     `yaml:"total_reviews"`
	TotalCorrect    int     `yaml:"total_correct"`
	RetentionScore  int     `yaml:"retention_score"`
	Streak          int     `yaml:"streak"`
	LapseCount      int     `yaml:"lapse_count"`
}

// NewRetention creates default retention state for a new knowledge note.
func NewRetention(created time.Time) *Retention {
	return &Retention{
		State:      "new",
		EaseFactor: 2.5,
		NextReview: created.Format("2006-01-02"),
	}
}

// Update applies an SM-2 grade to the retention state.
func (r *Retention) Update(grade int) {
	r.TotalReviews++

	if grade >= 3 { // PASS
		r.TotalCorrect++
		r.Streak++

		switch r.State {
		case "new":
			r.State = "learning"
			r.IntervalDays = 1
			r.RepetitionCount = 1

		case "learning":
			switch {
			case r.RepetitionCount == 0:
				r.IntervalDays = 1
			case r.RepetitionCount == 1:
				r.IntervalDays = 3
			default:
				r.State = "reviewing"
				r.IntervalDays = 7
			}
			r.RepetitionCount++

		case "reviewing":
			r.RepetitionCount++
			r.IntervalDays = int(math.Round(float64(r.IntervalDays) * r.EaseFactor))
			if r.IntervalDays > 90 {
				r.IntervalDays = 90
			}
			if r.Streak >= 5 && r.IntervalDays >= 30 {
				r.State = "solidified"
			}

		case "solidified":
			r.RepetitionCount++
			r.IntervalDays = int(math.Round(float64(r.IntervalDays) * r.EaseFactor))
			if r.IntervalDays > 90 {
				r.IntervalDays = 90
			}
		}

		// SM-2 ease factor update.
		r.EaseFactor += 0.1 - float64(5-grade)*(0.08+float64(5-grade)*0.02)
		if r.EaseFactor < 1.3 {
			r.EaseFactor = 1.3
		}

	} else { // FAIL (grade 0-2)
		r.Streak = 0
		r.LapseCount++
		if r.State == "reviewing" || r.State == "solidified" {
			r.State = "learning"
			r.RepetitionCount = 0
		}
		r.IntervalDays = 1
		r.EaseFactor -= 0.2
		if r.EaseFactor < 1.3 {
			r.EaseFactor = 1.3
		}
	}

	if r.TotalReviews > 0 {
		r.RetentionScore = int(math.Round(float64(r.TotalCorrect) / float64(r.TotalReviews) * 100))
	}
	r.NextReview = time.Now().AddDate(0, 0, r.IntervalDays).Format("2006-01-02")
	r.LastReviewed = time.Now().UTC().Format(time.RFC3339)
}

// LLMScoreToGrade converts an LLM score (0-100) to an SM-2 grade (0-5).
func LLMScoreToGrade(llmScore int, answerText string) int {
	if answerText == "" || answerText == "(skipped)" || answerText == "(no response)" {
		return 0
	}
	switch {
	case llmScore >= 95:
		return 5
	case llmScore >= 80:
		return 4
	case llmScore >= 60:
		return 3
	case llmScore >= 40:
		return 2
	case llmScore >= 20:
		return 1
	default:
		return 0
	}
}
