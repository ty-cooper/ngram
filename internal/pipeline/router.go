package pipeline

import (
	"fmt"
	"path/filepath"

	"github.com/ty-cooper/ngram/internal/capture"
)

// Route determines the destination directory (relative to vault root) and filename.
func Route(note *ProcessedNote) (dir string, filename string) {
	slug := capture.Slugify(note.Title)
	filename = fmt.Sprintf("%s-%s.md", note.ID, slug)

	// Box context takes priority.
	if note.Box != "" {
		phase := note.Phase
		if phase == "" {
			phase = "unsorted"
		}
		dir = filepath.Join("boxes", note.Box, phase)
		return
	}

	// Default: knowledge/{domain}/{topic_cluster}/
	domain := note.Domain
	if domain == "" {
		domain = "general"
	}
	cluster := note.TopicCluster
	if cluster == "" {
		cluster = "uncategorized"
	}
	dir = filepath.Join("knowledge", domain, cluster)
	return
}
