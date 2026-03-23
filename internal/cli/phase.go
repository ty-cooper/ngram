package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ty-cooper/ngram/internal/config"
)

var validPhases = []string{"recon", "enum", "exploit", "post", "loot"}

var phaseCmd = &cobra.Command{
	Use:       "phase <phase>",
	Short:     "Switch the active engagement phase",
	Long:      fmt.Sprintf("Updates PHASE in the nearest .boxrc. Valid phases: %s", strings.Join(validPhases, ", ")),
	Args:      cobra.ExactArgs(1),
	RunE:      phaseRun,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return validPhases, cobra.ShellCompDirectiveNoFileComp
	},
}

func phaseRun(cmd *cobra.Command, args []string) error {
	phase := args[0]
	if !isValidPhase(phase) {
		return fmt.Errorf("invalid phase %q (valid: %s)", phase, strings.Join(validPhases, ", "))
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	rcPath, err := findBoxRCPath(cwd)
	if err != nil {
		return err
	}

	if err := updatePhase(rcPath, phase); err != nil {
		return err
	}

	boxCtx, _ := config.ParseBoxRC(rcPath)
	boxName := ""
	if boxCtx != nil {
		boxName = boxCtx.Box
	}

	fmt.Printf("✓ phase → %s [%s]\n", phase, boxName)
	return nil
}

func isValidPhase(phase string) bool {
	for _, v := range validPhases {
		if v == phase {
			return true
		}
	}
	return false
}

func findBoxRCPath(dir string) (string, error) {
	for {
		path := filepath.Join(dir, ".boxrc")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .boxrc found in current directory or parents")
		}
		dir = parent
	}
}

func updatePhase(path, phase string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	var lines []string
	found := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PHASE=") {
			lines = append(lines, "PHASE="+phase)
			found = true
		} else {
			lines = append(lines, line)
		}
	}
	f.Close()

	if err := scanner.Err(); err != nil {
		return err
	}

	if !found {
		lines = append(lines, "PHASE="+phase)
	}

	content := strings.Join(lines, "\n") + "\n"

	tmp, err := os.CreateTemp(filepath.Dir(path), ".boxrc-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	tmp.Close()

	return os.Rename(tmpName, path)
}
