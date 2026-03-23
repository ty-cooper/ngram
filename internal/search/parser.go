package search

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// noteFrontmatter represents the YAML frontmatter of a structured vault note.
type noteFrontmatter struct {
	Title       string   `yaml:"title"`
	Summary     string   `yaml:"summary"`
	Domain      string   `yaml:"domain"`
	TopicCluster string  `yaml:"topic_cluster"`
	ContentType string   `yaml:"content_type"`
	Tags        []string `yaml:"tags"`
	Box         string   `yaml:"box"`
	Phase       string   `yaml:"phase"`
	Engagement  string   `yaml:"engagement"`
	Captured    string   `yaml:"captured"`
	ID          string   `yaml:"id"`
}

// ParseNoteFile reads a markdown file and returns a NoteDocument for indexing.
func ParseNoteFile(path, vaultPath string) (*NoteDocument, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)

	// Read frontmatter between --- delimiters.
	var fmLines []string
	inFrontmatter := false
	bodyStart := false
	var bodyLines []string

	for scanner.Scan() {
		line := scanner.Text()
		if !inFrontmatter && !bodyStart && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			inFrontmatter = false
			bodyStart = true
			continue
		}
		if inFrontmatter {
			fmLines = append(fmLines, line)
		} else if bodyStart {
			bodyLines = append(bodyLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	var fm noteFrontmatter
	if len(fmLines) > 0 {
		if err := yaml.Unmarshal([]byte(strings.Join(fmLines, "\n")), &fm); err != nil {
			return nil, fmt.Errorf("parse frontmatter in %s: %w", path, err)
		}
	}

	body := strings.TrimSpace(strings.Join(bodyLines, "\n"))

	relPath, _ := filepath.Rel(vaultPath, path)

	id := fm.ID
	if id == "" {
		// Fall back to filename without extension as ID.
		id = strings.TrimSuffix(filepath.Base(path), ".md")
	}

	var captured int64
	if fm.Captured != "" {
		if t, err := time.Parse(time.RFC3339, fm.Captured); err == nil {
			captured = t.Unix()
		}
	}

	return &NoteDocument{
		ID:          id,
		Title:       fm.Title,
		Body:        body,
		Summary:     fm.Summary,
		Tags:        fm.Tags,
		Domain:      fm.Domain,
		ContentType: fm.ContentType,
		Box:         fm.Box,
		Phase:       fm.Phase,
		Engagement:  fm.Engagement,
		FilePath:    relPath,
		Captured:    captured,
	}, nil
}

// WalkVault finds all indexable .md files in the vault, excluding system directories.
func WalkVault(vaultPath string) ([]string, error) {
	skipDirs := map[string]bool{
		"_inbox":      true,
		"_processing": true,
		"_archive":    true,
		"_meta":       true,
		"_trash":      true,
		"_templates":  true,
		".obsidian":   true,
		".git":        true,
	}

	var files []string
	err := filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(info.Name(), ".md") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
