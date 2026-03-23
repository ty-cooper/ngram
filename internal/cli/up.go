package cli

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
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

	// Check if already running.
	if running, _ := daemon.IsRunning(c.VaultPath); running {
		return fmt.Errorf("daemon already running (use 'n down' first)")
	}

	// Start Meilisearch.
	fmt.Println("starting meilisearch...")
	if err := daemon.StartMeilisearch(c.VaultPath); err != nil {
		log.Printf("warn: meilisearch: %v (continuing without search)", err)
	}

	// Build services.
	tax, _ := taxonomy.Load(c.VaultPath)

	var searchClient *search.Client
	sc, err := search.New(c.Meilisearch.Host, c.Meilisearch.APIKey)
	if err == nil && sc.Healthy() {
		sc.EnsureIndex()
		searchClient = sc
	}

	runner := llm.NewRunner(c.Model, c.VaultPath)

	// Preflight: check API auth before starting daemon.
	fmt.Print("checking anthropic auth... ")
	if err := runner.CheckAuth(cmd.Context()); err != nil {
		if err == llm.ErrAuthExpired {
			fmt.Println("✗")
			fmt.Printf("\n  %v\n\n", err)
			return err
		}
		fmt.Printf("✗ (%v)\n", err)
		fmt.Println("  continuing with degraded AI (notes will queue in _inbox/)")
	} else {
		fmt.Println("✓")
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

	fmt.Println("✓ ngram daemon starting")

	if !upForeground {
		fmt.Printf("  PID: %d\n", os.Getpid())
		fmt.Println("  use 'n status' to check health")
		fmt.Println("  use 'n down' to stop")
	}

	return d.Run(context.Background())
}
