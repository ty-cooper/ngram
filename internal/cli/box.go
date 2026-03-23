package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tylercooper/ngram/internal/vault"
)

var flagOS string

var boxCmd = &cobra.Command{
	Use:   "box <name> <ip>",
	Short: "Scaffold a new target box",
	Long:  "Creates the full folder structure and .boxrc for a new engagement target.",
	Args:  cobra.ExactArgs(2),
	RunE:  boxRun,
}

func init() {
	boxCmd.Flags().StringVar(&flagOS, "os", "", "target operating system (e.g. windows, linux)")
}

func boxRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	name := args[0]
	ip := args[1]

	boxDir := vault.BoxDir(c.VaultPath, name)
	phases := []string{"_recon", "_enum", "_exploit", "_post", "_loot"}

	for _, phase := range phases {
		if err := vault.EnsureDir(filepath.Join(boxDir, phase)); err != nil {
			return fmt.Errorf("create %s: %w", phase, err)
		}
	}

	boxrc := fmt.Sprintf("BOX=%s\nIP=%s\nPHASE=recon\n", name, ip)
	if flagOS != "" {
		boxrc += fmt.Sprintf("OS=%s\n", flagOS)
	}

	rcPath := filepath.Join(boxDir, ".boxrc")
	if err := os.WriteFile(rcPath, []byte(boxrc), 0o644); err != nil {
		return fmt.Errorf("write .boxrc: %w", err)
	}

	fmt.Printf("✓ boxes/%s/ created [ip: %s]\n", name, ip)
	return nil
}
