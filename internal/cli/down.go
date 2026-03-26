package cli

import (
	"fmt"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/daemon"
)

var downForce bool

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop all services",
	RunE:  downRun,
}

func init() {
	downCmd.Flags().BoolVar(&downForce, "force", false, "SIGKILL after 5s if SIGTERM doesn't work")
}

func downRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	running, hb := daemon.IsRunning(c.VaultPath)
	if !running || hb == nil {
		fmt.Println("daemon is not running")
		// Still try to stop Meilisearch in case it's orphaned.
		daemon.StopMeilisearch(c.VaultPath)
		return nil
	}

	// Send SIGTERM.
	if err := syscall.Kill(hb.PID, syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM to PID %d: %w", hb.PID, err)
	}
	fmt.Printf("✓ sent SIGTERM to PID %d\n", hb.PID)

	// Wait for process to exit (up to 10s).
	for i := 0; i < 20; i++ {
		if err := syscall.Kill(hb.PID, 0); err != nil {
			break // Process is gone.
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Check if still alive.
	if err := syscall.Kill(hb.PID, 0); err == nil {
		if downForce {
			fmt.Println("  daemon still running, sending SIGKILL...")
			syscall.Kill(hb.PID, syscall.SIGKILL)
		} else {
			fmt.Println("  warn: daemon still running after SIGTERM. Use --force to SIGKILL.")
		}
	}

	// Stop Meilisearch.
	fmt.Println("stopping meilisearch...")
	if err := daemon.StopMeilisearch(c.VaultPath); err != nil {
		fmt.Printf("  warn: meilisearch: %v\n", err)
	}

	fmt.Println("✓ all services stopped")
	return nil
}
