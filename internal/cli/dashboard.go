package cli

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/config"
	"github.com/ty-cooper/ngram/internal/dashboard"
	"github.com/ty-cooper/ngram/internal/search"
)

var dashCmd = &cobra.Command{
	Use:     "dash [box]",
	Aliases: []string{"dashboard"},
	Short:   "Live engagement dashboard",
	Args:    cobra.MaximumNArgs(1),
	RunE:    dashRun,
}

func init() {
	rootCmd.AddCommand(dashCmd)
}

func dashRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	var boxName string
	if len(args) > 0 {
		boxName = args[0]
	} else {
		cwd, _ := os.Getwd()
		boxCtx, _ := config.FindBoxRC(cwd)
		if boxCtx != nil {
			boxName = boxCtx.Box
		}
	}

	if boxName == "" {
		return fmt.Errorf("no box specified — use 'n dash <box>' or run from a box directory")
	}

	// Connect to search (optional).
	var client *search.Client
	sc, err := search.New(c.Meilisearch.Host, c.Meilisearch.APIKey)
	if err == nil {
		client = sc
	}

	model := dashboard.New(c.VaultPath, boxName, client)
	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}
