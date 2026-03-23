package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tylercooper/ngram/internal/daemon"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show service health and processing backlog",
	RunE:  statusRun,
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output as JSON")
}

func statusRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	running, hb := daemon.IsRunning(c.VaultPath)

	if !running {
		if hb != nil {
			fmt.Println("[ngram] daemon not running (stale heartbeat)")
		} else {
			fmt.Println("[ngram] daemon not running")
		}
	} else {
		fmt.Printf("[ngram] daemon running (PID %d)\n", hb.PID)

		if hb.Engaged {
			fmt.Printf("  ENGAGEMENT MODE: %s (quizzes paused)\n", hb.EngagementName)
		}

		fmt.Println()
		for name, status := range hb.Goroutines {
			marker := "✓"
			if status != "healthy" {
				marker = "✗"
			}
			fmt.Printf("  %s %-20s %s\n", marker, name, status)
		}
	}

	// Processing backlog.
	fmt.Println()
	inboxCount := countFiles(filepath.Join(c.VaultPath, "_inbox"))
	procCount := countFiles(filepath.Join(c.VaultPath, "_processing"))

	if inboxCount > 0 || procCount > 0 {
		fmt.Printf("  _inbox/:      %d pending\n", inboxCount)
		fmt.Printf("  _processing/: %d in flight\n", procCount)
	} else {
		fmt.Println("  queue: empty (all notes processed)")
	}

	return nil
}

func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			count++
		}
	}
	return count
}
