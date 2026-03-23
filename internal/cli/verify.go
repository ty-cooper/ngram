package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/vault"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify hash chain integrity of vault notes",
	RunE:  verifyRun,
}

func verifyRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	errors, err := vault.VerifyChain(c.VaultPath)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}

	if len(errors) == 0 {
		fmt.Println("✓ hash chain intact — all notes verified")
		return nil
	}

	fmt.Printf("✗ %d integrity errors found:\n\n", len(errors))
	for _, e := range errors {
		fmt.Printf("  %s\n", e)
	}
	return nil
}
