package pipeline

import (
	"path/filepath"
	"testing"
)

func TestRoute_Knowledge(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title:        "Raft Leader Election",
			ContentType:  "knowledge",
			Domain:       "distributed-systems",
			TopicCluster: "consensus",
		},
		ID: "a1b2c3d4",
	}

	dir, filename := Route(note)
	wantDir := filepath.Join("knowledge", "distributed-systems", "consensus")
	if dir != wantDir {
		t.Errorf("dir = %q, want %q", dir, wantDir)
	}
	if filename != "a1b2c3d4-raft-leader-election.md" {
		t.Errorf("filename = %q", filename)
	}
}

func TestRoute_Box(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title:       "Found SQLi on Login",
			ContentType: "knowledge",
			Domain:      "pentest",
		},
		ID:    "e5f6a7b8",
		Box:   "optimum",
		Phase: "exploit",
	}

	dir, filename := Route(note)
	wantDir := filepath.Join("boxes", "optimum", "exploit")
	if dir != wantDir {
		t.Errorf("dir = %q, want %q", dir, wantDir)
	}
	if filename != "e5f6a7b8-found-sqli-on-login.md" {
		t.Errorf("filename = %q", filename)
	}
}

func TestRoute_Defaults(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title:       "Random Note",
			ContentType: "reference",
		},
		ID: "11223344",
	}

	dir, _ := Route(note)
	wantDir := filepath.Join("knowledge", "general", "uncategorized")
	if dir != wantDir {
		t.Errorf("dir = %q, want %q", dir, wantDir)
	}
}

func TestRoute_BoxNoPhase(t *testing.T) {
	note := &ProcessedNote{
		StructuredNote: StructuredNote{
			Title: "Box Note",
		},
		ID:  "aabbccdd",
		Box: "target",
	}

	dir, _ := Route(note)
	wantDir := filepath.Join("boxes", "target", "unsorted")
	if dir != wantDir {
		t.Errorf("dir = %q, want %q", dir, wantDir)
	}
}
