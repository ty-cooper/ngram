package search

import (
	"fmt"
	"strings"
)

// ExtractCommands parses fenced code blocks from a note body and returns
// CommandDocuments for blocks that have a # [Tool] context comment.
func ExtractCommands(noteID, noteTitle, body string, meta CommandMeta) []CommandDocument {
	lines := strings.Split(body, "\n")
	var cmds []CommandDocument
	cmdIdx := 0

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "```") {
			continue
		}

		// Parse language from opening fence.
		lang := strings.TrimPrefix(trimmed, "```")
		lang = strings.TrimSpace(lang)

		// Collect code block content.
		var blockLines []string
		i++
		for i < len(lines) {
			if strings.TrimSpace(lines[i]) == "```" {
				break
			}
			blockLines = append(blockLines, lines[i])
			i++
		}

		if len(blockLines) == 0 {
			continue
		}

		// Check for # [Tool] comment on first line.
		firstLine := strings.TrimSpace(blockLines[0])
		tool, desc := parseToolComment(firstLine)
		if tool == "" {
			continue
		}

		// Command is everything after the comment line.
		var cmdLines []string
		for _, l := range blockLines[1:] {
			// Skip empty lines at start.
			if len(cmdLines) == 0 && strings.TrimSpace(l) == "" {
				continue
			}
			cmdLines = append(cmdLines, l)
		}

		command := strings.TrimSpace(strings.Join(cmdLines, "\n"))
		if command == "" {
			// Single-line command where the comment IS the only line.
			command = firstLine
		}

		cmds = append(cmds, CommandDocument{
			ID:           fmt.Sprintf("%s-cmd-%d", noteID, cmdIdx),
			ParentNoteID: noteID,
			ParentTitle:  noteTitle,
			Tool:         tool,
			Language:     lang,
			Command:      command,
			Description:  desc,
			Phase:        meta.Phase,
			Domain:       meta.Domain,
			Tags:         meta.Tags,
			FilePath:     meta.FilePath,
		})
		cmdIdx++
	}

	return cmds
}

// CommandMeta holds inherited metadata from the parent note.
type CommandMeta struct {
	Phase    string
	Domain   string
	Tags     []string
	FilePath string
}

// parseToolComment extracts tool name and description from a # [Tool] comment.
// Returns empty strings if the line doesn't match the pattern.
func parseToolComment(line string) (tool, description string) {
	// Must start with # or // followed by [
	var rest string
	if strings.HasPrefix(line, "# [") {
		rest = strings.TrimPrefix(line, "# ")
	} else if strings.HasPrefix(line, "// [") {
		rest = strings.TrimPrefix(line, "// ")
	} else if strings.HasPrefix(line, "-- [") {
		rest = strings.TrimPrefix(line, "-- ")
	} else if strings.HasPrefix(line, "REM [") {
		rest = strings.TrimPrefix(line, "REM ")
	} else {
		return "", ""
	}

	// Extract [Tool] part.
	if !strings.HasPrefix(rest, "[") {
		return "", ""
	}
	end := strings.Index(rest, "]")
	if end < 0 {
		return "", ""
	}

	tool = rest[1:end]
	description = strings.TrimSpace(rest[end+1:])

	return tool, description
}
