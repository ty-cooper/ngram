package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/search"
)

var cmdCmd = &cobra.Command{
	Use:   "cmd [query]",
	Short: "Search commands by tool, phase, or keyword",
	Long: `Search the commands index for specific code blocks.

Examples:
  n cmd --tool nmap                    # all nmap commands
  n cmd --tool mimikatz --phase post   # mimikatz post-exploitation
  n cmd SYN scan                       # text search for SYN scan
  n cmd --tool nmap SYN                # nmap commands mentioning SYN
  n cmd --facets                       # show all available filters`,
	RunE: cmdRun,
}

func init() {
	cmdCmd.Flags().String("tool", "", "filter by tool (e.g. nmap, mimikatz)")
	cmdCmd.Flags().String("phase", "", "filter by phase (e.g. recon, post)")
	cmdCmd.Flags().String("domain", "", "filter by domain")
	cmdCmd.Flags().String("tag", "", "filter by tag")
	cmdCmd.Flags().Bool("facets", false, "show available filter values")
	cmdCmd.Flags().Int("limit", 20, "max results")
}

func cmdRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	client, err := search.New(c.Meilisearch.Host, c.Meilisearch.APIKey)
	if err != nil {
		return fmt.Errorf("connect to meilisearch: %w", err)
	}

	showFacets, _ := cmd.Flags().GetBool("facets")
	if showFacets {
		return printFacets(client)
	}

	// Build filters.
	var filters []string
	if tool, _ := cmd.Flags().GetString("tool"); tool != "" {
		filters = append(filters, fmt.Sprintf("tool = \"%s\"", tool))
	}
	if phase, _ := cmd.Flags().GetString("phase"); phase != "" {
		filters = append(filters, fmt.Sprintf("phase = \"%s\"", phase))
	}
	if domain, _ := cmd.Flags().GetString("domain"); domain != "" {
		filters = append(filters, fmt.Sprintf("domain = \"%s\"", domain))
	}
	if tag, _ := cmd.Flags().GetString("tag"); tag != "" {
		filters = append(filters, fmt.Sprintf("tags = \"%s\"", tag))
	}

	query := strings.Join(args, " ")
	limit, _ := cmd.Flags().GetInt("limit")

	results, err := client.SearchCommands(query, filters, limit)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("no commands found")
		return nil
	}

	for i, r := range results {
		if i > 0 {
			fmt.Println("---")
		}
		fmt.Printf("[%s] %s\n", r.Tool, r.Description)
		if r.Language != "" {
			fmt.Printf("```%s\n%s\n```\n", r.Language, r.Command)
		} else {
			fmt.Printf("```\n%s\n```\n", r.Command)
		}
		fmt.Printf("  from: %s\n", r.ParentTitle)
	}

	return nil
}

func printFacets(client *search.Client) error {
	facets, err := client.CommandFacets()
	if err != nil {
		return err
	}

	fmt.Println("Available filters for 'n cmd':\n")

	order := []string{"tool", "phase", "domain", "tags", "language"}
	for _, field := range order {
		vals := facets[field]
		if len(vals) == 0 {
			continue
		}
		sort.Strings(vals)
		fmt.Printf("  --%s:  %s\n", field, strings.Join(vals, "  "))
	}

	noteFacets, err := client.NotesFacets()
	if err == nil && len(noteFacets) > 0 {
		fmt.Println("\nNote filters (for 'n ask'):\n")
		for _, field := range []string{"domain", "phase", "tags", "content_type"} {
			vals := noteFacets[field]
			if len(vals) == 0 {
				continue
			}
			sort.Strings(vals)
			fmt.Printf("  %s:  %s\n", field, strings.Join(vals, "  "))
		}
	}

	fmt.Println("\nUsage:")
	fmt.Println("  n cmd --tool nmap              all nmap commands")
	fmt.Println("  n cmd --tool mimikatz --phase post   mimikatz post-exploitation")
	fmt.Println("  n cmd SYN scan                 text search across all commands")
	fmt.Println("  n cmd --tool nmap SYN          nmap commands mentioning SYN")

	return nil
}
