package dream

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ty-cooper/ngram/internal/taxonomy"
)

// LintViolation describes a single lint issue found in a note.
type LintViolation struct {
	Rule    string // e.g. "missing-context-comment", "over-word-limit"
	Message string
}

// LintResult holds all violations for a single note.
type LintResult struct {
	Note       noteEntry
	Violations []LintViolation
}

var codeBlockRe = regexp.MustCompile("(?m)^```[a-z]*\\s*\n")
var contextCommentRe = regexp.MustCompile("(?m)^#\\s*\\[.+\\]")

// lintPass runs local pattern checks on all notes. No LLM calls.
func (r *Runner) lintPass(notes []noteEntry) []LintResult {
	tax, _ := taxonomy.Load(r.VaultPath)

	var results []LintResult
	for _, note := range notes {
		var violations []LintViolation

		// 1. Code blocks without # [Context] comment.
		violations = append(violations, checkCodeBlockContext(note.Body)...)

		// 2. Notes over 500 words (atomicity violation).
		if wc := wordCount(note.Body); wc > 500 {
			violations = append(violations, LintViolation{
				Rule:    "over-word-limit",
				Message: fmt.Sprintf("note is %d words, exceeds 500-word atomic limit", wc),
			})
		}

		// 3. Domain used as tag.
		if note.Domain != "" {
			for _, t := range note.Tags {
				if strings.EqualFold(t, note.Domain) {
					violations = append(violations, LintViolation{
						Rule:    "domain-as-tag",
						Message: fmt.Sprintf("domain %q should not also be a tag", note.Domain),
					})
				}
			}
		}

		// 4. System tags that shouldn't be on knowledge notes.
		blacklist := map[string]bool{"inbox": true, "test": true, "log": true, "capture-session": true}
		for _, t := range note.Tags {
			if blacklist[t] {
				violations = append(violations, LintViolation{
					Rule:    "system-tag",
					Message: fmt.Sprintf("system tag %q should not appear on knowledge notes", t),
				})
			}
		}

		// 5. Tags not in taxonomy.
		if tax != nil {
			for _, t := range note.Tags {
				if _, ok := tax.Tags[t]; !ok {
					// Check aliases too.
					resolved := tax.ResolveTag(t)
					if resolved == t {
						// Not found in taxonomy at all — could be new.
						// Only flag if it looks like a near-duplicate of an existing tag.
						for canonical := range tax.Tags {
							if levenshtein(t, canonical) <= 2 && t != canonical {
								violations = append(violations, LintViolation{
									Rule:    "near-duplicate-tag",
									Message: fmt.Sprintf("tag %q is near-duplicate of canonical %q", t, canonical),
								})
							}
						}
					}
				}
			}
		}

		// 6. Missing ## Related section.
		if !strings.Contains(note.Body, "## Related") {
			violations = append(violations, LintViolation{
				Rule:    "missing-related-section",
				Message: "note has no ## Related section for inter-note links",
			})
		}

		// 7. Missing footer metadata.
		if note.ID == "" {
			violations = append(violations, LintViolation{
				Rule:    "missing-id",
				Message: "note has no id",
			})
		}
		if note.Domain == "" {
			violations = append(violations, LintViolation{
				Rule:    "missing-domain",
				Message: "note has no domain set",
			})
		}
		if len(note.Tags) == 0 {
			violations = append(violations, LintViolation{
				Rule:    "missing-tags",
				Message: "note has no tags",
			})
		}

		// 8. More than 5 tags.
		if len(note.Tags) > 5 {
			violations = append(violations, LintViolation{
				Rule:    "too-many-tags",
				Message: fmt.Sprintf("note has %d tags, max is 5", len(note.Tags)),
			})
		}

		if len(violations) > 0 {
			results = append(results, LintResult{Note: note, Violations: violations})
		}
	}
	return results
}

// checkCodeBlockContext finds code blocks missing a # [Tool/Context] comment.
func checkCodeBlockContext(body string) []LintViolation {
	var violations []LintViolation

	blocks := codeBlockRe.FindAllStringIndex(body, -1)
	for _, loc := range blocks {
		// Get the content after the opening ```.
		rest := body[loc[1]:]
		endIdx := strings.Index(rest, "```")
		if endIdx < 0 {
			continue
		}
		blockContent := rest[:endIdx]

		// Check if the first non-empty line starts with # [
		lines := strings.Split(strings.TrimSpace(blockContent), "\n")
		if len(lines) == 0 {
			continue
		}
		firstLine := strings.TrimSpace(lines[0])
		if !contextCommentRe.MatchString(firstLine) {
			// Extract language hint for the message.
			opener := body[loc[0]:loc[1]]
			lang := strings.TrimSpace(strings.TrimPrefix(opener, "```"))
			if lang == "" {
				lang = "unknown"
			}
			preview := firstLine
			if len(preview) > 40 {
				preview = preview[:40] + "..."
			}
			violations = append(violations, LintViolation{
				Rule:    "missing-context-comment",
				Message: fmt.Sprintf("code block (%s) missing # [Tool] context comment: %q", lang, preview),
			})
		}
	}
	return violations
}

func wordCount(s string) int {
	return len(strings.Fields(s))
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la := utf8.RuneCountInString(a)
	lb := utf8.RuneCountInString(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	ra := []rune(a)
	rb := []rune(b)

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = minOf(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func minOf(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
