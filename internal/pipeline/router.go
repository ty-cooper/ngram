package pipeline

import "fmt"

// Route determines the destination directory and filename.
// Zettelkasten: flat structure, all notes in notes/, ID as filename.
// Organization via tags and [[links]], not folders.
func Route(note *ProcessedNote) (dir string, filename string) {
	filename = fmt.Sprintf("%s.md", note.ID)
	dir = "notes"
	return
}
