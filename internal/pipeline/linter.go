package pipeline

import (
	"fmt"
	"strings"
	"unicode"
)

// Violation describes a single linting failure.
type Violation struct {
	Rule   string
	Detail string
}

func (v Violation) String() string {
	if v.Detail != "" {
		return fmt.Sprintf("%s (%s)", v.Rule, v.Detail)
	}
	return v.Rule
}

var fillerWords = []string{
	"essentially", "basically", "simply", "just", "actually",
	"really", "important", "key", "crucial", "significant",
}

var weakStarters = []string{
	"It is ", "There are ", "There is ", "This is ", "Note that ",
}

// Lint checks a StructuredNote against deterministic quality rules.
func Lint(note *StructuredNote) []Violation {
	var vs []Violation

	// Check em dashes.
	if strings.Contains(note.Body, "\u2014") || strings.Contains(note.Summary, "\u2014") {
		vs = append(vs, Violation{Rule: "no-em-dash"})
	}

	// Check filler words in body.
	bodyWords := extractWords(note.Body)
	for _, filler := range fillerWords {
		if bodyWords[filler] {
			vs = append(vs, Violation{Rule: "no-filler", Detail: filler})
		}
	}

	// Check filler words in summary.
	summaryWords := extractWords(note.Summary)
	for _, filler := range fillerWords {
		if summaryWords[filler] && !bodyWords[filler] {
			vs = append(vs, Violation{Rule: "no-filler", Detail: filler})
		}
	}

	// Check weak starters.
	sentences := extractSentences(note.Body)
	for _, s := range sentences {
		for _, starter := range weakStarters {
			if strings.HasPrefix(s, starter) {
				vs = append(vs, Violation{Rule: "no-weak-starter", Detail: strings.TrimSpace(starter)})
			}
		}
	}

	// Check summary length.
	if len(note.Summary) > 120 {
		vs = append(vs, Violation{Rule: "summary-too-long", Detail: fmt.Sprintf("len: %d", len(note.Summary))})
	}

	return vs
}

// FormatViolations returns a human-readable string of all violations.
func FormatViolations(vs []Violation) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = v.String()
	}
	return "Violations: " + strings.Join(parts, ", ")
}

// extractWords splits text into a set of lowercased words.
func extractWords(text string) map[string]bool {
	words := make(map[string]bool)
	var current strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || r == '-' {
			current.WriteRune(unicode.ToLower(r))
		} else if current.Len() > 0 {
			words[current.String()] = true
			current.Reset()
		}
	}
	if current.Len() > 0 {
		words[current.String()] = true
	}
	return words
}

// extractSentences splits text into sentences (by period + space or newline).
func extractSentences(text string) []string {
	var sentences []string
	// Split on ". " or newlines, then trim.
	for _, part := range strings.Split(text, "\n") {
		for _, s := range strings.Split(part, ". ") {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				sentences = append(sentences, trimmed)
			}
		}
	}
	return sentences
}
