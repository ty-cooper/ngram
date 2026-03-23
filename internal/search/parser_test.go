package search

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseNoteFile(t *testing.T) {
	dir := t.TempDir()
	vaultPath := dir

	content := `---
title: "Raft Leader Election"
domain: "distributed-systems"
topic_cluster: "consensus"
content_type: "knowledge"
tags:
  - raft
  - consensus
  - leader-election
captured: "2026-03-22T10:42:00Z"
id: "a1b2c3d4"
summary: "How Raft elects a leader when the current leader fails."
---

When a follower's election timeout fires, it transitions to candidate state
and increments its term. It votes for itself and sends RequestVote RPCs to
all other servers.
`

	notePath := filepath.Join(dir, "knowledge", "dist-sys", "a1b2c3d4-raft.md")
	if err := os.MkdirAll(filepath.Dir(notePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(notePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseNoteFile(notePath, vaultPath)
	if err != nil {
		t.Fatalf("ParseNoteFile: %v", err)
	}

	if doc.ID != "a1b2c3d4" {
		t.Errorf("ID = %q, want %q", doc.ID, "a1b2c3d4")
	}
	if doc.Title != "Raft Leader Election" {
		t.Errorf("Title = %q, want %q", doc.Title, "Raft Leader Election")
	}
	if doc.Domain != "distributed-systems" {
		t.Errorf("Domain = %q, want %q", doc.Domain, "distributed-systems")
	}
	if doc.ContentType != "knowledge" {
		t.Errorf("ContentType = %q, want %q", doc.ContentType, "knowledge")
	}
	if len(doc.Tags) != 3 {
		t.Errorf("Tags = %v, want 3 tags", doc.Tags)
	}
	if doc.Summary != "How Raft elects a leader when the current leader fails." {
		t.Errorf("Summary = %q", doc.Summary)
	}
	wantCaptured, _ := time.Parse(time.RFC3339, "2026-03-22T10:42:00Z")
	if doc.Captured != wantCaptured.Unix() {
		t.Errorf("Captured = %d, want %d", doc.Captured, wantCaptured.Unix())
	}
	if doc.FilePath != filepath.Join("knowledge", "dist-sys", "a1b2c3d4-raft.md") {
		t.Errorf("FilePath = %q", doc.FilePath)
	}
	if !contains(doc.Body, "election timeout fires") {
		t.Errorf("Body should contain note content, got %q", doc.Body[:60])
	}
}

func TestParseNoteFile_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	notePath := filepath.Join(dir, "plain.md")
	if err := os.WriteFile(notePath, []byte("Just a plain note.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := ParseNoteFile(notePath, dir)
	if err != nil {
		t.Fatalf("ParseNoteFile: %v", err)
	}

	if doc.ID != "plain" {
		t.Errorf("ID = %q, want %q (filename fallback)", doc.ID, "plain")
	}
}

func TestWalkVault(t *testing.T) {
	dir := t.TempDir()

	// Create indexable files.
	for _, rel := range []string{
		"knowledge/pentest/a1.md",
		"boxes/optimum/recon/b1.md",
		"tools/nmap/c1.md",
	} {
		p := filepath.Join(dir, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("---\ntitle: test\n---\ntest"), 0o644)
	}

	// Create files that should be skipped.
	for _, rel := range []string{
		"_inbox/raw.md",
		"_processing/wip.md",
		"_archive/old.md",
		"_meta/taxonomy.yml",
		".git/config",
	} {
		p := filepath.Join(dir, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("skip"), 0o644)
	}

	files, err := WalkVault(dir)
	if err != nil {
		t.Fatalf("WalkVault: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("got %d files, want 3: %v", len(files), files)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
