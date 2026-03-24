package pipeline

import "testing"

func TestRoute_Knowledge(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title:        "Raft Leader Election",
			ContentType:  "knowledge",
			Domain:       "distributed-systems",
			TopicCluster: "consensus",
			Tags:         []string{"raft", "consensus"},
		},
		ID: "a1b2c3d4",
	}

	dir, filename := Route(note)
	if dir != "knowledge" {
		t.Errorf("dir = %q, want %q", dir, "knowledge")
	}
	if filename != "a1b2c3d4-raft-leader-election.md" {
		t.Errorf("filename = %q", filename)
	}
}

func TestRoute_Box(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title: "Found SQLi",
			Tags:  []string{"pentest", "sqli"},
		},
		ID:    "e5f6a7b8",
		Box:   "optimum",
		Phase: "exploit",
	}

	dir, _ := Route(note)
	want := "boxes/optimum/exploit"
	if dir != want {
		t.Errorf("dir = %q, want %q", dir, want)
	}
}

func TestRoute_NoDomain(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title: "Random thought",
		},
		ID: "c1d2e3f4",
	}

	dir, _ := Route(note)
	if dir != "knowledge" {
		t.Errorf("dir = %q, want %q", dir, "knowledge")
	}
}
