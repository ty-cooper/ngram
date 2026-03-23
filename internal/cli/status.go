package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tylercooper/ngram/internal/daemon"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show service health",
	RunE:  statusRun,
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
		return nil
	}

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

	return nil
}
