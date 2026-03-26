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
	Title        string   `yaml:"title"`
	Summary      string   `yaml:"summary"`
	Domain       string   `yaml:"domain"`
	TopicCluster string   `yaml:"topic_cluster"`
	ContentType  string   `yaml:"content_type"`
	Tags         []string `yaml:"tags"`
	Box          string   `yaml:"box"`
	Phase        string   `yaml:"phase"`
	Engagement   string   `yaml:"engagement"`
	Captured     string   `yaml:"captured"`
	Created      string   `yaml:"created"`
	ID           string   `yaml:"id"`
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

	fullBody := strings.TrimSpace(strings.Join(bodyLines, "\n"))

	// Parse footer metadata (after last ---).
	// New format: content above ---, metadata key: value lines below.
	body := fullBody
	if lastSep := strings.LastIndex(fullBody, "\n---\n"); lastSep >= 0 {
		body = strings.TrimSpace(fullBody[:lastSep])
		footer := fullBody[lastSep+5:]
		for _, line := range strings.Split(footer, "\n") {
			line = strings.TrimSpace(line)
			if k, v, ok := strings.Cut(line, ": "); ok {
				switch k {
				case "id":
					if fm.ID == "" {
						fm.ID = v
					}
				case "type":
					if fm.ContentType == "" {
						fm.ContentType = v
					}
				case "domain":
					if fm.Domain == "" {
						fm.Domain = v
					}
				case "topic_cluster":
					if fm.TopicCluster == "" {
						fm.TopicCluster = v
					}
				case "box":
					if fm.Box == "" {
						fm.Box = v
					}
				case "phase":
					if fm.Phase == "" {
						fm.Phase = v
					}
				case "tags":
					if len(fm.Tags) == 0 {
						for _, t := range strings.Fields(v) {
							fm.Tags = append(fm.Tags, strings.TrimPrefix(t, "#"))
						}
					}
				}
			}
		}
	}

	// Extract title from H1 if not in frontmatter.
	if fm.Title == "" {
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "# ") {
				fm.Title = strings.TrimPrefix(line, "# ")
				break
			}
		}
	}

	// Extract summary from italic line after H1.
	if fm.Summary == "" {
		lines := strings.Split(body, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "# ") && i+2 < len(lines) {
				next := strings.TrimSpace(lines[i+2])
				if strings.HasPrefix(next, "*") && strings.HasSuffix(next, "*") && !strings.HasPrefix(next, "**") {
					fm.Summary = strings.Trim(next, "*")
				}
				break
			}
		}
	}

	relPath, _ := filepath.Rel(vaultPath, path)

	id := fm.ID
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(path), ".md")
	}

	var captured int64
	if fm.Created != "" {
		if t, err := time.Parse(time.RFC3339, fm.Created); err == nil {
			captured = t.Unix()
		}
	} else if fm.Captured != "" {
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
