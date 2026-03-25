package vault

import (
	"os"
	"path/filepath"
	"strings"
)

// FindNoteByID walks the vault looking for a note file matching the given ID prefix.
// Returns the full path or empty string if not found.
func FindNoteByID(vaultPath, id string) string {
	var result string
	filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), id) {
			result = path
			return filepath.SkipAll
		}
		return nil
	})
	return result
}

// ParseFrontmatterField reads a single YAML frontmatter field from a note file.
func ParseFrontmatterField(content, field string) string {
	lines := strings.Split(content, "\n")
	inFM := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if inFM {
				return ""
			}
			inFM = true
			continue
		}
		if !inFM {
			continue
		}
		idx := strings.Index(trimmed, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		if key == field {
			val := strings.TrimSpace(trimmed[idx+1:])
			return strings.Trim(val, `"'`)
		}
	}
	return ""
}
