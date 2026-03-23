package llm

import (
	"context"
	"testing"
)

func TestRun_ModelOff(t *testing.T) {
	r := NewRunner("off", t.TempDir())
	_, err := r.Run(context.Background(), "anything")
	if err != ErrModelOff {
		t.Errorf("expected ErrModelOff, got %v", err)
	}
}

func TestRun_NoAPIKey(t *testing.T) {
	// With no ANTHROPIC_API_KEY and cloud model, CheckAuth should fail.
	t.Setenv("ANTHROPIC_API_KEY", "")
	r := NewRunner("cloud", t.TempDir())
	err := r.CheckAuth(context.Background())
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestCheckAuth_ModelOff(t *testing.T) {
	r := NewRunner("off", t.TempDir())
	if err := r.CheckAuth(context.Background()); err != nil {
		t.Errorf("expected nil for MODEL=off, got %v", err)
	}
}
