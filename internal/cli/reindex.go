package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/search"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Rebuild the Meilisearch index from vault files",
	RunE:  reindexRun,
}

func reindexRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	client, err := search.New(c.Meilisearch.Host, c.Meilisearch.APIKey)
	if err != nil {
		return fmt.Errorf("connect to meilisearch: %w", err)
	}

	if err := client.EnsureIndex(); err != nil {
		return fmt.Errorf("ensure notes index: %w", err)
	}
	if err := client.EnsureCommandsIndex(); err != nil {
		return fmt.Errorf("ensure commands index: %w", err)
	}

	files, err := search.WalkVault(c.VaultPath)
	if err != nil {
		return fmt.Errorf("walk vault: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("no notes found to index")
		return nil
	}

	start := time.Now()
	var docs []search.NoteDocument
	var cmds []search.CommandDocument
	var skipped int

	for _, f := range files {
		doc, err := search.ParseNoteFile(f, c.VaultPath)
		if err != nil {
			skipped++
			continue
		}
		docs = append(docs, *doc)

		// Extract commands from code blocks.
		extracted := search.ExtractCommands(doc.ID, doc.Title, doc.Body, search.CommandMeta{
			Phase:    doc.Phase,
			Domain:   doc.Domain,
			Tags:     doc.Tags,
			FilePath: doc.FilePath,
		})
		cmds = append(cmds, extracted...)
	}

	// Index notes in batches of 100.
	batchSize := 100
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		if err := client.IndexNotes(docs[i:end]); err != nil {
			return fmt.Errorf("index notes batch: %w", err)
		}
	}

	// Clear and rebuild commands index.
	if len(cmds) > 0 {
		client.ClearCommandsIndex()
		for i := 0; i < len(cmds); i += batchSize {
			end := i + batchSize
			if end > len(cmds) {
				end = len(cmds)
			}
			if err := client.IndexCommands(cmds[i:end]); err != nil {
				return fmt.Errorf("index commands batch: %w", err)
			}
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("indexed %d notes, %d commands (%s)", len(docs), len(cmds), elapsed.Round(time.Millisecond))
	if skipped > 0 {
		fmt.Printf(", %d skipped", skipped)
	}
	fmt.Println()
	return nil
}
