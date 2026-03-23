package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

const testManifest = `session_id: "2026-03-23T10:42:00Z"
capture_mode: "mixed"
box: "optimum"
phase: "exploit"
items:
  - type: screenshot
    file: ss-001.png
    timestamp: "2026-03-23T10:42:01Z"
  - type: text
    content: "found writable cron job"
    timestamp: "2026-03-23T10:42:05Z"
  - type: screenshot
    file: ss-002.png
    timestamp: "2026-03-23T10:42:08Z"
`

func TestIsBundle(t *testing.T) {
	dir := t.TempDir()

	// Not a bundle (no manifest).
	if IsBundle(dir) {
		t.Error("empty dir should not be a bundle")
	}

	// Create manifest.
	os.WriteFile(filepath.Join(dir, "manifest.yml"), []byte(testManifest), 0o644)
	if !IsBundle(dir) {
		t.Error("dir with manifest.yml should be a bundle")
	}

	// File path is not a bundle.
	if IsBundle(filepath.Join(dir, "manifest.yml")) {
		t.Error("file should not be a bundle")
	}
}

func TestLoadBundle(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "manifest.yml"), []byte(testManifest), 0o644)

	bundle, err := LoadBundle(dir)
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}

	if bundle.Box != "optimum" {
		t.Errorf("Box = %q, want optimum", bundle.Box)
	}
	if len(bundle.Items) != 3 {
		t.Errorf("Items = %d, want 3", len(bundle.Items))
	}
	if bundle.Items[0].Type != "screenshot" {
		t.Errorf("Item 0 type = %q, want screenshot", bundle.Items[0].Type)
	}
	if bundle.Items[1].Content != "found writable cron job" {
		t.Errorf("Item 1 content = %q", bundle.Items[1].Content)
	}
}

func TestBuildBundlePrompt(t *testing.T) {
	bundle := &CaptureBundle{
		Box:   "optimum",
		Phase: "exploit",
		Items: []BundleItem{
			{Type: "screenshot", File: "ss-001.png", Timestamp: "2026-03-23T10:42:01Z"},
			{Type: "text", Content: "found cron job", Timestamp: "2026-03-23T10:42:05Z"},
		},
	}

	prompt := BuildBundlePrompt(bundle, "/tmp")
	if !stringContains(prompt, "optimum") {
		t.Error("prompt missing box context")
	}
	if !stringContains(prompt, "ss-001.png") {
		t.Error("prompt missing screenshot reference")
	}
	if !stringContains(prompt, "found cron job") {
		t.Error("prompt missing text content")
	}
}

func TestBundleTextContent(t *testing.T) {
	bundle := &CaptureBundle{
		Items: []BundleItem{
			{Type: "screenshot", File: "ss-001.png"},
			{Type: "text", Content: "first note"},
			{Type: "screenshot", File: "ss-002.png"},
			{Type: "text", Content: "second note"},
		},
	}

	text := BundleTextContent(bundle)
	if !stringContains(text, "first note") || !stringContains(text, "second note") {
		t.Errorf("BundleTextContent = %q", text)
	}
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
