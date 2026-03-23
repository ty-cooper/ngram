package llm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRun_MockModel(t *testing.T) {
	// Create a mock script that echoes fixture JSON.
	dir := t.TempDir()
	script := filepath.Join(dir, "mock-claude")
	err := os.WriteFile(script, []byte(`#!/bin/sh
cat <<'FIXTURE'
{"title":"Test Note","summary":"A test.","body":"Test body content.","content_type":"knowledge","domain":"testing","topic_cluster":"unit-tests","tags":["test","mock"]}
FIXTURE
`), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	r := &Runner{
		BinaryPath: script,
		Model:      "mock",
		VaultPath:  dir,
	}

	out, err := r.Run(context.Background(), "structure this note")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(out) == 0 {
		t.Fatal("expected non-empty output")
	}

	if got := string(out); got == "" {
		t.Fatal("expected JSON output")
	}
}

func TestRun_ModelOff(t *testing.T) {
	r := &Runner{Model: "off"}
	_, err := r.Run(context.Background(), "anything")
	if err != ErrModelOff {
		t.Errorf("expected ErrModelOff, got %v", err)
	}
}

func TestRun_BinaryNotFound(t *testing.T) {
	r := &Runner{
		BinaryPath: "/nonexistent/binary",
		Model:      "cloud",
		VaultPath:  t.TempDir(),
	}
	_, err := r.Run(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
}
