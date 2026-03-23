package capture

import (
	"regexp"
	"strings"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a title to a filename-safe slug.
// Lowercase, hyphens for separators, max 50 chars truncated at word boundary.
func Slugify(title string) string {
	s := strings.ToLower(title)
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	if len(s) <= 50 {
		return s
	}

	s = s[:50]
	if i := strings.LastIndex(s, "-"); i > 20 {
		s = s[:i]
	}
	return strings.TrimRight(s, "-")
}

// AutoTitle extracts a title from the first ~8 words of text.
func AutoTitle(text string) string {
	words := strings.Fields(text)
	if len(words) > 8 {
		words = words[:8]
	}
	return strings.Join(words, " ")
}
