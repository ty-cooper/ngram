package pipeline

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tylercooper/ngram/internal/llm"
	"github.com/tylercooper/ngram/internal/taxonomy"
)

func TestProcess_MockModel(t *testing.T) {
	vaultDir := t.TempDir()

	// Create mock claude script.
	mockScript := filepath.Join(vaultDir, "mock-claude")
	os.WriteFile(mockScript, []byte(`#!/bin/sh
cat <<'FIXTURE'
{"title":"Raft Leader Election","summary":"How Raft elects a leader.","body":"When a follower timeout fires, it becomes a candidate and requests votes.","content_type":"knowledge","domain":"distributed-systems","topic_cluster":"consensus","tags":["raft","consensus"]}
FIXTURE
`), 0o755)

	// Create vault directories.
	os.MkdirAll(filepath.Join(vaultDir, "_inbox"), 0o755)
	os.MkdirAll(filepath.Join(vaultDir, "_meta"), 0o755)

	// Create a taxonomy file.
	os.WriteFile(filepath.Join(vaultDir, "_meta", "taxonomy.yml"), []byte(`
domains:
  distributed-systems:
    aliases: [dist-sys]
tags:
  raft:
    aliases: [raft-consensus]
  consensus:
    aliases: []
`), 0o644)

	// Initialize git repo for commit step.
	runGit(t, vaultDir, "init")
	runGit(t, vaultDir, "config", "user.email", "test@test.com")
	runGit(t, vaultDir, "config", "user.name", "Test")

	// Write a raw note to _inbox/.
	rawNote := `---
captured: "2026-03-23T10:00:00Z"
title: "raft notes"
source: "terminal"
---

Raft uses leader election via randomized timeouts. When a follower's
election timeout fires, it transitions to candidate state.
`
	inboxPath := filepath.Join(vaultDir, "_inbox", "1774234800-raft-notes.md")
	os.WriteFile(inboxPath, []byte(rawNote), 0o644)

	// Load taxonomy.
	tax, err := taxonomy.Load(vaultDir)
	if err != nil {
		t.Fatalf("load taxonomy: %v", err)
	}

	proc := &Processor{
		VaultPath: vaultDir,
		Runner: &llm.Runner{
			BinaryPath: mockScript,
			Model:      "mock",
			VaultPath:  vaultDir,
		},
		Taxonomy:   tax,
		MaxRetries: 0,
	}

	err = proc.Process(context.Background(), inboxPath)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	// Verify _processing/ is empty.
	procEntries, _ := os.ReadDir(filepath.Join(vaultDir, "_processing"))
	for _, e := range procEntries {
		if !strings.HasPrefix(e.Name(), ".") {
			t.Errorf("_processing/ should be empty, found %s", e.Name())
		}
	}

	// Verify _archive/ has the raw note.
	archiveEntries, _ := os.ReadDir(filepath.Join(vaultDir, "_archive"))
	if len(archiveEntries) != 1 {
		t.Errorf("expected 1 file in _archive/, got %d", len(archiveEntries))
	}

	// Verify structured note exists in knowledge/.
	found := false
	filepath.Walk(filepath.Join(vaultDir, "knowledge"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			found = true
			content, _ := os.ReadFile(path)
			s := string(content)
			if !strings.Contains(s, `content_type: "knowledge"`) {
				t.Errorf("missing content_type in structured note")
			}
			if !strings.Contains(s, "retention:") {
				t.Errorf("knowledge note should have retention block")
			}
			if !strings.Contains(s, "state: new") {
				t.Errorf("retention state should be new")
			}
		}
		return nil
	})
	if !found {
		t.Error("no structured note found in knowledge/")
	}

	// Verify API usage log.
	usagePath := filepath.Join(vaultDir, "_meta", "api-usage.jsonl")
	if _, err := os.Stat(usagePath); os.IsNotExist(err) {
		t.Error("api-usage.jsonl not created")
	}
}

func TestProcess_ModelOff(t *testing.T) {
	vaultDir := t.TempDir()
	os.MkdirAll(filepath.Join(vaultDir, "_inbox"), 0o755)

	runGit(t, vaultDir, "init")
	runGit(t, vaultDir, "config", "user.email", "test@test.com")
	runGit(t, vaultDir, "config", "user.name", "Test")

	rawNote := `---
captured: "2026-03-23T10:00:00Z"
title: "raw note"
source: "terminal"
---

Some raw content.
`
	inboxPath := filepath.Join(vaultDir, "_inbox", "1774234800-raw-note.md")
	os.WriteFile(inboxPath, []byte(rawNote), 0o644)

	proc := &Processor{
		VaultPath: vaultDir,
		Runner:    &llm.Runner{Model: "off"},
	}

	err := proc.Process(context.Background(), inboxPath)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	destDir := filepath.Join(vaultDir, "knowledge", "general", "unsorted")
	entries, _ := os.ReadDir(destDir)
	if len(entries) != 1 {
		t.Errorf("expected 1 file in unsorted, got %d", len(entries))
	}
}

func TestStripFrontmatter(t *testing.T) {
	input := "---\ntitle: test\n---\n\nBody content."
	got := stripFrontmatter(input)
	if got != "Body content." {
		t.Errorf("stripFrontmatter = %q", got)
	}
}

func TestStripCodeFences(t *testing.T) {
	input := []byte("```json\n{\"title\":\"test\"}\n```")
	got := stripCodeFences(input)
	if string(got) != `{"title":"test"}` {
		t.Errorf("stripCodeFences = %q", string(got))
	}
}

func TestParseInboxMeta(t *testing.T) {
	content := "---\nsource: \"pipe\"\nbox: \"optimum\"\nphase: \"exploit\"\n---\n\nBody."
	source, box, phase := parseInboxMeta(content)
	if source != "pipe" {
		t.Errorf("source = %q", source)
	}
	if box != "optimum" {
		t.Errorf("box = %q", box)
	}
	if phase != "exploit" {
		t.Errorf("phase = %q", phase)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
}
