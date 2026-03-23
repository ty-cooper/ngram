package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ty-cooper/ngram/internal/llm"
	"github.com/ty-cooper/ngram/internal/notify"
	"github.com/ty-cooper/ngram/internal/search"
	"github.com/ty-cooper/ngram/internal/taxonomy"
	"github.com/ty-cooper/ngram/internal/vault"
)

// Processor runs the structuring pipeline for notes in _inbox/.
type Processor struct {
	VaultPath    string
	Runner       *llm.Runner
	Taxonomy     *taxonomy.Taxonomy
	SearchClient *search.Client // nil if Meilisearch unavailable
	Dedup        *Deduplicator  // nil to skip dedup
	MaxRetries   int            // default 2
}

// Process runs the full pipeline for a single file in _inbox/.
func (p *Processor) Process(ctx context.Context, inboxPath string) error {
	start := time.Now()

	// 1. Move to _processing/.
	procPath, err := p.moveToProcessing(inboxPath)
	if err != nil {
		return fmt.Errorf("move to processing: %w", err)
	}

	// 2. Read raw content.
	raw, err := os.ReadFile(procPath)
	if err != nil {
		return fmt.Errorf("read raw: %w", err)
	}
	rawContent := string(raw)

	// Parse inbox frontmatter to get source metadata.
	source, box, phase := parseInboxMeta(rawContent)
	body := stripFrontmatter(rawContent)

	// 3. If MODEL=off, write raw directly.
	if p.Runner.Model == "off" {
		return p.writeRawDirect(rawContent, procPath)
	}

	// 4-6. Structure with retry loop.
	maxRetries := p.MaxRetries
	if maxRetries == 0 {
		maxRetries = 2
	}

	structured, err := p.structureWithRetry(ctx, body, maxRetries)
	if err != nil {
		// On total failure, leave in _processing/ for manual review.
		return fmt.Errorf("structure: %w", err)
	}

	// 7. Resolve taxonomy.
	if p.Taxonomy != nil {
		structured.Domain = p.Taxonomy.ResolveDomain(structured.Domain)
		structured.Tags = p.Taxonomy.ResolveTags(structured.Tags)
	}

	// 8. Dedup check.
	if p.Dedup != nil {
		result := p.Dedup.Check(ctx, structured, procPath)
		switch result.Action {
		case "duplicate":
			log.Printf("ngram: dedup — duplicate of %s", result.TargetID)
			return p.archiveRaw(procPath)
		case "appended":
			log.Printf("ngram: dedup — appended to %s", result.TargetID)
			p.gitCommit(result.TargetID, result.TargetID, "append")
			return p.archiveRaw(procPath)
		}
		// "proceed" — continue with normal pipeline.
	}

	// 9. Create ProcessedNote.
	processed := &ProcessedNote{
		StructuredNote: *structured,
		ID:             GenerateID(),
		Source:         source,
		Box:            box,
		Phase:          phase,
		Created:        time.Now().UTC(),
	}

	// 9. Route and write.
	dir, filename := Route(processed)
	destDir := filepath.Join(p.VaultPath, dir)
	if err := vault.EnsureDir(destDir); err != nil {
		return fmt.Errorf("ensure dir %s: %w", dir, err)
	}

	destPath := filepath.Join(destDir, filename)
	content := BuildNoteContent(processed)

	if err := atomicWrite(destPath, content); err != nil {
		return fmt.Errorf("write note: %w", err)
	}

	relPath := filepath.Join(dir, filename)
	log.Printf("ngram: structured %s → %s", processed.ID, relPath)

	// 10. Index in Meilisearch.
	if p.SearchClient != nil {
		doc := search.NoteDocument{
			ID:          processed.ID,
			Title:       processed.Title,
			Body:        processed.Body,
			Summary:     processed.Summary,
			Tags:        processed.Tags,
			Domain:      processed.Domain,
			ContentType: processed.ContentType,
			Box:         processed.Box,
			Phase:       processed.Phase,
			FilePath:    relPath,
			Captured:    processed.Created.Unix(),
		}
		if err := p.SearchClient.IndexNote(doc); err != nil {
			log.Printf("warn: index failed for %s: %v", processed.ID, err)
		}
	}

	// 11. Desktop notification.
	notify.Send("Ngram", fmt.Sprintf("Structured: %s → %s", processed.Title, relPath))

	// 12. Git commit.
	p.gitCommit(relPath, processed.ID, source)

	// 12. Archive raw.
	if err := p.archiveRaw(procPath); err != nil {
		log.Printf("warn: archive failed: %v", err)
	}

	// 13. Log usage.
	duration := time.Since(start).Milliseconds()
	p.logUsage(processed.ID, "processor", duration)

	return nil
}

func (p *Processor) moveToProcessing(path string) (string, error) {
	procDir := filepath.Join(p.VaultPath, "_processing")
	if err := vault.EnsureDir(procDir); err != nil {
		return "", err
	}
	dest := filepath.Join(procDir, filepath.Base(path))
	if err := os.Rename(path, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func (p *Processor) structureWithRetry(ctx context.Context, rawBody string, maxRetries int) (*StructuredNote, error) {
	prompt := BuildStructuringPrompt(p.Taxonomy, rawBody)

	for attempt := 0; attempt <= maxRetries; attempt++ {
		out, err := p.Runner.Run(ctx, prompt, llm.WithJSONSchema(StructuredNoteSchema))
		if err != nil {
			if errors.Is(err, llm.ErrModelOff) {
				return nil, err
			}
			return nil, fmt.Errorf("llm call (attempt %d): %w", attempt+1, err)
		}

		// Strip markdown code fences if Claude wraps JSON in them.
		out = stripCodeFences(out)

		note, err := ParseStructuredJSON(out)
		if err != nil {
			if attempt < maxRetries {
				log.Printf("warn: parse failed (attempt %d), retrying: %v", attempt+1, err)
				log.Printf("debug: raw output (%d bytes): %s", len(out), truncate(string(out), 500))
				continue
			}
			return nil, fmt.Errorf("parse after %d attempts: %w", attempt+1, err)
		}

		violations := Lint(note)
		if len(violations) == 0 {
			return note, nil
		}

		if attempt < maxRetries {
			log.Printf("warn: lint violations (attempt %d): %s", attempt+1, FormatViolations(violations))
			prompt = BuildRetryPrompt(p.Taxonomy, rawBody, violations, note)
			continue
		}

		// Final attempt still has violations — proceed anyway with warning.
		log.Printf("warn: proceeding with %d lint violations after %d retries", len(violations), maxRetries)
		return note, nil
	}

	return nil, fmt.Errorf("unreachable")
}

func (p *Processor) writeRawDirect(rawContent string, processingPath string) error {
	// Write the raw note as-is to knowledge/unsorted/.
	dir := filepath.Join(p.VaultPath, "knowledge", "general", "unsorted")
	if err := vault.EnsureDir(dir); err != nil {
		return err
	}

	filename := filepath.Base(processingPath)
	destPath := filepath.Join(dir, filename)

	if err := atomicWrite(destPath, rawContent); err != nil {
		return err
	}

	relPath := filepath.Join("knowledge", "general", "unsorted", filename)
	log.Printf("ngram: raw (MODEL=off) → %s", relPath)

	// Index if possible.
	if p.SearchClient != nil {
		doc, err := search.ParseNoteFile(destPath, p.VaultPath)
		if err == nil {
			p.SearchClient.IndexNote(*doc)
		}
	}

	p.gitCommit(relPath, filename, "raw")
	return p.archiveRaw(processingPath)
}

func (p *Processor) gitCommit(relPath, noteID, source string) {
	msg := fmt.Sprintf("ngram: structured %s from %s", noteID, source)

	add := exec.Command("git", "add", relPath)
	add.Dir = p.VaultPath
	if err := add.Run(); err != nil {
		log.Printf("warn: git add: %v", err)
		return
	}

	commit := exec.Command("git", "commit", "-m", msg)
	commit.Dir = p.VaultPath
	if err := commit.Run(); err != nil {
		log.Printf("warn: git commit: %v", err)
	}
}

func (p *Processor) archiveRaw(processingPath string) error {
	archiveDir := filepath.Join(p.VaultPath, "_archive")
	if err := vault.EnsureDir(archiveDir); err != nil {
		return err
	}
	dest := filepath.Join(archiveDir, filepath.Base(processingPath))
	return os.Rename(processingPath, dest)
}

func (p *Processor) logUsage(noteID, component string, durationMs int64) {
	metaDir := filepath.Join(p.VaultPath, "_meta")
	vault.EnsureDir(metaDir)

	entry := map[string]interface{}{
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"component":   component,
		"note_id":     noteID,
		"duration_ms": durationMs,
	}
	data, _ := json.Marshal(entry)

	f, err := os.OpenFile(filepath.Join(metaDir, "api-usage.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(data)
	f.Write([]byte("\n"))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// atomicWrite writes content to a file atomically via temp file + rename.
func atomicWrite(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// parseInboxMeta extracts source, box, and phase from inbox YAML frontmatter.
func parseInboxMeta(content string) (source, box, phase string) {
	source = "terminal"
	lines := strings.Split(content, "\n")
	inFM := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFM {
				return
			}
			inFM = true
			continue
		}
		if !inFM {
			continue
		}
		if k, v, ok := parseYAMLLine(trimmed); ok {
			switch k {
			case "source":
				source = v
			case "box":
				box = v
			case "phase":
				phase = v
			}
		}
	}
	return
}

func parseYAMLLine(line string) (key, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	value = strings.Trim(value, `"'`)
	return key, value, true
}

// stripFrontmatter removes the YAML frontmatter block from content.
func stripFrontmatter(content string) string {
	lines := strings.Split(content, "\n")
	inFM := false
	start := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFM {
				inFM = true
				continue
			}
			start = i + 1
			break
		}
	}
	if start > 0 && start < len(lines) {
		return strings.TrimSpace(strings.Join(lines[start:], "\n"))
	}
	return content
}

// StripCodeFencesExported is the exported version of stripCodeFences.
func StripCodeFencesExported(data []byte) []byte {
	return stripCodeFences(data)
}

// stripCodeFences removes markdown code fences wrapping JSON.
func stripCodeFences(data []byte) []byte {
	s := bytes.TrimSpace(data)
	if bytes.HasPrefix(s, []byte("```")) {
		// Remove first line.
		if idx := bytes.IndexByte(s, '\n'); idx >= 0 {
			s = s[idx+1:]
		}
		// Remove last line if it's ```.
		if bytes.HasSuffix(s, []byte("```")) {
			s = s[:len(s)-3]
		}
		s = bytes.TrimSpace(s)
	}
	return s
}
