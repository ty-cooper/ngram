package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tylercooper/ngram/internal/capture"
	"github.com/tylercooper/ngram/internal/config"
)

var runCmd = &cobra.Command{
	Use:                "run [command...]",
	Short:              "Execute a command and capture its output as a note",
	Long:               "Runs the command via /bin/sh -c, streams output to terminal, and saves both the command and output to _inbox/.",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE:               runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	command := strings.Join(args, " ")

	var stdout, stderr bytes.Buffer
	sh := exec.Command("/bin/sh", "-c", command)
	sh.Stdout = io.MultiWriter(os.Stdout, &stdout)
	sh.Stderr = io.MultiWriter(os.Stderr, &stderr)
	sh.Stdin = os.Stdin

	_ = sh.Run() // capture output even on non-zero exit

	body := stdout.String()
	if errOut := stderr.String(); errOut != "" {
		body += "\n--- stderr ---\n" + errOut
	}
	body = strings.TrimRight(body, "\n")

	title := flagTitle
	if title == "" {
		title = capture.AutoTitle(command)
	}

	cwd, _ := os.Getwd()
	boxCtx, _ := config.FindBoxRC(cwd)

	meta := capture.NoteMetadata{
		Title:   title,
		Source:  "command-capture",
		Command: command,
		BoxCtx:  boxCtx,
		Time:    time.Now(),
	}

	result, err := capture.WriteNote(c.VaultPath, body, meta)
	if err != nil {
		return fmt.Errorf("write note: %w", err)
	}

	fmt.Println(capture.Confirmation(result.RelPath, boxCtx))
	return nil
}
