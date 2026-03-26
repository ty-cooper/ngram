package pipeline

import (
	"fmt"
	"log"

	"github.com/ty-cooper/ngram/internal/search"
)

// RecallResult represents a related note from another engagement.
type RecallResult struct {
	NoteID     string
	Title      string
	Box        string
	Engagement string
	Summary    string
	FilePath   string
	Score      float64
}

// recallPass searches for related knowledge from other engagements.
// Returns nil if search is unavailable or no box context.
func (p *Processor) recallPass(note *ProcessedNote) []RecallResult {
	if p.SearchClient == nil || note.Box == "" {
		return nil
	}

	query := note.Title + " " + note.Summary
	filter := fmt.Sprintf(`box != "%s"`, note.Box)

	results, err := p.SearchClient.FindSimilarFiltered(query, 5, filter)
	if err != nil {
		log.Printf("warn: recall search failed: %v", err)
		return nil
	}

	var recalls []RecallResult
	for _, r := range results {
		if r.Score >= 0.6 {
			recalls = append(recalls, RecallResult{
				NoteID:     r.ID,
				Title:      r.Title,
				Box:        r.Box,
				Engagement: r.Engagement,
				Summary:    r.Summary,
				FilePath:   r.FilePath,
				Score:      r.Score,
			})
		}
	}
	return recalls
}

// addRecallLinks populates the Related field on a ProcessedNote instead of appending text.
func addRecallLinks(note *ProcessedNote, recalls []RecallResult) {
	for _, r := range recalls {
		note.Related = append(note.Related, RelatedLink{
			ID:    r.NoteID,
			Title: r.Title,
		})
	}
}

// RecallSearch performs a cross-engagement search for the CLI command.
func RecallSearch(client *search.Client, query string, excludeBox string, limit int64) ([]RecallResult, error) {
	filter := ""
	if excludeBox != "" {
		filter = fmt.Sprintf(`box != "%s"`, excludeBox)
	}

	results, err := client.FindSimilarFiltered(query, limit, filter)
	if err != nil {
		return nil, err
	}

	var recalls []RecallResult
	for _, r := range results {
		recalls = append(recalls, RecallResult{
			NoteID:     r.ID,
			Title:      r.Title,
			Box:        r.Box,
			Engagement: r.Engagement,
			Summary:    r.Summary,
			FilePath:   r.FilePath,
			Score:      r.Score,
		})
	}
	return recalls, nil
}
