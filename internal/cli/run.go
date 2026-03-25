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
	"github.com/ty-cooper/ngram/internal/capture"
	"github.com/ty-cooper/ngram/internal/config"
	"github.com/ty-cooper/ngram/internal/parsers"
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

	// Extract --tool flag manually since DisableFlagParsing is on.
	var toolFlag string
	var commandArgs []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--tool" && i+1 < len(args) {
			toolFlag = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--tool=") {
			toolFlag = strings.TrimPrefix(args[i], "--tool=")
		} else if args[i] == "--" {
			commandArgs = append(commandArgs, args[i+1:]...)
			break
		} else {
			commandArgs = append(commandArgs, args[i])
		}
	}

	command := strings.Join(commandArgs, " ")

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

	// Detect and run parser.
	tool := toolFlag
	if tool == "" {
		tool = parsers.DetectTool(command)
	}

	var parsedResult *parsers.ParseResult
	if tool != "" {
		parsedResult, _ = parsers.Parse(tool, body)
	}

	// If parser produced structured output, prepend it.
	if parsedResult != nil && parsedResult.Markdown != "" {
		body = "## Parsed Output\n\n" + parsedResult.Markdown + "\n## Raw Output\n\n```\n" + body + "\n```"
	}

	title := flagTitle
	if title == "" {
		if parsedResult != nil && parsedResult.Summary != "" {
			title = parsedResult.Summary
		} else {
			title = capture.AutoTitle(command)
		}
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

	if parsedResult != nil {
		meta.Tool = parsedResult.Tool
		meta.FindingsCount = len(parsedResult.Findings)
	}

	result, err := capture.WriteNote(c.VaultPath, body, meta)
	if err != nil {
		return fmt.Errorf("write note: %w", err)
	}

	ctx := capture.Confirmation(result.RelPath, boxCtx)
	if parsedResult != nil && len(parsedResult.Findings) > 0 {
		ctx += fmt.Sprintf(" (%d findings parsed)", len(parsedResult.Findings))
	}
	fmt.Println(ctx)
	return nil
}
