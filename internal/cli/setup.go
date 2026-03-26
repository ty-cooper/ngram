package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"os/exec"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/config"
	"github.com/ty-cooper/ngram/internal/daemon"
	"github.com/ty-cooper/ngram/internal/vault"
)

func cmdExec(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "One-command interactive setup",
	Long:  "Configure vault, write config, initialize directories, and optionally install as a service.",
	RunE:  setupRun,
}

func setupRun(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	home, _ := os.UserHomeDir()
	defaultVault := filepath.Join(home, ".obsidian.ngram")

	// 1. Vault path.
	fmt.Printf("vault path [%s]: ", defaultVault)
	vaultPath, _ := reader.ReadString('\n')
	vaultPath = strings.TrimSpace(vaultPath)
	if vaultPath == "" {
		vaultPath = defaultVault
	}
	vaultPath = config.ExpandHome(vaultPath)

	// 2. Check API key.
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("\n  ANTHROPIC_API_KEY not set.")
		fmt.Println("  Get one at https://console.anthropic.com")
		fmt.Println("  Add to ~/.zshrc: export ANTHROPIC_API_KEY=sk-ant-...")
		fmt.Println("  Continuing setup — AI processing will be disabled until key is set.")
		fmt.Println()
	} else {
		fmt.Println("  ANTHROPIC_API_KEY ✓")
	}

	// 3. Write config.
	cfgPath := filepath.Join(home, ".ngram.yml")
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Printf("config already exists at %s, overwrite? [y/N]: ", cfgPath)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			fmt.Println("  keeping existing config")
			goto initVault
		}
	}

	{
		content := fmt.Sprintf("vault_path: %s\nmodel: cloud\n", vaultPath)
		if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		fmt.Printf("  wrote %s ✓\n", cfgPath)
	}

initVault:
	// 4. Initialize vault.
	if err := vault.Init(vaultPath); err != nil {
		return fmt.Errorf("init vault: %w", err)
	}
	fmt.Printf("  vault initialized at %s ✓\n", vaultPath)

	// 5. Init git if not already a repo.
	if _, err := os.Stat(filepath.Join(vaultPath, ".git")); os.IsNotExist(err) {
		fmt.Println("  initializing git repo...")
		cmd := cmdExec("git", "init", vaultPath)
		cmd.Run()
		fmt.Println("  git ✓")
	}

	// 6. Install as service?
	fmt.Print("install as login item (auto-start on boot)? [Y/n]: ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "" || answer == "y" || answer == "yes" {
		binary, _ := os.Executable()
		if err := installService(binary, vaultPath); err != nil {
			fmt.Printf("  warn: install failed: %v\n", err)
		} else {
			fmt.Println("  service installed ✓")
		}
	}

	// 7. Summary.
	fmt.Println("\n✓ ngram setup complete")
	fmt.Printf("  vault:  %s\n", vaultPath)
	fmt.Printf("  config: %s\n", cfgPath)
	fmt.Println("\n  next steps:")
	fmt.Println("    n up          — start the daemon")
	fmt.Println("    n new <text>  — capture a note")
	fmt.Println("    n doctor      — check system health")

	return nil
}

func installService(binary, vaultPath string) error {
	return daemon.Install(binary, vaultPath)
}
