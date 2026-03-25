package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/config"
	"github.com/ty-cooper/ngram/internal/pipeline"
	"github.com/ty-cooper/ngram/internal/search"
)

var recallAll bool

var recallCmd = &cobra.Command{
	Use:   "recall [query...]",
	Short: "Search across engagements for related knowledge",
	Args:  cobra.MinimumNArgs(1),
	RunE:  recallRun,
}

func init() {
	recallCmd.Flags().BoolVar(&recallAll, "all", false, "include current engagement in results")
	rootCmd.AddCommand(recallCmd)
}

func recallRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	client, err := search.New(c.Meilisearch.Host, c.Meilisearch.APIKey)
	if err != nil {
		return fmt.Errorf("search unavailable: %w", err)
	}

	query := strings.Join(args, " ")

	// Detect current box from .boxrc.
	var excludeBox string
	if !recallAll {
		cwd, _ := cmd.Flags().GetString("cwd")
		if cwd == "" {
			cwd = "."
		}
		boxCtx, _ := config.FindBoxRC(cwd)
		if boxCtx != nil {
			excludeBox = boxCtx.Box
		}
	}

	results, err := pipeline.RecallSearch(client, query, excludeBox, 10)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("No related knowledge found.")
		return nil
	}

	for i, r := range results {
		ctx := r.Box
		if ctx == "" {
			ctx = "general"
		}
		summary := r.Summary
		if summary == "" {
			summary = "(no summary)"
		}
		fmt.Printf("[%d] %s (%s) — %.0f%%\n", i+1, r.Title, ctx, r.Score*100)
		fmt.Printf("    %s\n", summary)
		fmt.Printf("    %s\n\n", r.FilePath)
	}

	return nil
}
