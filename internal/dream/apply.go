package dream

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (r *Runner) Apply(report *Report) error {
	if len(report.Merges) == 0 && len(report.Archives) == 0 && len(report.Reclusters) == 0 && len(report.Retags) == 0 && len(report.Reformats) == 0 {
		log.Println("dream: vault is clean, nothing to do")
		return nil
	}

	// Stash any dirty working tree state (e.g. Obsidian workspace changes).
	r.git("add", "-A")
	r.git("commit", "-m", "existing changes")

	branch := r.nextBranch()

	if err := r.git("checkout", "-b", branch); err != nil {
		return fmt.Errorf("create branch %s: %w", branch, err)
	}

	// Apply merges.
	for _, merge := range report.Merges {
		if err := r.applyMerge(merge); err != nil {
			log.Printf("dream: merge %v failed: %v", merge.NoteIDs, err)
			continue
		}
		r.git("add", "-A")
		r.git("commit", "-m", fmt.Sprintf("dream: merge %s — %s", strings.Join(merge.NoteIDs, " + "), merge.Reason))
	}

	// Apply archives.
	for _, archive := range report.Archives {
		if err := r.applyArchive(archive); err != nil {
			log.Printf("dream: archive %v failed: %v", archive.NoteIDs, err)
			continue
		}
		r.git("add", "-A")
		r.git("commit", "-m", fmt.Sprintf("dream: archive %s — %s", strings.Join(archive.NoteIDs, ", "), archive.Reason))
	}

	// Apply reclusters.
	for _, rc := range report.Reclusters {
		if err := r.applyRecluster(rc); err != nil {
			log.Printf("dream: recluster failed: %v", err)
			continue
		}
		r.git("add", "-A")
		r.git("commit", "-m", fmt.Sprintf("dream: recluster → %s — %s", rc.NewCluster, rc.Reason))
	}

	// Apply reformats.
	for _, rf := range report.Reformats {
		if err := r.applyReformat(rf); err != nil {
			log.Printf("dream: reformat %v failed: %v", rf.NoteIDs, err)
			continue
		}
		r.git("add", "-A")
		r.git("commit", "-m", fmt.Sprintf("dream: reformat %s — %s", strings.Join(rf.NoteIDs, ", "), rf.Reason))
	}

	// Check if there are any commits on this branch beyond main.
	if !r.hasBranchCommits(branch) {
		log.Println("dream: no changes to commit, cleaning up branch")
		r.git("checkout", "main")
		r.git("branch", "-D", branch)
		return nil
	}

	// Push and create PR.
	if err := r.git("push", "-u", "origin", branch); err != nil {
		log.Printf("dream: push failed (no remote?): %v", err)
		r.git("checkout", "main")
		return nil
	}

	r.createPR(report, branch)
	r.git("checkout", "main")
	return nil
}

func (r *Runner) applyMerge(merge Action) error {
	if len(merge.NoteIDs) < 2 || merge.MergedBody == "" {
		return fmt.Errorf("invalid merge action")
	}

	// Find the first note's path to determine the target location.
	var targetPath string
	for _, id := range merge.NoteIDs {
		path := r.findNoteByID(id)
		if path != "" {
			if targetPath == "" {
				targetPath = path
			} else {
				// Delete secondary notes — git history preserves them.
				os.Remove(path)
			}
		}
	}

	if targetPath == "" {
		return fmt.Errorf("no notes found for IDs: %v", merge.NoteIDs)
	}

	// Read original frontmatter, replace body.
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return err
	}

	content := string(data)
	if idx := strings.Index(content[4:], "\n---\n"); idx >= 0 {
		fm := content[:idx+4+5] // frontmatter including closing ---
		newContent := fm + "\n" + merge.MergedBody + "\n"
		return os.WriteFile(targetPath, []byte(newContent), 0o644)
	}

	return os.WriteFile(targetPath, []byte(merge.MergedBody), 0o644)
}

func (r *Runner) applyArchive(archive Action) error {
	for _, id := range archive.NoteIDs {
		path := r.findNoteByID(id)
		if path != "" {
			// Delete the file — git history preserves it.
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) applyRecluster(rc Action) error {
	if len(rc.OldClusters) == 0 || rc.NewCluster == "" {
		return fmt.Errorf("recluster missing old_clusters or new_cluster")
	}

	knowledgeDir := filepath.Join(r.VaultPath, "knowledge")
	return filepath.Walk(knowledgeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		changed := false
		for _, old := range rc.OldClusters {
			if old == rc.NewCluster {
				continue
			}
			for _, pattern := range []string{
				"topic_cluster: \"" + old + "\"",
				"topic_cluster: " + old,
			} {
				if strings.Contains(content, pattern) {
					replacement := "topic_cluster: \"" + rc.NewCluster + "\""
					content = strings.Replace(content, pattern, replacement, 1)
					changed = true
				}
			}
		}
		if changed {
			os.WriteFile(path, []byte(content), 0o644)
		}
		return nil
	})
}

func (r *Runner) applyReformat(rf Action) error {
	if len(rf.NoteIDs) == 0 || rf.MergedBody == "" {
		return fmt.Errorf("invalid reformat action")
	}

	path := r.findNoteByID(rf.NoteIDs[0])
	if path == "" {
		return fmt.Errorf("note %s not found", rf.NoteIDs[0])
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	if strings.HasPrefix(content, "---\n") {
		if idx := strings.Index(content[4:], "\n---\n"); idx >= 0 {
			fm := content[:idx+4+5] // frontmatter including closing ---
			newContent := fm + "\n" + rf.MergedBody + "\n"
			return os.WriteFile(path, []byte(newContent), 0o644)
		}
	}

	return os.WriteFile(path, []byte(rf.MergedBody), 0o644)
}

func (r *Runner) findNoteByID(id string) string {
	var found string
	filepath.Walk(filepath.Join(r.VaultPath, "knowledge"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, id) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func (r *Runner) nextBranch() string {
	date := time.Now().Format("2006-01-02")
	base := "dream/" + date

	// List existing branches matching this date.
	cmd := exec.Command("git", "branch", "-a", "--list", base+"*")
	cmd.Dir = r.VaultPath
	out, _ := cmd.Output()

	if len(strings.TrimSpace(string(out))) == 0 {
		return base
	}

	// Count existing branches for this date and increment.
	n := 0
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "remotes/origin/")
		if line == base || strings.HasPrefix(line, base+"-") {
			n++
		}
	}
	if n == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, n+1)
}

func (r *Runner) hasBranchCommits(branch string) bool {
	cmd := exec.Command("git", "log", "main.."+branch, "--oneline")
	cmd.Dir = r.VaultPath
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

func (r *Runner) git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.VaultPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *Runner) createPR(report *Report, branch string) {
	body := fmt.Sprintf("## Ngram Dream Cycle — %s\n\n", report.Date)

	if len(report.Merges) > 0 {
		body += fmt.Sprintf("### Merges (%d)\n", len(report.Merges))
		for _, m := range report.Merges {
			body += fmt.Sprintf("- %s — %s\n", strings.Join(m.NoteIDs, " + "), m.Reason)
		}
		body += "\n"
	}

	if len(report.Archives) > 0 {
		body += fmt.Sprintf("### Archives (%d)\n", len(report.Archives))
		for _, a := range report.Archives {
			body += fmt.Sprintf("- %s — %s\n", strings.Join(a.NoteIDs, ", "), a.Reason)
		}
		body += "\n"
	}

	if len(report.Reclusters) > 0 {
		body += fmt.Sprintf("### Taxonomy (%d)\n", len(report.Reclusters))
		for _, rc := range report.Reclusters {
			body += fmt.Sprintf("- → %s — %s\n", rc.NewCluster, rc.Reason)
		}
		body += "\n"
	}

	if len(report.Reformats) > 0 {
		body += fmt.Sprintf("### Reformats (%d)\n", len(report.Reformats))
		for _, rf := range report.Reformats {
			body += fmt.Sprintf("- %s — %s\n", strings.Join(rf.NoteIDs, ", "), rf.Reason)
		}
		body += "\n"
	}

	body += fmt.Sprintf("### No Action (%d notes reviewed)\n", report.NoAction)

	title := fmt.Sprintf("dream: nightly consolidation %s", report.Date)

	ghBin, err := exec.LookPath("gh")
	if err != nil {
		// Homebrew on macOS may not be in PATH for non-login shells.
		for _, candidate := range []string{"/opt/homebrew/bin/gh", "/usr/local/bin/gh"} {
			if _, serr := os.Stat(candidate); serr == nil {
				ghBin = candidate
				break
			}
		}
		if ghBin == "" {
			log.Printf("dream: gh not found in PATH, skipping PR creation")
			return
		}
	}

	cmd := exec.Command(ghBin, "pr", "create", "--title", title, "--body", body, "--base", "main", "--head", branch)
	cmd.Dir = r.VaultPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("dream: gh pr create failed: %v (branch %s still exists)", err, branch)
	}
}
