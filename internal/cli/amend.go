package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var amendCmd = &cobra.Command{
	Use:   "amend [text...]",
	Short: "Append text to the most recently captured note",
	Args:  cobra.MinimumNArgs(1),
	RunE:  amendRun,
}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the most recently captured note in $EDITOR",
	RunE:  editRun,
}

func amendRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	path, err := findLastNote(c.VaultPath)
	if err != nil {
		return err
	}

	text := strings.Join(args, " ")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open note: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "\n%s\n", text); err != nil {
		return fmt.Errorf("append: %w", err)
	}

	rel, _ := filepath.Rel(c.VaultPath, path)
	fmt.Printf("✓ amended %s\n", rel)
	return nil
}

func editRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	path, err := findLastNote(c.VaultPath)
	if err != nil {
		return err
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	editorCmd := exec.Command(editor, path)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	return editorCmd.Run()
}

// findLastNote returns the most recently modified .md file in _inbox/ or _processing/.
// Falls back to searching the entire vault for the newest note.
func findLastNote(vaultPath string) (string, error) {
	// Check _inbox/ first (not yet processed).
	dirs := []string{
		filepath.Join(vaultPath, "_inbox"),
		filepath.Join(vaultPath, "_processing"),
	}

	for _, dir := range dirs {
		path, err := newestMD(dir)
		if err == nil {
			return path, nil
		}
	}

	// Fall back to searching knowledge/ and boxes/.
	searchDirs := []string{
		filepath.Join(vaultPath, "knowledge"),
		filepath.Join(vaultPath, "boxes"),
	}

	var newest string
	var newestTime int64

	for _, dir := range searchDirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".md") && info.ModTime().Unix() > newestTime {
				newest = path
				newestTime = info.ModTime().Unix()
			}
			return nil
		})
	}

	if newest == "" {
		return "", fmt.Errorf("no recent notes found")
	}
	return newest, nil
}

func newestMD(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var mds []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mds = append(mds, e)
		}
	}

	if len(mds) == 0 {
		return "", fmt.Errorf("no .md files")
	}

	sort.Slice(mds, func(i, j int) bool {
		ii, _ := mds[i].Info()
		jj, _ := mds[j].Info()
		return ii.ModTime().After(jj.ModTime())
	})

	return filepath.Join(dir, mds[0].Name()), nil
}
