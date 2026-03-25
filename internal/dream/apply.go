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
	if len(report.Merges) == 0 && len(report.Archives) == 0 && len(report.Reclusters) == 0 && len(report.Retags) == 0 {
		log.Println("dream: vault is clean, nothing to do")
		return nil
	}

	branch := fmt.Sprintf("dream/%s", time.Now().Format("2006-01-02"))

	// Create or switch to branch in the vault repo.
	if err := r.git("checkout", "-b", branch); err != nil {
		// Branch may already exist from a prior partial run.
		if err := r.git("checkout", branch); err != nil {
			return fmt.Errorf("checkout branch: %w", err)
		}
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
				// Move secondary notes to _trash/.
				trashDir := filepath.Join(r.VaultPath, "_trash")
				os.MkdirAll(trashDir, 0o755)
				dest := filepath.Join(trashDir, filepath.Base(path))
				os.Rename(path, dest)
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
	trashDir := filepath.Join(r.VaultPath, "_trash")
	os.MkdirAll(trashDir, 0o755)

	for _, id := range archive.NoteIDs {
		path := r.findNoteByID(id)
		if path != "" {
			dest := filepath.Join(trashDir, filepath.Base(path))
			if err := os.Rename(path, dest); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) applyRecluster(rc Action) error {
	// Walk all notes, find ones in the old cluster, update frontmatter.
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
		// Check if this note matches the old cluster in the reason field.
		if rc.NewCluster != "" && strings.Contains(content, "topic_cluster:") {
			// Simple string replacement for near-synonym clusters.
			for _, oldCluster := range strings.Split(rc.Reason, " → ") {
				oldCluster = strings.TrimSpace(oldCluster)
				if oldCluster != "" && oldCluster != rc.NewCluster {
					content = strings.Replace(content, "topic_cluster: \""+oldCluster+"\"", "topic_cluster: \""+rc.NewCluster+"\"", 1)
					content = strings.Replace(content, "topic_cluster: "+oldCluster, "topic_cluster: "+rc.NewCluster, 1)
				}
			}
			os.WriteFile(path, []byte(content), 0o644)
		}
		return nil
	})
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
