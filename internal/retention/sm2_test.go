package retention

import (
	"testing"
	"time"
)

func TestNewRetention(t *testing.T) {
	r := NewRetention(time.Now())
	if r.State != "new" {
		t.Errorf("State = %q, want new", r.State)
	}
	if r.EaseFactor != 2.5 {
		t.Errorf("EaseFactor = %f, want 2.5", r.EaseFactor)
	}
}

func TestUpdate_NewToLearning(t *testing.T) {
	r := NewRetention(time.Now())
	r.Update(4) // pass
	if r.State != "learning" {
		t.Errorf("State = %q, want learning", r.State)
	}
	if r.IntervalDays != 1 {
		t.Errorf("IntervalDays = %d, want 1", r.IntervalDays)
	}
	if r.RepetitionCount != 1 {
		t.Errorf("RepetitionCount = %d, want 1", r.RepetitionCount)
	}
}

func TestUpdate_LearningProgression(t *testing.T) {
	r := NewRetention(time.Now())
	r.Update(4) // new → learning, interval 1
	r.Update(4) // learning rep 1 → interval 3
	if r.IntervalDays != 3 {
		t.Errorf("IntervalDays = %d, want 3", r.IntervalDays)
	}
	r.Update(4) // learning rep 2 → reviewing, interval 7
	if r.State != "reviewing" {
		t.Errorf("State = %q, want reviewing", r.State)
	}
	if r.IntervalDays != 7 {
		t.Errorf("IntervalDays = %d, want 7", r.IntervalDays)
	}
}

func TestUpdate_ReviewingGrowth(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 7, RepetitionCount: 3}
	r.Update(5) // perfect recall
	if r.IntervalDays != 18 { // round(7 * 2.5) = 18
		t.Errorf("IntervalDays = %d, want 18", r.IntervalDays)
	}
}

func TestUpdate_IntervalCap90(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 80, RepetitionCount: 10}
	r.Update(5)
	if r.IntervalDays > 90 {
		t.Errorf("IntervalDays = %d, should be capped at 90", r.IntervalDays)
	}
}

func TestUpdate_SolidifiedTransition(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 30, RepetitionCount: 5, Streak: 4}
	r.Update(4) // streak becomes 5, interval >= 30
	if r.State != "solidified" {
		t.Errorf("State = %q, want solidified", r.State)
	}
}

func TestUpdate_SolidifiedStays(t *testing.T) {
	r := &Retention{State: "solidified", EaseFactor: 2.5, IntervalDays: 60, RepetitionCount: 10, Streak: 8}
	r.Update(5)
	if r.State != "solidified" {
		t.Errorf("State = %q, want solidified", r.State)
	}
	if r.IntervalDays > 90 {
		t.Errorf("IntervalDays = %d, should cap at 90", r.IntervalDays)
	}
}

func TestUpdate_FailFromReviewing(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 18, RepetitionCount: 5, Streak: 3}
	r.Update(1) // fail
	if r.State != "learning" {
		t.Errorf("State = %q, want learning", r.State)
	}
	if r.IntervalDays != 1 {
		t.Errorf("IntervalDays = %d, want 1", r.IntervalDays)
	}
	if r.Streak != 0 {
		t.Errorf("Streak = %d, want 0", r.Streak)
	}
	if r.LapseCount != 1 {
		t.Errorf("LapseCount = %d, want 1", r.LapseCount)
	}
	if r.RepetitionCount != 0 {
		t.Errorf("RepetitionCount = %d, want 0", r.RepetitionCount)
	}
}

func TestUpdate_FailFromSolidified(t *testing.T) {
	r := &Retention{State: "solidified", EaseFactor: 2.5, IntervalDays: 90, RepetitionCount: 15, Streak: 10}
	r.Update(0) // complete blackout
	if r.State != "learning" {
		t.Errorf("State = %q, want learning", r.State)
	}
	if r.IntervalDays != 1 {
		t.Errorf("IntervalDays = %d, want 1", r.IntervalDays)
	}
}

func TestUpdate_FailFromLearning(t *testing.T) {
	r := &Retention{State: "learning", EaseFactor: 2.5, IntervalDays: 3, RepetitionCount: 1}
	r.Update(2) // fail stays in learning
	if r.State != "learning" {
		t.Errorf("State = %q, want learning", r.State)
	}
	if r.IntervalDays != 1 {
		t.Errorf("IntervalDays = %d, want 1", r.IntervalDays)
	}
}

func TestUpdate_EaseFactorFloor(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 1.3, IntervalDays: 7, RepetitionCount: 3}
	r.Update(0) // fail, ease drops
	if r.EaseFactor < 1.3 {
		t.Errorf("EaseFactor = %f, should not go below 1.3", r.EaseFactor)
	}
}

func TestUpdate_EaseIncreases(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 7, RepetitionCount: 3}
	before := r.EaseFactor
	r.Update(5) // perfect
	if r.EaseFactor <= before {
		t.Errorf("EaseFactor should increase on grade 5, was %f now %f", before, r.EaseFactor)
	}
}

func TestUpdate_EaseDecreases(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 7, RepetitionCount: 3}
	before := r.EaseFactor
	r.Update(3) // correct with difficulty
	if r.EaseFactor >= before {
		t.Errorf("EaseFactor should decrease on grade 3, was %f now %f", before, r.EaseFactor)
	}
}

func TestUpdate_RetentionScore(t *testing.T) {
	r := NewRetention(time.Now())
	r.Update(5) // 1/1 = 100%
	if r.RetentionScore != 100 {
		t.Errorf("RetentionScore = %d, want 100", r.RetentionScore)
	}
	r.Update(0) // 1/2 = 50%
	if r.RetentionScore != 50 {
		t.Errorf("RetentionScore = %d, want 50", r.RetentionScore)
	}
}

func TestUpdate_NextReviewSet(t *testing.T) {
	r := NewRetention(time.Now())
	r.Update(4)
	if r.NextReview == "" {
		t.Error("NextReview should be set after update")
	}
	if r.LastReviewed == "" {
		t.Error("LastReviewed should be set after update")
	}
}

func TestUpdate_StreakTracking(t *testing.T) {
	r := NewRetention(time.Now())
	r.Update(4)
	r.Update(4)
	r.Update(4)
	if r.Streak != 3 {
		t.Errorf("Streak = %d, want 3", r.Streak)
	}
	r.Update(1) // fail resets
	if r.Streak != 0 {
		t.Errorf("Streak = %d, want 0 after fail", r.Streak)
	}
}

func TestUpdate_LapseCountAccumulates(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 7, RepetitionCount: 3}
	r.Update(1)
	r.Update(4)
	r.Update(4)
	r.Update(4) // back to reviewing
	r.Update(0) // lapse again
	if r.LapseCount != 2 {
		t.Errorf("LapseCount = %d, want 2", r.LapseCount)
	}
}

func TestUpdate_Grade0Blackout(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 30, RepetitionCount: 8, Streak: 5}
	r.Update(0)
	if r.State != "learning" {
		t.Errorf("State = %q, want learning", r.State)
	}
	if r.IntervalDays != 1 {
		t.Errorf("IntervalDays = %d, want 1", r.IntervalDays)
	}
}

func TestUpdate_Grade3Boundary(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 7, RepetitionCount: 3}
	r.Update(3) // pass, but barely
	if r.TotalCorrect != 1 {
		t.Errorf("TotalCorrect = %d, want 1 (grade 3 is a pass)", r.TotalCorrect)
	}
}

func TestUpdate_Grade2Fail(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 7, RepetitionCount: 3}
	r.Update(2) // fail
	if r.TotalCorrect != 0 {
		t.Errorf("TotalCorrect = %d, want 0 (grade 2 is a fail)", r.TotalCorrect)
	}
}

func TestUpdate_FullSuccessSequence(t *testing.T) {
	r := NewRetention(time.Now())
	// Day 0: new
	r.Update(4) // → learning, interval 1
	assertState(t, r, "learning", 1)
	r.Update(4) // → learning, interval 3
	assertState(t, r, "learning", 3)
	r.Update(4) // → reviewing, interval 7
	assertState(t, r, "reviewing", 7)
	r.Update(5) // reviewing, interval grows
	if r.IntervalDays <= 7 {
		t.Errorf("interval should grow past 7, got %d", r.IntervalDays)
	}
}

func TestUpdate_FailAndRecover(t *testing.T) {
	r := &Retention{State: "reviewing", EaseFactor: 2.5, IntervalDays: 18, RepetitionCount: 5, Streak: 3}
	r.Update(1) // fail → learning, interval 1
	assertState(t, r, "learning", 1)
	r.Update(4) // learning rep 0 → interval 1
	r.Update(4) // learning rep 1 → interval 3
	assertState(t, r, "learning", 3)
	r.Update(4) // learning rep 2 → reviewing, interval 7
	assertState(t, r, "reviewing", 7)
}

func TestLLMScoreToGrade(t *testing.T) {
	tests := []struct {
		score  int
		answer string
		want   int
	}{
		{95, "perfect answer", 5},
		{80, "good answer", 4},
		{60, "okay answer", 3},
		{40, "partial answer", 2},
		{20, "bad answer", 1},
		{10, "terrible", 0},
		{100, "", 0},
		{100, "(skipped)", 0},
		{100, "(no response)", 0},
	}

	for _, tt := range tests {
		got := LLMScoreToGrade(tt.score, tt.answer)
		if got != tt.want {
			t.Errorf("LLMScoreToGrade(%d, %q) = %d, want %d", tt.score, tt.answer, got, tt.want)
		}
	}
}

func assertState(t *testing.T, r *Retention, state string, interval int) {
	t.Helper()
	if r.State != state {
		t.Errorf("State = %q, want %q", r.State, state)
	}
	if r.IntervalDays != interval {
		t.Errorf("IntervalDays = %d, want %d", r.IntervalDays, interval)
	}
}
