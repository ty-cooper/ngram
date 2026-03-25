package dream

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ty-cooper/ngram/internal/llm"
	"github.com/ty-cooper/ngram/internal/search"
	"github.com/ty-cooper/ngram/internal/taxonomy"
)

type Action struct {
	Type        string   `json:"type"` // merge, archive, recluster, retag, nothing
	NoteIDs     []string `json:"note_ids"`
	Reason      string   `json:"reason"`
	MergedTitle string   `json:"merged_title,omitempty"`
	MergedBody  string   `json:"merged_body,omitempty"`
	OldClusters []string `json:"old_clusters,omitempty"`
	NewCluster  string   `json:"new_cluster,omitempty"`
	NewTags     []string `json:"new_tags,omitempty"`
}

type Report struct {
	Date       string   `json:"date"`
	Merges     []Action `json:"merges"`
	Archives   []Action `json:"archives"`
	Reclusters []Action `json:"reclusters"`
	Retags     []Action `json:"retags"`
	Reviewed   int      `json:"reviewed"`
	NoAction   int      `json:"no_action"`
}

type Runner struct {
	VaultPath string
	Search    *search.Client
	LLM       *llm.Runner
}

func (r *Runner) Run(ctx context.Context) (*Report, error) {
	report := &Report{Date: time.Now().Format("2006-01-02")}

	// 1. Load all notes and filter by dream state.
	allNotes, err := r.loadNotes()
	if err != nil {
		return nil, fmt.Errorf("load notes: %w", err)
	}

	state := loadState(r.VaultPath)
	var notes []noteEntry
	for _, n := range allNotes {
		if state.needsReview(n.ID, n.Modified) {
			notes = append(notes, n)
		}
	}
	report.Reviewed = len(notes)
	log.Printf("dream: loaded %d notes, %d need review", len(allNotes), len(notes))

	if len(notes) == 0 {
		return report, nil
	}

	// 2. Dedup pass — find similar pairs via Meilisearch.
	dupGroups, err := r.findDuplicates(ctx, notes)
	if err != nil {
		log.Printf("dream: dedup pass error: %v", err)
	} else {
		for _, group := range dupGroups {
			action, err := r.decideMerge(ctx, group)
			if err != nil {
				log.Printf("dream: merge decision error: %v", err)
				continue
			}
			if action.Type == "merge" {
				report.Merges = append(report.Merges, action)
			}
		}
	}

	// 3. Quality pass — find notes that are too short, too long, or missing structure.
	quality, err := r.qualitySweep(ctx, notes)
	if err != nil {
		log.Printf("dream: quality pass error: %v", err)
	} else {
		for _, a := range quality {
			switch a.Type {
			case "archive":
				report.Archives = append(report.Archives, a)
			case "retag":
				report.Retags = append(report.Retags, a)
			}
		}
	}

	// 4. Cluster pass — detect near-synonym clusters.
	clusters, err := r.clusterSweep(ctx, notes)
	if err != nil {
		log.Printf("dream: cluster pass error: %v", err)
	} else {
		report.Reclusters = append(report.Reclusters, clusters...)
	}

	report.NoAction = report.Reviewed - len(report.Merges) - len(report.Archives) - len(report.Reclusters) - len(report.Retags)
	if report.NoAction < 0 {
		report.NoAction = 0
	}

	// Stamp all reviewed notes so they're skipped until modified.
	now := time.Now().UTC()
	for _, n := range notes {
		state[n.ID] = now
	}
	if err := saveState(r.VaultPath, state); err != nil {
		log.Printf("dream: save state failed: %v", err)
	}

	return report, nil
}

type noteEntry struct {
	ID       string
	Path     string
	Title    string
	Body     string
	Tags     []string
	Domain   string
	Cluster  string
	Modified time.Time
}

func (r *Runner) loadNotes() ([]noteEntry, error) {
	knowledgeDir := filepath.Join(r.VaultPath, "knowledge")
	var notes []noteEntry

	err := filepath.Walk(knowledgeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		entry := noteEntry{Path: path}

		// Parse frontmatter.
		if strings.HasPrefix(content, "---\n") {
			parts := strings.SplitN(content[4:], "\n---\n", 2)
			if len(parts) == 2 {
				fm := parts[0]
				entry.Body = strings.TrimSpace(parts[1])
				for _, line := range strings.Split(fm, "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "id: ") {
						entry.ID = strings.Trim(strings.TrimPrefix(line, "id: "), "\"")
					} else if strings.HasPrefix(line, "title: ") {
						entry.Title = strings.Trim(strings.TrimPrefix(line, "title: "), "\"")
					} else if strings.HasPrefix(line, "domain: ") {
						entry.Domain = strings.Trim(strings.TrimPrefix(line, "domain: "), "\"")
					} else if strings.HasPrefix(line, "topic_cluster: ") {
						entry.Cluster = strings.Trim(strings.TrimPrefix(line, "topic_cluster: "), "\"")
					} else if strings.HasPrefix(line, "modified: ") {
						val := strings.Trim(strings.TrimPrefix(line, "modified: "), "\"")
						if t, err := time.Parse(time.RFC3339, val); err == nil {
							entry.Modified = t
						}
					}
				}
			}
		}
		if entry.ID == "" {
			entry.ID = strings.TrimSuffix(filepath.Base(path), ".md")
		}
		notes = append(notes, entry)
		return nil
	})

	return notes, err
}

type dupGroup struct {
	source  noteEntry
	matches []noteEntry
}

func (r *Runner) findDuplicates(ctx context.Context, notes []noteEntry) ([]dupGroup, error) {
	seen := map[string]bool{}
	var groups []dupGroup

	for _, note := range notes {
		if seen[note.ID] || note.Title == "" {
			continue
		}
		// Search for similar notes by title + first 200 chars of body.
		query := note.Title
		if len(note.Body) > 200 {
			query += " " + note.Body[:200]
		} else if note.Body != "" {
			query += " " + note.Body
		}

		similar, err := r.Search.FindSimilar(query, 6)
		if err != nil {
			continue
		}

		var matches []noteEntry
		for _, hit := range similar {
			if hit.ID == note.ID {
				continue
			}
			if seen[hit.ID] {
				continue
			}
			if hit.Score < 0.6 { // Lowered for hybrid search score distribution
				continue
			}
			matches = append(matches, noteEntry{
				ID:    hit.ID,
				Path:  hit.FilePath,
				Title: hit.Title,
			})
		}
		if len(matches) > 0 {
			seen[note.ID] = true
			for _, m := range matches {
				seen[m.ID] = true
			}
			groups = append(groups, dupGroup{source: note, matches: matches[:min(3, len(matches))]})
		}
	}
	return groups, nil
}

func (r *Runner) decideMerge(ctx context.Context, group dupGroup) (Action, error) {
	// Build prompt with source + matches.
	var noteDescriptions strings.Builder
	fmt.Fprintf(&noteDescriptions, "SOURCE NOTE:\nID: %s\nTitle: %s\nBody (first 500 chars): %.500s\n\n", group.source.ID, group.source.Title, group.source.Body)
	for i, m := range group.matches {
		data, _ := os.ReadFile(m.Path)
		body := ""
		if content := string(data); strings.Contains(content, "\n---\n") {
			parts := strings.SplitN(content, "\n---\n", 2)
			if len(parts) == 2 {
				body = parts[1]
			}
		}
		fmt.Fprintf(&noteDescriptions, "SIMILAR NOTE %d:\nID: %s\nTitle: %s\nBody (first 500 chars): %.500s\n\n", i+1, m.ID, m.Title, body)
	}

	prompt := fmt.Sprintf(`You are reviewing notes for a knowledge base consolidation.

%s

Decide ONE of:
- "merge": These notes cover the same topic and should be combined into one atomic note. Provide merged_title and merged_body.
- "nothing": These notes are distinct enough to keep separate.

Return JSON only:
{"type":"merge","note_ids":["id1","id2"],"reason":"...","merged_title":"...","merged_body":"..."}
or
{"type":"nothing","note_ids":["id1","id2"],"reason":"..."}`, noteDescriptions.String())

	out, err := r.LLM.Run(ctx, prompt)
	if err != nil {
		return Action{Type: "nothing"}, err
	}

	out = stripCodeFences(out)
	var action Action
	if err := json.Unmarshal(out, &action); err != nil {
		return Action{Type: "nothing"}, fmt.Errorf("parse merge decision: %w", err)
	}
	return action, nil
}

func (r *Runner) qualitySweep(ctx context.Context, notes []noteEntry) ([]Action, error) {
	var actions []Action
	for _, note := range notes {
		// Archive empty or near-empty notes.
		bodyLen := len(strings.TrimSpace(note.Body))
		if bodyLen < 20 {
			actions = append(actions, Action{
				Type:    "archive",
				NoteIDs: []string{note.ID},
				Reason:  fmt.Sprintf("note body is %d chars, effectively empty", bodyLen),
			})
		}
	}
	return actions, nil
}

func (r *Runner) clusterSweep(ctx context.Context, notes []noteEntry) ([]Action, error) {
	// Load allowed clusters from taxonomy.
	tc, err := taxonomy.LoadClusters(r.VaultPath)
	if err != nil {
		return nil, fmt.Errorf("load topic clusters: %w", err)
	}

	// Build allowed cluster set from taxonomy.
	allowed := map[string]bool{}
	for _, dc := range tc.Domains {
		for name := range dc.Clusters {
			allowed[name] = true
		}
	}

	// Collect clusters actually in use from notes.
	clusterNotes := map[string][]string{}
	for _, n := range notes {
		if n.Cluster != "" {
			clusterNotes[n.Cluster] = append(clusterNotes[n.Cluster], n.ID)
		}
	}

	if len(clusterNotes) < 2 {
		return nil, nil
	}

	inUse := make([]string, 0, len(clusterNotes))
	for c := range clusterNotes {
		inUse = append(inUse, c)
	}

	allowedList := make([]string, 0, len(allowed))
	for c := range allowed {
		allowedList = append(allowedList, c)
	}

	prompt := fmt.Sprintf(`Review these topic clusters for a knowledge base. Identify any that are near-synonyms and should be merged.

Clusters in use: %s

Allowed clusters (from taxonomy): %s

Rules:
- new_cluster MUST be one of the allowed clusters. If none fits, return [].
- old_clusters lists the clusters being replaced.

For each merge, return:
[{"type":"recluster","old_clusters":["old-name-1","old-name-2"],"reason":"...","new_cluster":"canonical-name"}]
If no merges needed, return [].`, strings.Join(inUse, ", "), strings.Join(allowedList, ", "))

	out, err := r.LLM.Run(ctx, prompt)
	if err != nil {
		return nil, err
	}

	out = stripCodeFences(out)
	var actions []Action
	if err := json.Unmarshal(out, &actions); err != nil {
		return nil, fmt.Errorf("parse cluster sweep: %w", err)
	}

	// Filter out any actions with new_cluster not in allowed set.
	var valid []Action
	for _, a := range actions {
		if allowed[a.NewCluster] {
			valid = append(valid, a)
		} else {
			log.Printf("dream: dropping recluster → %q (not in taxonomy)", a.NewCluster)
		}
	}
	return valid, nil
}

func stripCodeFences(data []byte) []byte {
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		if strings.HasSuffix(s, "```") {
			s = s[:len(s)-3]
		}
		s = strings.TrimSpace(s)
	}
	return []byte(s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
