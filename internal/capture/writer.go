package capture

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tylercooper/ngram/internal/config"
)

type WriteResult struct {
	RelPath string // path relative to vault root (e.g. "_inbox/1711043200-sqli-on-login.md")
	AbsPath string
}

// WriteNote atomically writes a capture note to _inbox/.
func WriteNote(vaultPath string, body string, meta NoteMetadata) (*WriteResult, error) {
	inboxDir := filepath.Join(vaultPath, "_inbox")
	if err := os.MkdirAll(inboxDir, 0o755); err != nil {
		return nil, fmt.Errorf("create inbox: %w", err)
	}

	if meta.Time.IsZero() {
		meta.Time = time.Now()
	}

	slug := Slugify(meta.Title)
	filename := fmt.Sprintf("%d-%s.md", meta.Time.Unix(), slug)
	finalPath := filepath.Join(inboxDir, filename)

	content := BuildFrontmatter(meta) + "\n" + body + "\n"

	tmp, err := os.CreateTemp(inboxDir, ".tmp-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, finalPath); err != nil {
		os.Remove(tmpName)
		return nil, fmt.Errorf("rename to final: %w", err)
	}

	relPath := filepath.Join("_inbox", filename)
	return &WriteResult{RelPath: relPath, AbsPath: finalPath}, nil
}

// Confirmation formats the confirmation line printed after capture.
func Confirmation(relPath string, boxCtx *config.BoxContext) string {
	ctx := ""
	if boxCtx != nil && boxCtx.Box != "" {
		ctx = boxCtx.Box
		if boxCtx.Phase != "" {
			ctx += "/" + boxCtx.Phase
		}
		ctx = " [" + ctx + "]"
	}
	return fmt.Sprintf("✓ %s%s", relPath, ctx)
}
