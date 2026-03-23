package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/config"
	"github.com/ty-cooper/ngram/internal/vault"
)

var ssCmd = &cobra.Command{
	Use:   "ss",
	Short: "Capture a screenshot and send to _inbox/ for processing",
	RunE:  ssRun,
}

func ssRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	// Create capture bundle directory in _inbox/.
	ts := time.Now().Unix()
	bundleName := fmt.Sprintf("%d-screenshot", ts)
	bundleDir := filepath.Join(c.VaultPath, "_inbox", bundleName)
	if err := vault.EnsureDir(bundleDir); err != nil {
		return fmt.Errorf("create bundle dir: %w", err)
	}

	imgPath := filepath.Join(bundleDir, "capture.png")

	// macOS: use screencapture -i (interactive region select).
	capture := exec.Command("screencapture", "-i", imgPath)
	capture.Stdout = os.Stdout
	capture.Stderr = os.Stderr
	if err := capture.Run(); err != nil {
		// User cancelled (Esc).
		os.RemoveAll(bundleDir)
		return fmt.Errorf("screenshot cancelled")
	}

	// Verify file was created (user might have pressed Esc).
	if _, err := os.Stat(imgPath); os.IsNotExist(err) {
		os.RemoveAll(bundleDir)
		return fmt.Errorf("screenshot cancelled")
	}

	// Write manifest.yml for the processor to detect as a capture bundle.
	boxrc, _ := config.FindBoxRC("")
	box, phase := "", ""
	if boxrc != nil {
		box = boxrc.Box
		phase = boxrc.Phase
	}
	manifest := fmt.Sprintf(`items:
  - type: image
    path: capture.png
box: %q
phase: %q
source: screenshot
`, box, phase)

	manifestPath := filepath.Join(bundleDir, "manifest.yml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	relPath := filepath.Join("_inbox", bundleName)
	fmt.Printf("✓ %s/capture.png\n", relPath)
	return nil
}
