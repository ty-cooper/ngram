package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// CaptureBundle represents a mixed-media capture session from the overlay.
type CaptureBundle struct {
	SessionID   string        `yaml:"session_id"`
	CaptureMode string        `yaml:"capture_mode"`
	Box         string        `yaml:"box"`
	IP          string        `yaml:"ip"`
	Phase       string        `yaml:"phase"`
	Domain      string        `yaml:"domain"`
	Engagement  string        `yaml:"engagement"`
	Items       []BundleItem  `yaml:"items"`
}

// BundleItem is a single item in a capture bundle.
type BundleItem struct {
	Type      string `yaml:"type"`      // "screenshot" or "text"
	File      string `yaml:"file"`      // screenshot filename
	Content   string `yaml:"content"`   // text content
	Timestamp string `yaml:"timestamp"`
}

// IsBundle checks if a path is a capture bundle directory (contains manifest.yml).
func IsBundle(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	_, err = os.Stat(filepath.Join(path, "manifest.yml"))
	return err == nil
}

// LoadBundle reads and parses a capture bundle from a directory.
func LoadBundle(dir string) (*CaptureBundle, error) {
	data, err := os.ReadFile(filepath.Join(dir, "manifest.yml"))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var bundle CaptureBundle
	if err := yaml.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &bundle, nil
}

// BuildBundlePrompt creates the prompt for the LLM to process a capture bundle.
// Screenshots are described by their filenames (Claude vision handles the images).
func BuildBundlePrompt(bundle *CaptureBundle, dir string) string {
	var b strings.Builder

	b.WriteString("You are processing a mixed-media capture session from the Ngram knowledge system.\n")
	b.WriteString("The user captured screenshots and text annotations in sequence.\n")
	b.WriteString("Segment this into one or more atomic structured notes.\n\n")

	if bundle.Box != "" {
		fmt.Fprintf(&b, "Context: box=%s, phase=%s, ip=%s\n", bundle.Box, bundle.Phase, bundle.IP)
	}
	if bundle.Domain != "" {
		fmt.Fprintf(&b, "Domain hint: %s\n", bundle.Domain)
	}

	b.WriteString("\nCAPTURE SESSION ITEMS:\n\n")
	for i, item := range bundle.Items {
		switch item.Type {
		case "screenshot":
			fmt.Fprintf(&b, "[%d] SCREENSHOT: %s (timestamp: %s)\n", i+1, item.File, item.Timestamp)
		case "text":
			fmt.Fprintf(&b, "[%d] TEXT: %s (timestamp: %s)\n", i+1, item.Content, item.Timestamp)
		}
	}

	b.WriteString("\nRules:\n")
	b.WriteString("- Each output note covers ONE concept or finding\n")
	b.WriteString("- Screenshots belong with the text that describes them\n")
	b.WriteString("- Consecutive screenshots without text = one finding\n")
	b.WriteString("- If the session covers one topic, output a single note\n\n")

	b.WriteString("Return ONLY valid JSON matching this schema:\n")
	b.WriteString(`{"notes": [{"title": "...", "summary": "...", "body": "...", "content_type": "knowledge", "domain": "...", "topic_cluster": "...", "tags": ["..."], "screenshots": ["ss-001.png"]}]}`)
	b.WriteString("\n")

	return b.String()
}

// BundleTextContent extracts all text content from a bundle for indexing.
func BundleTextContent(bundle *CaptureBundle) string {
	var parts []string
	for _, item := range bundle.Items {
		if item.Type == "text" {
			parts = append(parts, item.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}
