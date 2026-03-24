package pipeline

import (
	"fmt"
	"path/filepath"

	"github.com/ty-cooper/ngram/internal/capture"
)

// Route determines the destination directory and filename.
// Flat storage: all non-box notes go to knowledge/ (one directory).
// Box notes go to boxes/{box}/{phase}/.
// Organization is via tags and frontmatter, not folders.
func Route(note *ProcessedNote) (dir string, filename string) {
	slug := capture.Slugify(note.Title)
	filename = fmt.Sprintf("%s-%s.md", note.ID, slug)

	// Box notes: boxes/{box}/{phase}/
	if note.Box != "" {
		phase := note.Phase
		if phase == "" {
			phase = "unsorted"
		}
		dir = filepath.Join("boxes", note.Box, phase)
		return
	}

	// Everything else: flat in knowledge/
	dir = "knowledge"
	return
}
