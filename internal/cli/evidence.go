package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/vault"
)

var evidenceCmd = &cobra.Command{
	Use:   "evidence <note-id>",
	Short: "Display the evidence chain for a note",
	Args:  cobra.ExactArgs(1),
	RunE:  evidenceRun,
}

func init() {
	rootCmd.AddCommand(evidenceCmd)
}

func evidenceRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	noteID := args[0]
	path := vault.FindNoteByID(c.VaultPath, noteID)
	if path == "" {
		return fmt.Errorf("note %s not found", noteID)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)

	title := vault.ParseFrontmatterField(content, "title")
	command := parseNestedField(content, "evidence", "command")
	captured := parseNestedField(content, "evidence", "captured")
	tool := parseNestedField(content, "evidence", "tool")
	sourceFile := parseNestedField(content, "evidence", "source_file")
	sessionID := parseNestedField(content, "evidence", "session_id")

	fmt.Printf("Evidence Chain for: %s\n", title)
	fmt.Println(strings.Repeat("━", 50))

	if command != "" {
		fmt.Printf("Command:    %s\n", command)
	}
	if captured != "" {
		fmt.Printf("Captured:   %s\n", captured)
	}
	if tool != "" {
		fmt.Printf("Tool:       %s\n", tool)
	}
	if sourceFile != "" {
		fmt.Printf("Source:     %s\n", sourceFile)
	}
	if sessionID != "" {
		fmt.Printf("Session:    %s\n", sessionID)
	}

	if command == "" && tool == "" && sessionID == "" {
		fmt.Println("(no evidence chain recorded)")
	}

	return nil
}

// parseNestedField reads a field nested under a parent key in YAML frontmatter.
// e.g., parseNestedField(content, "evidence", "command") reads evidence.command.
func parseNestedField(content, parent, field string) string {
	lines := strings.Split(content, "\n")
	inFM := false
	inParent := false
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
		// Check if we're entering the parent block.
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if strings.HasPrefix(trimmed, parent+":") {
				inParent = true
				continue
			}
			inParent = false
			continue
		}
		if inParent {
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
	}
	return ""
}
