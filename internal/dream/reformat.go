package dream

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

// ReformatAction describes an LLM-proposed rewrite for a note that failed lint.
type ReformatAction struct {
	NoteID      string   `json:"note_id"`
	Violations  []string `json:"violations"`
	RewriteBody string   `json:"rewrite_body" jsonschema:"description=The complete rewritten note body with all violations fixed"`
}

// reformatPass sends lint-failing notes to the LLM for rewriting.
// Only called for notes with fixable violations. Returns actions for the apply step.
func (r *Runner) reformatPass(ctx context.Context, results []LintResult) []Action {
	var actions []Action

	for _, result := range results {
		// Skip notes that only have informational violations (missing-related is added by pipeline).
		fixable := filterFixable(result.Violations)
		if len(fixable) == 0 {
			continue
		}

		// Read current note content.
		data, err := os.ReadFile(result.Note.Path)
		if err != nil {
			log.Printf("dream: reformat — can't read %s: %v", result.Note.ID, err)
			continue
		}

		content := string(data)
		var body string
		if strings.HasPrefix(content, "---\n") {
			parts := strings.SplitN(content[4:], "\n---\n", 2)
			if len(parts) == 2 {
				body = parts[1]
			}
		}
		if body == "" {
			body = content
		}

		var violationDescs []string
		for _, v := range fixable {
			violationDescs = append(violationDescs, fmt.Sprintf("- [%s] %s", v.Rule, v.Message))
		}

		prompt := fmt.Sprintf(`Rewrite this note to fix the following lint violations.
Preserve ALL information — do not remove or summarize any content.

VIOLATIONS TO FIX:
%s

RULES:
- Code blocks MUST start with a # [Tool/Context] comment (e.g. # [Mimikatz], # [PowerShell], # [Bash], # [Target Shell])
- If a code block's tool/context is ambiguous, infer from the commands used
- Remove any tag that duplicates the domain name
- Remove system tags: inbox, test, log, capture-session
- Keep the note under 500 words if possible — if the note is too long, split into the most important atomic concept and note what was removed
- Preserve all code blocks, commands, and technical details exactly
- Use {{PLACEHOLDER}} syntax for variable values in commands

CURRENT NOTE BODY:
%s

Return ONLY the rewritten note body (markdown). No JSON wrapper.`, strings.Join(violationDescs, "\n"), body)

		resp, err := r.LLM.Run(ctx, prompt)
		if err != nil {
			log.Printf("dream: reformat LLM failed for %s: %v", result.Note.ID, err)
			continue
		}

		rewritten := strings.TrimSpace(string(resp))
		if rewritten == "" || rewritten == "ok" {
			continue
		}

		actions = append(actions, Action{
			Type:       "reformat",
			NoteIDs:    []string{result.Note.ID},
			Reason:     fmt.Sprintf("lint violations: %s", strings.Join(ruleNames(fixable), ", ")),
			MergedBody: rewritten,
		})
	}

	return actions
}

// fixableRules are violations the LLM can actually fix by rewriting.
var fixableRules = map[string]bool{
	"missing-context-comment": true,
	"over-word-limit":         true,
	"domain-as-tag":           true,
	"system-tag":              true,
	"near-duplicate-tag":      true,
	"too-many-tags":           true,
}

func filterFixable(violations []LintViolation) []LintViolation {
	var out []LintViolation
	for _, v := range violations {
		if fixableRules[v.Rule] {
			out = append(out, v)
		}
	}
	return out
}

func ruleNames(violations []LintViolation) []string {
	seen := map[string]bool{}
	var names []string
	for _, v := range violations {
		if !seen[v.Rule] {
			seen[v.Rule] = true
			names = append(names, v.Rule)
		}
	}
	return names
}
