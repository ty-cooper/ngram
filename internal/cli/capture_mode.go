package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var captureOnCmd = &cobra.Command{
	Use:   "capture-on",
	Short: "Start auto-capture mode (log all commands)",
	RunE:  captureOnRun,
}

var captureOffCmd = &cobra.Command{
	Use:   "capture-off",
	Short: "Stop auto-capture mode",
	RunE:  captureOffRun,
}

func captureOnRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	state := map[string]interface{}{
		"active": true,
	}
	data, _ := json.MarshalIndent(state, "", "  ")

	metaDir := filepath.Join(c.VaultPath, "_meta")
	os.MkdirAll(metaDir, 0o755)

	if err := os.WriteFile(filepath.Join(metaDir, "capture-mode.json"), data, 0o644); err != nil {
		return err
	}

	fmt.Println("✓ capture mode ON")
	fmt.Println("  all commands will be logged to the active box")
	fmt.Println("  run 'n capture-off' to stop")
	return nil
}

func captureOffRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	state := map[string]interface{}{
		"active": false,
	}
	data, _ := json.MarshalIndent(state, "", "  ")

	path := filepath.Join(c.VaultPath, "_meta", "capture-mode.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	fmt.Println("✓ capture mode OFF")
	return nil
}
