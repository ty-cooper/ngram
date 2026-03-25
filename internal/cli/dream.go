package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/dream"
	"github.com/ty-cooper/ngram/internal/llm"
	"github.com/ty-cooper/ngram/internal/search"
)

var flagDryRun bool

var dreamCmd = &cobra.Command{
	Use:   "dream",
	Short: "Nightly knowledge sweep — dedup, prune, recluster, or do nothing",
	Long: `Scans the entire vault for duplicate notes, empty notes, near-synonym
clusters, and quality issues. Creates a PR against the vault repo with
proposed changes. Each change is a separate commit so you can cherry-pick.

Use --dry-run to see what would change without modifying anything.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		vaultPath := cfg.VaultPath

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		searchClient, err := search.New(cfg.Meilisearch.Host, cfg.Meilisearch.APIKey)
		if err != nil {
			return fmt.Errorf("connect to meilisearch: %w", err)
		}
		if !searchClient.Healthy() {
			return fmt.Errorf("meilisearch is not running — start it with 'n up' first")
		}
		if embCfg := buildEmbedderConfig(cfg); embCfg.Source != "" {
			if err := searchClient.ConfigureEmbedder(embCfg); err != nil {
				log.Printf("dream: embedder config failed: %v (using keyword search)", err)
			}
		}

		runner := &dream.Runner{
			VaultPath: vaultPath,
			Search:    searchClient,
			LLM:       llm.NewRunner(cfg.Model, vaultPath),
		}

		log.Println("dream: starting knowledge sweep...")
		report, err := runner.Run(ctx)
		if err != nil {
			return fmt.Errorf("dream: %w", err)
		}

		// Print report.
		reportJSON, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(reportJSON))

		if flagDryRun {
			fmt.Println("\n(dry run — no changes applied)")
			return nil
		}

		// Apply changes on a branch and create PR.
		if err := runner.Apply(report); err != nil {
			return fmt.Errorf("apply: %w", err)
		}

		return nil
	},
}

func init() {
	dreamCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "show what would change without modifying anything")
	rootCmd.AddCommand(dreamCmd)
}
