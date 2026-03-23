package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/vault"
)

var (
	migrateSource string
	migrateDryRun bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Batch-process an existing vault into _inbox/ for AI structuring",
	Long:  "Copies all .md files from source directory into _inbox/ for processing by the AI pipeline. Originals are never modified.",
	RunE:  migrateRun,
}

func init() {
	migrateCmd.Flags().StringVar(&migrateSource, "source", "", "source directory to migrate from (required)")
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "show what would be migrated without copying")
	migrateCmd.MarkFlagRequired("source")
}

func migrateRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	// Validate source.
	info, err := os.Stat(migrateSource)
	if err != nil {
		return fmt.Errorf("source %q: %w", migrateSource, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source %q is not a directory", migrateSource)
	}

	// Find all .md files in source.
	var files []string
	filepath.Walk(migrateSource, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Skip hidden dirs.
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			files = append(files, path)
		}
		return nil
	})

	if len(files) == 0 {
		fmt.Println("no .md files found in source")
		return nil
	}

	fmt.Printf("found %d .md files in %s\n", len(files), migrateSource)

	if migrateDryRun {
		for _, f := range files {
			rel, _ := filepath.Rel(migrateSource, f)
			fmt.Printf("  would copy: %s\n", rel)
		}
		return nil
	}

	// Ensure inbox exists.
	inboxDir, err := vault.InboxDir(c.VaultPath)
	if err != nil {
		return err
	}

	copied := 0
	skipped := 0

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			skipped++
			continue
		}

		rel, _ := filepath.Rel(migrateSource, f)
		slug := strings.ReplaceAll(rel, string(os.PathSeparator), "-")
		slug = strings.TrimSuffix(slug, ".md")

		// Add frontmatter if missing.
		content := string(data)
		if !strings.HasPrefix(strings.TrimSpace(content), "---") {
			fm := fmt.Sprintf("---\ncaptured: %q\ntitle: %q\nsource: \"migration\"\nmigrated_from: %q\n---\n\n",
				time.Now().UTC().Format(time.RFC3339),
				filepath.Base(strings.TrimSuffix(f, ".md")),
				rel,
			)
			content = fm + content
		}

		destFile := fmt.Sprintf("%d-%s.md", time.Now().UnixNano()/1e6, slug)
		destPath := filepath.Join(inboxDir, destFile)

		if err := os.WriteFile(destPath, []byte(content), 0o644); err != nil {
			skipped++
			continue
		}
		copied++
	}

	fmt.Printf("migrated %d notes to _inbox/ (%d skipped)\n", copied, skipped)
	fmt.Println("run 'n up' to start processing, or notes will process when the daemon is running")
	return nil
}
