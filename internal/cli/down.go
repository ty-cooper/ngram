package cli

import (
	"fmt"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tylercooper/ngram/internal/daemon"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop all services",
	RunE:  downRun,
}

func downRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	running, hb := daemon.IsRunning(c.VaultPath)
	if !running || hb == nil {
		return fmt.Errorf("daemon is not running")
	}

	// Send SIGTERM to the daemon process.
	if err := syscall.Kill(hb.PID, syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM to PID %d: %w", hb.PID, err)
	}

	fmt.Printf("✓ sent SIGTERM to PID %d\n", hb.PID)

	// Stop Meilisearch.
	fmt.Println("stopping meilisearch...")
	if err := daemon.StopMeilisearch(c.VaultPath); err != nil {
		fmt.Printf("warn: meilisearch: %v\n", err)
	}

	fmt.Println("✓ all services stopped")
	return nil
}
