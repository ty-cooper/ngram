package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/vault"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize vault directory structure",
	Long:  "Creates all required directories and seed files in the vault.",
	RunE:  initRun,
}

func initRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	info, statErr := os.Stat(c.VaultPath)

	if statErr != nil && os.IsNotExist(statErr) {
		// Path doesn't exist — create it.
		fmt.Printf("creating vault at %s\n", c.VaultPath)
		if err := os.MkdirAll(c.VaultPath, 0o755); err != nil {
			return fmt.Errorf("create vault path: %w", err)
		}
	} else if statErr != nil {
		return fmt.Errorf("stat vault path: %w", statErr)
	} else if info.IsDir() {
		// Path exists. Check if it already has vault structure.
		if hasVaultStructure(c.VaultPath) {
			fmt.Printf("vault already exists at %s\n", c.VaultPath)
			fmt.Print("reinitialize? this overwrites seed files (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("aborted")
				return nil
			}
		}
	}

	if err := vault.Init(c.VaultPath); err != nil {
		return fmt.Errorf("init vault: %w", err)
	}

	fmt.Printf("✓ vault initialized at %s\n", c.VaultPath)
	return nil
}

func hasVaultStructure(path string) bool {
	markers := []string{"_inbox", "_meta", "knowledge"}
	for _, m := range markers {
		if info, err := os.Stat(path + "/" + m); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}
