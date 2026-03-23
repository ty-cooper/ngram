package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tylercooper/ngram/internal/llm"
	"github.com/tylercooper/ngram/internal/rag"
	"github.com/tylercooper/ngram/internal/search"
)

var (
	askDomain      string
	askSourcesOnly bool
	askVerbose     bool
)

var askCmd = &cobra.Command{
	Use:   "ask [question...]",
	Short: "Ask a question — RAG synthesis from your vault",
	Long:  "Searches the vault and synthesizes an answer with inline [noteID] citations.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  askRun,
}

func init() {
	askCmd.Flags().StringVar(&askDomain, "domain", "", "restrict search to domain")
	askCmd.Flags().BoolVar(&askSourcesOnly, "sources-only", false, "show matching notes only, no LLM synthesis")
	askCmd.Flags().BoolVar(&askVerbose, "verbose", false, "show retrieved notes")
}

func askRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	client, err := search.New(c.Meilisearch.Host, c.Meilisearch.APIKey)
	if err != nil {
		return fmt.Errorf("connect to meilisearch: %w", err)
	}

	runner := &llm.Runner{
		BinaryPath: "claude",
		Model:      c.Model,
		VaultPath:  c.VaultPath,
	}

	engine := &rag.Engine{
		SearchClient: client,
		Runner:       runner,
		VaultPath:    c.VaultPath,
	}

	question := strings.Join(args, " ")
	result, err := engine.Ask(cmd.Context(), question, rag.AskOptions{
		Domain:      askDomain,
		SourcesOnly: askSourcesOnly,
		Verbose:     askVerbose,
	})
	if err != nil {
		return fmt.Errorf("ask: %w", err)
	}

	if result.GapDetected {
		fmt.Println("No relevant notes found. Gap logged to _meta/knowledge-gaps.jsonl.")
		return nil
	}

	if askSourcesOnly || result.Answer == "" {
		for i, s := range result.Sources {
			fmt.Printf("[%d] [%s] %s\n", i+1, s.ID, s.Title)
			fmt.Printf("    %s\n\n", s.FilePath)
		}
		return nil
	}

	fmt.Println(result.Answer)

	if askVerbose && len(result.Sources) > 0 {
		fmt.Println("\n---")
		fmt.Println("Sources:")
		for _, s := range result.Sources {
			fmt.Printf("  [%s] %s (%s)\n", s.ID, s.Title, s.FilePath)
		}
	}

	return nil
}
