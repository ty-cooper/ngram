package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tylercooper/ngram/internal/search"
	"github.com/tylercooper/ngram/internal/search/query"
)

var (
	searchLimit  int
	searchJSON   bool
	searchOffset int
)

var searchCmd = &cobra.Command{
	Use:     "search [query...]",
	Aliases: []string{"nq"},
	Short:   "Search the vault",
	Long:    "Plain-language, fuzzy, typo-tolerant search. Supports field:value filters (domain:, box:, phase:, tag:, type:, engagement:).",
	Args:    cobra.MinimumNArgs(1),
	RunE:    searchRun,
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 10, "max results")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON")
	searchCmd.Flags().IntVar(&searchOffset, "offset", 0, "result offset for pagination")
}

func searchRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	client, err := search.New(c.Meilisearch.Host, c.Meilisearch.APIKey)
	if err != nil {
		return fmt.Errorf("connect to meilisearch: %w", err)
	}

	raw := strings.Join(args, " ")
	parsed := query.Parse(raw)

	resp, err := client.Search(parsed.Query, search.SearchOptions{
		Filter: parsed.Filter,
		Limit:  int64(searchLimit),
		Offset: int64(searchOffset),
	})
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if searchJSON {
		return json.NewEncoder(os.Stdout).Encode(resp)
	}

	if len(resp.Results) == 0 {
		fmt.Println("no results")
		return nil
	}

	for i, r := range resp.Results {
		num := searchOffset + i + 1
		badge := ""
		if r.ContentType != "" {
			badge = fmt.Sprintf(" [%s]", r.ContentType)
		}
		fmt.Printf("[%d] %s%s\n", num, r.Title, badge)
		fmt.Printf("    %s\n", r.FilePath)
		if r.Snippet != "" {
			fmt.Printf("    %s\n", r.Snippet)
		}
		fmt.Println()
	}

	fmt.Printf("%d results (%dms)\n", resp.TotalHits, resp.ProcessingMs)
	return nil
}
