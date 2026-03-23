package pipeline

import "testing"

func TestRoute_Flat(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title:       "Raft Leader Election",
			ContentType: "knowledge",
			Tags:        []string{"distributed-systems", "consensus", "raft"},
		},
		ID: "a1b2c3d4",
	}

	dir, filename := Route(note)
	if dir != "notes" {
		t.Errorf("dir = %q, want %q", dir, "notes")
	}
	if filename != "a1b2c3d4.md" {
		t.Errorf("filename = %q, want %q", filename, "a1b2c3d4.md")
	}
}

func TestRoute_AlwaysFlat(t *testing.T) {
	// Even box notes go to notes/ — box is a tag, not a folder.
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title: "Found SQLi",
			Tags:  []string{"pentest", "sqli"},
		},
		ID:  "e5f6a7b8",
		Box: "optimum",
	}

	dir, filename := Route(note)
	if dir != "notes" {
		t.Errorf("dir = %q, want %q", dir, "notes")
	}
	if filename != "e5f6a7b8.md" {
		t.Errorf("filename = %q", filename)
	}
}
