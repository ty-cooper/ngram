package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/config"
	"github.com/ty-cooper/ngram/internal/daemon"
	"github.com/ty-cooper/ngram/internal/llm"
	"github.com/ty-cooper/ngram/internal/pipeline"
	"github.com/ty-cooper/ngram/internal/search"
	"github.com/ty-cooper/ngram/internal/taxonomy"
)

var (
	upForeground bool
	upInstall    bool
	upUninstall  bool
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start all services",
	RunE:  upRun,
}

func init() {
	upCmd.Flags().BoolVar(&upForeground, "foreground", false, "run in foreground (for launchd/systemd)")
	upCmd.Flags().BoolVar(&upInstall, "install", false, "install as system service and start")
	upCmd.Flags().BoolVar(&upUninstall, "uninstall", false, "remove system service")
}

func upRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	if upUninstall {
		if err := daemon.Uninstall(); err != nil {
			return fmt.Errorf("uninstall: %w", err)
		}
		fmt.Println("✓ service uninstalled")
		return nil
	}

	if upInstall {
		binary, _ := os.Executable()
		if err := daemon.Install(binary, c.VaultPath); err != nil {
			return fmt.Errorf("install: %w", err)
		}
		fmt.Println("✓ service installed and started")
		return nil
	}

	// Check if already running — exit 0 so launchd doesn't restart loop.
	if running, _ := daemon.IsRunning(c.VaultPath); running {
		fmt.Println("ngram is already running. Use 'n down' to stop, or 'n status' to check health.")
		return nil
	}

	// Startup summary.
	fmt.Println("ngram starting...")
	fmt.Printf("  %-14s %s\n", "vault:", c.VaultPath)

	// Start Meilisearch.
	msOK := false
	if err := daemon.StartMeilisearch(c.VaultPath); err != nil {
		fmt.Printf("  %-14s ✗ %v\n", "meilisearch:", err)
	} else {
		fmt.Printf("  %-14s %s ✓\n", "meilisearch:", c.Meilisearch.Host)
		msOK = true
	}

	// Build services.
	tax, _ := taxonomy.Load(c.VaultPath)

	var searchClient *search.Client
	if msOK {
		sc, err := search.New(c.Meilisearch.Host, c.Meilisearch.APIKey)
		if err == nil && sc.Healthy() {
			sc.EnsureIndex()
			if embCfg := buildEmbedderConfig(c); embCfg.Source != "" {
				if err := sc.ConfigureEmbedder(embCfg); err != nil {
					fmt.Printf("  %-14s ✗ %v\n", "embeddings:", err)
				} else {
					fmt.Printf("  %-14s enabled (OpenAI) ✓\n", "embeddings:")
				}
			} else {
				fmt.Printf("  %-14s disabled (no OPENAI_API_KEY)\n", "embeddings:")
			}
			searchClient = sc
		}
	}

	// Tracing status.
	if os.Getenv("LANGCHAIN_API_KEY") != "" {
		fmt.Printf("  %-14s enabled (LangSmith) ✓\n", "tracing:")
	}

	runner := llm.NewRunner(c.Model, c.VaultPath)

	// Preflight: check API auth.
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Printf("  %-14s ✗ not set — get one at https://console.anthropic.com\n", "anthropic:")
		fmt.Println("  notes will queue in _inbox/ until key is set")
	} else if err := runner.CheckAuth(cmd.Context()); err != nil {
		fmt.Printf("  %-14s ✗ %v\n", "anthropic:", err)
	} else {
		fmt.Printf("  %-14s authenticated ✓\n", "anthropic:")
	}

	var dedup *pipeline.Deduplicator
	if searchClient != nil {
		dedup = &pipeline.Deduplicator{
			VaultPath:    c.VaultPath,
			SearchClient: searchClient,
			Runner:       runner,
		}
	}

	proc := &pipeline.Processor{
		VaultPath:    c.VaultPath,
		Runner:       runner,
		Taxonomy:     tax,
		SearchClient: searchClient,
		Dedup:        dedup,
		MaxRetries:   2,
	}

	watcher := &pipeline.Watcher{
		VaultPath: c.VaultPath,
		Processor: proc,
	}

	d := daemon.New(c.VaultPath)
	d.Services = []daemon.Service{
		{Name: "processor", Run: watcher.Start},
	}
	d.OnShutdown = append(d.OnShutdown, runner.Close)

	fmt.Printf("  %-14s %d\n", "PID:", os.Getpid())
	fmt.Println("✓ ngram daemon running")

	return d.Run(context.Background())
}

func buildEmbedderConfig(c *config.Config) search.EmbedderConfig {
	switch c.Model {
	case "cloud", "":
		if c.Embeddings.OpenAIAPIKey == "" {
			return search.EmbedderConfig{}
		}
		return search.EmbedderConfig{
			Source: "openAi",
			Model:  "text-embedding-3-small",
			APIKey: c.Embeddings.OpenAIAPIKey,
		}
	default:
		return search.EmbedderConfig{}
	}
}
