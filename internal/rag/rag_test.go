package rag

import (
	"testing"
)

func TestBuildSynthesisPrompt(t *testing.T) {
	notes := []noteContent{
		{id: "abc123", title: "Raft Consensus", body: "Raft is a consensus algorithm."},
		{id: "def456", title: "Paxos Overview", body: "Paxos is another consensus algorithm."},
	}

	prompt := buildSynthesisPrompt("how does consensus work?", notes)

	if !containsStr(prompt, "how does consensus work?") {
		t.Error("prompt missing question")
	}
	if !containsStr(prompt, "[abc123]") {
		t.Error("prompt missing note ID abc123")
	}
	if !containsStr(prompt, "[def456]") {
		t.Error("prompt missing note ID def456")
	}
	if !containsStr(prompt, "citation") {
		t.Error("prompt missing citation instruction")
	}
}

func TestExtractID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"knowledge/pentest/scanning/a1b2c3d4-nmap-scan.md", "a1b2c3d4"},
		{"boxes/optimum/exploit/e5f6a7b8-sqli.md", "e5f6a7b8"},
		{"simple.md", "simple"},
	}
	for _, tt := range tests {
		if got := extractID(tt.input); got != tt.want {
			t.Errorf("extractID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello world", 5); got != "hello..." {
		t.Errorf("truncate = %q", got)
	}
	if got := truncate("hi", 10); got != "hi" {
		t.Errorf("truncate = %q", got)
	}
}

func TestStripCodeFences(t *testing.T) {
	input := []byte("```json\n[\"query1\", \"query2\"]\n```")
	got := stripCodeFences(input)
	if string(got) != `["query1", "query2"]` {
		t.Errorf("stripCodeFences = %q", string(got))
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && contains(s, sub)
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
