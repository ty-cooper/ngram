package capture

import (
	"fmt"
	"strings"
	"time"

	"github.com/tylercooper/ngram/internal/config"
)

type NoteMetadata struct {
	Title   string
	Source  string // "terminal", "pipe", "command-capture"
	Command string // only for command-capture
	BoxCtx  *config.BoxContext
	Time    time.Time
}

// BuildFrontmatter generates YAML frontmatter for an inbox note.
func BuildFrontmatter(m NoteMetadata) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "captured: %q\n", m.Time.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "title: %q\n", m.Title)
	fmt.Fprintf(&b, "source: %q\n", m.Source)

	if m.Command != "" {
		fmt.Fprintf(&b, "command: %q\n", m.Command)
	}

	if m.BoxCtx != nil {
		if m.BoxCtx.Box != "" {
			fmt.Fprintf(&b, "box: %q\n", m.BoxCtx.Box)
		}
		if m.BoxCtx.IP != "" {
			fmt.Fprintf(&b, "ip: %q\n", m.BoxCtx.IP)
		}
		if m.BoxCtx.Phase != "" {
			fmt.Fprintf(&b, "phase: %q\n", m.BoxCtx.Phase)
		}
		if m.BoxCtx.Engagement != "" {
			fmt.Fprintf(&b, "engagement: %q\n", m.BoxCtx.Engagement)
		}
	}

	b.WriteString("---\n")
	return b.String()
}
