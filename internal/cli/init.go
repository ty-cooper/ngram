package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/vault"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize vault directory structure",
	Long:  "Creates all required directories and seed files in the vault. Safe to run on an existing vault.",
	RunE:  initRun,
}

func initRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	if err := vault.Init(c.VaultPath); err != nil {
		return fmt.Errorf("init vault: %w", err)
	}

	fmt.Printf("✓ vault initialized at %s\n", c.VaultPath)
	return nil
}
