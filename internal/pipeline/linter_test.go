package pipeline

import (
	"testing"
)

func TestLint_Clean(t *testing.T) {
	note := &StructuredNote{
		Title:   "Clean Note",
		Summary: "A clean note with no violations.",
		Body:    "Raft uses leader election via randomized timeouts. Candidates request votes from peers.",
	}
	vs := Lint(note)
	if len(vs) != 0 {
		t.Errorf("expected no violations, got %v", vs)
	}
}

func TestLint_EmDash(t *testing.T) {
	note := &StructuredNote{
		Title:   "Em Dash",
		Summary: "Fine summary.",
		Body:    "Raft \u2014 a consensus algorithm.",
	}
	vs := Lint(note)
	if !hasRule(vs, "no-em-dash") {
		t.Error("expected no-em-dash violation")
	}
}

func TestLint_FillerWords(t *testing.T) {
	for _, word := range []string{"essentially", "basically", "simply", "just", "actually", "really", "important", "key", "crucial", "significant"} {
		note := &StructuredNote{
			Title:   "Filler",
			Summary: "Fine.",
			Body:    "This is " + word + " a test note.",
		}
		vs := Lint(note)
		if !hasRule(vs, "no-filler") {
			t.Errorf("expected no-filler violation for %q", word)
		}
	}
}

func TestLint_WeakStarter(t *testing.T) {
	tests := []string{
		"It is important to note.",
		"There are many approaches.",
		"There is a way.",
		"This is a method.",
		"Note that this works.",
	}
	for _, body := range tests {
		note := &StructuredNote{
			Title:   "Weak",
			Summary: "Fine.",
			Body:    body,
		}
		vs := Lint(note)
		if !hasRule(vs, "no-weak-starter") {
			t.Errorf("expected no-weak-starter for %q", body)
		}
	}
}

func TestLint_SummaryTooLong(t *testing.T) {
	long := ""
	for i := 0; i < 130; i++ {
		long += "x"
	}
	note := &StructuredNote{
		Title:   "Long Summary",
		Summary: long,
		Body:    "Content.",
	}
	vs := Lint(note)
	if !hasRule(vs, "summary-too-long") {
		t.Error("expected summary-too-long violation")
	}
}

func TestFormatViolations(t *testing.T) {
	vs := []Violation{
		{Rule: "no-filler", Detail: "basically"},
		{Rule: "summary-too-long", Detail: "len: 145"},
	}
	got := FormatViolations(vs)
	if got != "Violations: no-filler (basically), summary-too-long (len: 145)" {
		t.Errorf("FormatViolations = %q", got)
	}
}

func hasRule(vs []Violation, rule string) bool {
	for _, v := range vs {
		if v.Rule == rule {
			return true
		}
	}
	return false
}
