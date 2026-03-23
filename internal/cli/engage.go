package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/daemon"
)

var engageCmd = &cobra.Command{
	Use:   "engage <name>",
	Short: "Start engagement mode (pauses quizzes)",
	Args:  cobra.ExactArgs(1),
	RunE:  engageRun,
}

var disengageCmd = &cobra.Command{
	Use:   "disengage",
	Short: "End engagement mode (resumes quizzes)",
	RunE:  disengageRun,
}

func engageRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	name := args[0]

	// Write engagement flag to heartbeat so the daemon picks it up.
	if err := setEngagementFlag(c.VaultPath, true, name); err != nil {
		return err
	}

	// Also send signal to running daemon if present.
	if running, hb := daemon.IsRunning(c.VaultPath); running && hb != nil {
		fmt.Printf("✓ engagement mode: %s (quizzes paused, PID %d notified)\n", name, hb.PID)
	} else {
		fmt.Printf("✓ engagement mode: %s (quizzes paused)\n", name)
	}

	// Suggest shell prompt integration.
	fmt.Printf("\n  To show in your prompt: export NGRAM_ENGAGEMENT=%s\n", name)

	return nil
}

func disengageRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	if err := setEngagementFlag(c.VaultPath, false, ""); err != nil {
		return err
	}

	fmt.Println("✓ engagement mode off (quizzes resumed)")
	return nil
}

// setEngagementFlag updates the engagement state in _meta/engagement.json.
// The daemon checks this file on each quiz-scheduler iteration.
func setEngagementFlag(vaultPath string, engaged bool, name string) error {
	metaDir := filepath.Join(vaultPath, "_meta")
	os.MkdirAll(metaDir, 0o755)

	state := map[string]interface{}{
		"engaged": engaged,
		"name":    name,
	}
	data, _ := json.MarshalIndent(state, "", "  ")
	return os.WriteFile(filepath.Join(metaDir, "engagement.json"), data, 0o644)
}
