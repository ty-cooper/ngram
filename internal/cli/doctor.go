package cli

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/daemon"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health",
	RunE:  doctorRun,
}

func doctorRun(cmd *cobra.Command, args []string) error {
	ok := true

	// 1. Config.
	c, err := loadConfig()
	if err != nil {
		check(false, "config", fmt.Sprintf("~/.ngram.yml: %v", err))
		return nil
	}
	check(true, "config", "~/.ngram.yml")

	// 2. Config validation.
	issues := c.Validate()
	for _, issue := range issues {
		check(false, "config", issue)
		ok = false
	}

	// 3. Vault exists.
	if _, err := os.Stat(c.VaultPath); err == nil {
		check(true, "vault", c.VaultPath)
	} else {
		check(false, "vault", fmt.Sprintf("%s does not exist — run 'n setup'", c.VaultPath))
		ok = false
	}

	// 4. Vault structure.
	for _, dir := range []string{"_inbox", "_meta", "knowledge"} {
		p := filepath.Join(c.VaultPath, dir)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			check(false, "vault", fmt.Sprintf("missing %s/ — run 'n init'", dir))
			ok = false
		}
	}

	// 5. Git repo.
	gitDir := filepath.Join(c.VaultPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		check(true, "git", "vault is a git repo")
	} else {
		check(false, "git", "vault is not a git repo — run 'git init' in vault")
		ok = false
	}

	// 6. Docker.
	if err := exec.Command("docker", "info").Run(); err != nil {
		check(false, "docker", "not running — start Docker Desktop")
		ok = false
	} else {
		check(true, "docker", "running")
	}

	// 7. Meilisearch.
	msHealthy := false
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(c.Meilisearch.Host + "/health")
	if err == nil && resp.StatusCode == 200 {
		msHealthy = true
		resp.Body.Close()
		check(true, "meilisearch", c.Meilisearch.Host)
	} else {
		check(false, "meilisearch", fmt.Sprintf("%s not responding — run 'n up'", c.Meilisearch.Host))
		ok = false
	}
	_ = msHealthy

	// 8. Anthropic API key.
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		check(true, "anthropic", "ANTHROPIC_API_KEY set")
	} else {
		check(false, "anthropic", "ANTHROPIC_API_KEY not set — get one at https://console.anthropic.com")
		ok = false
	}

	// 9. Optional: OpenAI (embeddings).
	if os.Getenv("OPENAI_API_KEY") != "" || c.Embeddings.OpenAIAPIKey != "" {
		check(true, "embeddings", "OpenAI API key set")
	} else {
		warn("embeddings", "OPENAI_API_KEY not set — hybrid search disabled")
	}

	// 10. Optional: LangSmith (tracing).
	if os.Getenv("LANGCHAIN_API_KEY") != "" {
		check(true, "tracing", "LangSmith API key set")
	} else {
		warn("tracing", "LANGCHAIN_API_KEY not set — tracing disabled")
	}

	// 11. Daemon.
	if running, _ := daemon.IsRunning(c.VaultPath); running {
		check(true, "daemon", "running")
	} else {
		check(false, "daemon", "not running — run 'n up'")
		ok = false
	}

	fmt.Println()
	if ok {
		fmt.Println("✓ all checks passed")
	} else {
		fmt.Println("✗ some checks failed — fix the issues above")
	}
	return nil
}

func check(ok bool, component, msg string) {
	if ok {
		fmt.Printf("  ✓ %-14s %s\n", component, msg)
	} else {
		fmt.Printf("  ✗ %-14s %s\n", component, msg)
	}
}

func warn(component, msg string) {
	fmt.Printf("  ~ %-14s %s\n", component, msg)
}
