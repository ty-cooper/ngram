package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ComputeNoteHash generates a SHA256 hash of note content + timestamp + previous hash.
func ComputeNoteHash(content, timestamp, prevHash string) string {
	h := sha256.New()
	h.Write([]byte(content))
	h.Write([]byte(timestamp))
	h.Write([]byte(prevHash))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyChain walks all notes in the vault and checks hash chain integrity.
// Returns a list of broken links (notes where the hash doesn't match).
func VerifyChain(vaultPath string) ([]ChainError, error) {
	var files []string
	filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == ".obsidian" || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(info.Name(), ".md") {
			files = append(files, path)
		}
		return nil
	})

	var errors []ChainError

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		content := string(data)

		hash := extractFrontmatterField(content, "hash")
		if hash == "" {
			continue // No hash = not part of the chain
		}

		timestamp := extractFrontmatterField(content, "created")
		prevHash := extractFrontmatterField(content, "prev_hash")
		body := stripFrontmatterAudit(content)

		computed := ComputeNoteHash(body, timestamp, prevHash)
		if computed != hash {
			rel, _ := filepath.Rel(vaultPath, f)
			errors = append(errors, ChainError{
				FilePath:     rel,
				StoredHash:   hash,
				ComputedHash: computed,
			})
		}
	}

	return errors, nil
}

// ChainError describes a hash chain integrity failure.
type ChainError struct {
	FilePath     string
	StoredHash   string
	ComputedHash string
}

func (e ChainError) String() string {
	return fmt.Sprintf("%s: stored=%s computed=%s", e.FilePath, e.StoredHash[:12], e.ComputedHash[:12])
}

func extractFrontmatterField(content, field string) string {
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
		if strings.HasPrefix(trimmed, field+":") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, field+":"))
			return strings.Trim(val, `"'`)
		}
	}
	return ""
}

func stripFrontmatterAudit(content string) string {
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
