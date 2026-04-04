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

	start := time.Now()
	notes, cmds, err := client.FullReindex(c.VaultPath)
	if err != nil {
		return err
	}

	elapsed := time.Since(start)
	fmt.Printf("indexed %d notes, %d commands (%s)\n", notes, cmds, elapsed.Round(time.Millisecond))
	return nil
}
