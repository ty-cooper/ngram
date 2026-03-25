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

// Process runs the full pipeline for a single file or capture bundle in _inbox/.
func (p *Processor) Process(ctx context.Context, inboxPath string) error {
	start := time.Now()

	// Check if this is a capture bundle (directory with manifest.yml).
	if IsBundle(inboxPath) {
		return p.processBundle(ctx, inboxPath, start)
	}

	// Check if this is a standalone image (photo of handwritten notes, screenshot, etc).
	if IsImage(filepath.Base(inboxPath)) {
		return p.processImage(ctx, inboxPath, start)
	}

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
	meta := parseInboxMetaFull(rawContent)
	source, box, phase := meta.Source, meta.Box, meta.Phase
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

	notes, err := p.structureWithRetry(ctx, body, maxRetries)
	if err != nil {
		if errors.Is(err, ErrDiscard) {
			log.Printf("ngram: discarded — %v", err)
			if archErr := p.archiveRaw(procPath); archErr != nil {
				log.Printf("warn: archive discarded: %v", archErr)
			}
			return nil
		}
		return fmt.Errorf("structure: %w", err)
	}

	// Process each atomic note.
	for _, structured := range notes {
		// Resolve taxonomy.
		if p.Taxonomy != nil {
			structured.Domain = p.Taxonomy.ResolveDomain(structured.Domain)
			structured.Tags = p.Taxonomy.ResolveTags(structured.Tags)
			p.Taxonomy.RegisterTags(structured.Tags, p.VaultPath)
			p.Taxonomy.RegisterDomain(structured.Domain, p.VaultPath)
		}

		// Dedup check.
		if p.Dedup != nil {
			result := p.Dedup.Check(ctx, structured, procPath)
			switch result.Action {
			case "duplicate":
				log.Printf("ngram: dedup — duplicate of %s", result.TargetID)
				continue
			case "appended":
				log.Printf("ngram: dedup — appended to %s", result.TargetID)
				p.gitCommit(result.TargetID, result.TargetID, "append")
				continue
			}
		}

		processed := &ProcessedNote{
			StructuredNote: *structured,
			ID:             GenerateID(),
			Source:         source,
			Box:            box,
			Phase:          phase,
			Created:        time.Now().UTC(),
			Evidence: EvidenceChain{
				SourceCommand: meta.Command,
				CapturedAt:    meta.Captured,
				Tool:          meta.Tool,
				SourceFile:    filepath.Base(inboxPath),
				SessionID:     meta.Session,
			},
		}

		// Cross-engagement recall.
		if recalls := p.recallPass(processed); len(recalls) > 0 {
			log.Printf("ngram: recall — %d related notes from other engagements", len(recalls))
			appendRecallSection(processed, recalls)
		}

		dir, filename := Route(processed)
		destDir := filepath.Join(p.VaultPath, dir)
		if err := vault.EnsureDir(destDir); err != nil {
			log.Printf("warn: ensure dir %s: %v", dir, err)
			continue
		}

		destPath := filepath.Join(destDir, filename)
		content := BuildNoteContent(processed)

		if err := atomicWrite(destPath, content); err != nil {
			log.Printf("warn: write note: %v", err)
			continue
		}

		relPath := filepath.Join(dir, filename)
		log.Printf("ngram: structured %s → %s", processed.ID, relPath)

		if p.SearchClient != nil {
			doc := search.NoteDocument{
				ID:            processed.ID,
				Title:         processed.Title,
				Body:          processed.Body,
				Summary:       processed.Summary,
				Tags:          processed.Tags,
				Domain:        processed.Domain,
				ContentType:   processed.ContentType,
				Box:           processed.Box,
				Phase:         processed.Phase,
				FilePath:      relPath,
				SourceCommand: processed.Evidence.SourceCommand,
				Tool:          processed.Evidence.Tool,
				SessionID:     processed.Evidence.SessionID,
				Captured:      processed.Created.Unix(),
			}
			if err := p.SearchClient.IndexNote(doc); err != nil {
				log.Printf("warn: index failed for %s: %v", processed.ID, err)
			}
		}

		p.gitCommit(relPath, processed.ID, source)
	}

	notify.Send("Ngram", fmt.Sprintf("Structured %d note(s) from input", len(notes)))

	if err := p.archiveRaw(procPath); err != nil {
		log.Printf("warn: archive failed: %v", err)
	}

	duration := time.Since(start).Milliseconds()
	p.logUsage("batch", "processor", duration)

	return nil
}

// processBundle handles a capture session directory with manifest.yml.
func (p *Processor) processBundle(ctx context.Context, inboxPath string, start time.Time) error {
	// Move entire directory to _processing/.
	procDir := filepath.Join(p.VaultPath, "_processing")
	if err := vault.EnsureDir(procDir); err != nil {
		return fmt.Errorf("ensure processing dir: %w", err)
	}
	bundleDir := filepath.Join(procDir, filepath.Base(inboxPath))
	if err := os.Rename(inboxPath, bundleDir); err != nil {
		return fmt.Errorf("move bundle to processing: %w", err)
	}

	bundle, err := LoadBundle(bundleDir)
	if err != nil {
		return fmt.Errorf("load bundle: %w", err)
	}

	if p.Runner.Model == "off" {
		// Write text content as raw note.
		text := BundleTextContent(bundle)
		if text == "" {
			text = "(capture session with screenshots only)"
		}
		raw := fmt.Sprintf("---\nsource: capture-session\n---\n\n%s\n", text)
		return p.writeRawDirect(raw, bundleDir)
	}

	// Build prompt with text content and screenshot references.
	prompt := BuildBundlePrompt(bundle, bundleDir)

	// Collect screenshot paths for Claude vision.
	var imagePaths []string
	for _, item := range bundle.Items {
		if item.Type == "screenshot" {
			imagePaths = append(imagePaths, filepath.Join(bundleDir, item.File))
		}
	}

	// Structure via instructor.
	opts := []llm.RunOption{
		llm.WithSystemPrompt(StructuringSystemPrompt),
	}
	if len(imagePaths) > 0 {
		opts = append(opts, llm.WithImages(imagePaths))
	}

	var resp StructuredNotesResponse
	if err := p.Runner.Instruct(ctx, prompt, &resp, opts...); err != nil {
		return fmt.Errorf("instruct bundle: %w", err)
	}
	notes, err := ValidateResponse(&resp)
	if err != nil {
		return fmt.Errorf("validate bundle: %w", err)
	}
	structured := notes[0]

	// Resolve taxonomy.
	if p.Taxonomy != nil {
		structured.Domain = p.Taxonomy.ResolveDomain(structured.Domain)
		structured.Tags = p.Taxonomy.ResolveTags(structured.Tags)
	}

	// Create processed note.
	processed := &ProcessedNote{
		StructuredNote: *structured,
		ID:             GenerateID(),
		Source:         "capture-session",
		Box:            bundle.Box,
		Phase:          bundle.Phase,
		Created:        time.Now().UTC(),
		Evidence: EvidenceChain{
			CapturedAt: time.Now().UTC().Format(time.RFC3339),
			SessionID:  bundle.SessionID,
			SourceFile: filepath.Base(inboxPath),
		},
	}

	// Route and write note.
	dir, filename := Route(processed)
	destDir := filepath.Join(p.VaultPath, dir)
	if err := vault.EnsureDir(destDir); err != nil {
		return fmt.Errorf("ensure dir %s: %w", dir, err)
	}

	// Copy screenshots to _assets/ and embed links in the note.
	assetsDir := filepath.Join(p.VaultPath, "_assets")
	vault.EnsureDir(assetsDir)
	var screenshots []string
	for _, item := range bundle.Items {
		if item.Type == "screenshot" {
			src := filepath.Join(bundleDir, item.File)
			assetName := fmt.Sprintf("%s-%s", processed.ID, item.File)
			dst := filepath.Join(assetsDir, assetName)
			if data, err := os.ReadFile(src); err == nil {
				os.WriteFile(dst, data, 0o644)
				screenshots = append(screenshots, assetName)
			}
		}
	}
	if len(screenshots) > 0 {
		processed.Evidence.Screenshots = screenshots
		var embeds strings.Builder
		embeds.WriteString("\n\n## Screenshots\n\n")
		for _, ss := range screenshots {
			fmt.Fprintf(&embeds, "![[%s]]\n\n", ss)
		}
		processed.Body += embeds.String()
	}

	destPath := filepath.Join(destDir, filename)
	content := BuildNoteContent(processed)

	if err := atomicWrite(destPath, content); err != nil {
		return fmt.Errorf("write note: %w", err)
	}

	relPath := filepath.Join(dir, filename)
	log.Printf("ngram: structured bundle %s → %s (%d items)", processed.ID, relPath, len(bundle.Items))

	// Index.
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

	notify.Send("Ngram", fmt.Sprintf("Structured bundle: %s → %s", processed.Title, relPath))
	p.gitCommit(relPath, processed.ID, "capture-session")

	// Archive the bundle directory.
	archiveDir := filepath.Join(p.VaultPath, "_archive")
	vault.EnsureDir(archiveDir)
	os.Rename(bundleDir, filepath.Join(archiveDir, filepath.Base(bundleDir)))

	duration := time.Since(start).Milliseconds()
	p.logUsage(processed.ID, "processor-bundle", duration)

	return nil
}

// processImage handles standalone image files (photos, screenshots, handwritten notes).
// Sends the image to Claude via vision and structures the extracted content.
func (p *Processor) processImage(ctx context.Context, inboxPath string, start time.Time) error {
	procPath, err := p.moveToProcessing(inboxPath)
	if err != nil {
		return fmt.Errorf("move image to processing: %w", err)
	}

	if p.Runner.Model == "off" {
		return p.writeRawDirect(fmt.Sprintf("---\nsource: image\n---\n\n(image: %s)\n", filepath.Base(inboxPath)), procPath)
	}

	prompt := "Extract all text, diagrams, and knowledge from this image. If it contains handwritten notes, transcribe them accurately. Structure the content as atomic notes."

	var resp StructuredNotesResponse
	if err := p.Runner.Instruct(ctx, prompt, &resp,
		llm.WithSystemPrompt(StructuringSystemPrompt),
		llm.WithImages([]string{procPath}),
	); err != nil {
		if errors.Is(err, llm.ErrModelOff) {
			return err
		}
		return fmt.Errorf("instruct image: %w", err)
	}

	notes, err := ValidateResponse(&resp)
	if err != nil {
		if errors.Is(err, ErrDiscard) {
			log.Printf("ngram: image discarded — %v", err)
			return p.archiveRaw(procPath)
		}
		return fmt.Errorf("validate image: %w", err)
	}

	for _, structured := range notes {
		if p.Taxonomy != nil {
			structured.Domain = p.Taxonomy.ResolveDomain(structured.Domain)
			structured.Tags = p.Taxonomy.ResolveTags(structured.Tags)
			p.Taxonomy.RegisterTags(structured.Tags, p.VaultPath)
			p.Taxonomy.RegisterDomain(structured.Domain, p.VaultPath)
		}

		processed := &ProcessedNote{
			StructuredNote: *structured,
			ID:             GenerateID(),
			Source:         "image",
			Created:        time.Now().UTC(),
			Evidence: EvidenceChain{
				CapturedAt: time.Now().UTC().Format(time.RFC3339),
				SourceFile: filepath.Base(inboxPath),
			},
		}

		dir, filename := Route(processed)
		destDir := filepath.Join(p.VaultPath, dir)
		if err := vault.EnsureDir(destDir); err != nil {
			log.Printf("warn: ensure dir %s: %v", dir, err)
			continue
		}

		// Copy image to _assets/ and link from the note.
		assetsDir := filepath.Join(p.VaultPath, "_assets")
		vault.EnsureDir(assetsDir)
		imgName := fmt.Sprintf("%s-%s", processed.ID, filepath.Base(inboxPath))
		imgDest := filepath.Join(assetsDir, imgName)
		if imgData, err := os.ReadFile(procPath); err == nil {
			os.WriteFile(imgDest, imgData, 0o644)
			processed.Evidence.Screenshots = []string{imgName}
			processed.Body += fmt.Sprintf("\n\n## Source Image\n\n![[%s]]\n", imgName)
		}

		destPath := filepath.Join(destDir, filename)
		content := BuildNoteContent(processed)
		if err := atomicWrite(destPath, content); err != nil {
			log.Printf("warn: write note: %v", err)
			continue
		}

		relPath := filepath.Join(dir, filename)
		log.Printf("ngram: structured image %s → %s", processed.ID, relPath)

		if p.SearchClient != nil {
			doc := search.NoteDocument{
				ID:          processed.ID,
				Title:       processed.Title,
				Body:        processed.Body,
				Summary:     processed.Summary,
				Tags:        processed.Tags,
				Domain:      processed.Domain,
				ContentType: processed.ContentType,
				FilePath:    relPath,
				Captured:    processed.Created.Unix(),
			}
			if err := p.SearchClient.IndexNote(doc); err != nil {
				log.Printf("warn: index failed for %s: %v", processed.ID, err)
			}
		}

		p.gitCommit(relPath, processed.ID, "image")
	}

	notify.Send("Ngram", fmt.Sprintf("Structured %d note(s) from image", len(notes)))

	if err := p.archiveRaw(procPath); err != nil {
		log.Printf("warn: archive image: %v", err)
	}

	duration := time.Since(start).Milliseconds()
	p.logUsage("image", "processor-image", duration)
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

func (p *Processor) structureWithRetry(ctx context.Context, rawBody string, maxRetries int) ([]*StructuredNote, error) {
	prompt := BuildStructuringPrompt(p.Taxonomy, rawBody, p.VaultPath)

	var resp StructuredNotesResponse
	err := p.Runner.Instruct(ctx, prompt, &resp,
		llm.WithSystemPrompt(StructuringSystemPrompt),
	)
	if err != nil {
		if errors.Is(err, llm.ErrModelOff) {
			return nil, err
		}
		return nil, fmt.Errorf("instruct: %w", err)
	}

	return ValidateResponse(&resp)
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
	// Skip if vault is not a git repo.
	check := exec.Command("git", "rev-parse", "--git-dir")
	check.Dir = p.VaultPath
	if err := check.Run(); err != nil {
		return
	}

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

// InboxMeta holds all metadata parsed from an inbox note's frontmatter.
type InboxMeta struct {
	Source    string
	Box      string
	Phase    string
	Command  string
	Tool     string
	Captured string
	Session  string
}

// parseInboxMeta extracts source, box, and phase from inbox YAML frontmatter.
func parseInboxMeta(content string) (source, box, phase string) {
	meta := parseInboxMetaFull(content)
	return meta.Source, meta.Box, meta.Phase
}

// parseInboxMetaFull extracts all metadata from inbox YAML frontmatter.
func parseInboxMetaFull(content string) InboxMeta {
	meta := InboxMeta{Source: "terminal"}
	lines := strings.Split(content, "\n")
	inFM := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFM {
				return meta
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
				meta.Source = v
			case "box":
				meta.Box = v
			case "phase":
				meta.Phase = v
			case "command":
				meta.Command = v
			case "tool":
				meta.Tool = v
			case "captured":
				meta.Captured = v
			case "session_id":
				meta.Session = v
			}
		}
	}
	return meta
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
