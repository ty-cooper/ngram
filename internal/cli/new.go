package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/capture"
	"github.com/ty-cooper/ngram/internal/config"
	"github.com/ty-cooper/ngram/internal/vault"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Open $EDITOR for multi-line note capture",
	RunE:  newRun,
}

func newRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	// Create temp file.
	tmp, err := os.CreateTemp("", "ngram-*.md")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	// Open $EDITOR.
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	editorCmd := exec.Command(editor, tmpPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor: %w", err)
	}

	// Read result.
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("read temp: %w", err)
	}

	body := strings.TrimSpace(string(content))
	if body == "" {
		fmt.Println("Empty note, nothing captured.")
		return nil
	}

	// Detect title from first line.
	title := detectTitle(body)
	slug := "note"
	if title != "" {
		slug = capture.Slugify(title)
	}

	// Build frontmatter.
	ts := time.Now().UTC()
	boxrc, _ := config.FindBoxRC("")

	var fm strings.Builder
	fm.WriteString("---\n")
	fmt.Fprintf(&fm, "captured: %q\n", ts.Format(time.RFC3339))
	if title != "" {
		fmt.Fprintf(&fm, "title: %q\n", title)
	}
	fm.WriteString("source: \"cli-editor\"\n")
	fm.WriteString("capture_mode: \"editor\"\n")
	if boxrc.Box != "" {
		fmt.Fprintf(&fm, "box: %q\n", boxrc.Box)
	}
	if boxrc.Phase != "" {
		fmt.Fprintf(&fm, "phase: %q\n", boxrc.Phase)
	}
	if boxrc.IP != "" {
		fmt.Fprintf(&fm, "ip: %q\n", boxrc.IP)
	}
	fm.WriteString("---\n\n")

	// Write to _inbox/.
	inboxDir := filepath.Join(c.VaultPath, "_inbox")
	vault.EnsureDir(inboxDir)

	filename := fmt.Sprintf("%d-%s.md", ts.Unix(), slug)
	destPath := filepath.Join(inboxDir, filename)

	if err := os.WriteFile(destPath, []byte(fm.String()+body+"\n"), 0o600); err != nil {
		return fmt.Errorf("write note: %w", err)
	}

	relPath := filepath.Join("_inbox", filename)
	context := ""
	if boxrc.Box != "" {
		context = fmt.Sprintf(" [%s/%s]", boxrc.Box, boxrc.Phase)
	}
	fmt.Printf("✓ %s%s\n", relPath, context)
	return nil
}

// detectTitle extracts a title from the first line if it looks like a heading
// or a short line followed by a blank line.
func detectTitle(body string) string {
	lines := strings.SplitN(body, "\n", 3)
	first := strings.TrimSpace(lines[0])

	// Markdown heading.
	if strings.HasPrefix(first, "# ") {
		return strings.TrimPrefix(first, "# ")
	}

	// Short first line followed by blank line.
	if len(first) > 0 && len(first) <= 80 && len(lines) >= 2 && strings.TrimSpace(lines[1]) == "" {
		return first
	}

	return ""
}
