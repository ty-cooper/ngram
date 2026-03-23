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
		return fmt.Errorf("ensure index: %w", err)
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
	var skipped int

	for _, f := range files {
		doc, err := search.ParseNoteFile(f, c.VaultPath)
		if err != nil {
			skipped++
			continue
		}
		docs = append(docs, *doc)
	}

	// Batch in chunks of 100.
	batchSize := 100
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		if err := client.IndexNotes(docs[i:end]); err != nil {
			return fmt.Errorf("index batch: %w", err)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("indexed %d notes (%s)", len(docs), elapsed.Round(time.Millisecond))
	if skipped > 0 {
		fmt.Printf(", %d skipped", skipped)
	}
	fmt.Println()
	return nil
}
