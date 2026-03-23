package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/capture"
	"github.com/ty-cooper/ngram/internal/config"
)

var (
	flagTitle string
	cfg       *config.Config
)

var rootCmd = &cobra.Command{
	Use:           "n [text...]",
	Short:         "Ngram — personal knowledge capture",
	Long:          "Capture notes, run commands, manage engagements. Everything lands in _inbox/ for AI processing.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

var captureCmd = &cobra.Command{
	Use:    "capture [text...]",
	Hidden: true,
	Args:   cobra.ArbitraryArgs,
	RunE:   rootRun,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagTitle, "title", "t", "", "explicit note title")

	rootCmd.AddCommand(captureCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(boxCmd)
	rootCmd.AddCommand(phaseCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(reindexCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(engageCmd)
	rootCmd.AddCommand(disengageCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(quizCmd)
	rootCmd.AddCommand(migrateCmd)
	rootCmd.AddCommand(captureOnCmd)
	rootCmd.AddCommand(captureOffCmd)
	rootCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(amendCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(domainsCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(ssCmd)
}

func Execute() error {
	// Cobra treats unknown positional words as subcommand errors.
	// We intercept os.Args to detect when the first arg is NOT a known
	// subcommand, and route it to the capture command instead.
	if len(os.Args) > 1 && !isSubcommand(os.Args[1]) && !strings.HasPrefix(os.Args[1], "-") {
		os.Args = append([]string{os.Args[0], "capture"}, os.Args[1:]...)
	} else if isPiped() && (len(os.Args) == 1 || onlyFlags(os.Args[1:])) {
		// Pipe mode: n [-t title] (receiving stdin)
		os.Args = insertAfter(os.Args, 0, "capture")
	}
	return rootCmd.Execute()
}

func isSubcommand(name string) bool {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name || cmd.HasAlias(name) {
			return true
		}
	}
	return false
}

func onlyFlags(args []string) bool {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			return false
		}
		// skip the value of flags like -t "title"
		if (a == "-t" || a == "--title") && i+1 < len(args) {
			i++
		}
	}
	return true
}

func insertAfter(args []string, idx int, val string) []string {
	result := make([]string, 0, len(args)+1)
	result = append(result, args[:idx+1]...)
	result = append(result, val)
	result = append(result, args[idx+1:]...)
	return result
}

func loadConfig() (*config.Config, error) {
	if cfg != nil {
		return cfg, nil
	}
	c, err := config.Load()
	if err != nil {
		return nil, err
	}
	cfg = c
	return cfg, nil
}

func rootRun(cmd *cobra.Command, args []string) error {
	isPipe := isPiped()

	if len(args) == 0 && !isPipe {
		return cmd.Help()
	}

	c, err := loadConfig()
	if err != nil {
		return err
	}

	var body string
	source := "terminal"

	if isPipe {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		body = strings.TrimRight(string(data), "\n")
		source = "pipe"
	} else {
		body = strings.Join(args, " ")
	}

	title := flagTitle
	if title == "" {
		title = capture.AutoTitle(body)
	}

	cwd, _ := os.Getwd()
	boxCtx, _ := config.FindBoxRC(cwd)

	meta := capture.NoteMetadata{
		Title:  title,
		Source: source,
		BoxCtx: boxCtx,
		Time:   time.Now(),
	}

	result, err := capture.WriteNote(c.VaultPath, body, meta)
	if err != nil {
		return fmt.Errorf("write note: %w", err)
	}

	fmt.Println(capture.Confirmation(result.RelPath, boxCtx))
	return nil
}

func isPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
