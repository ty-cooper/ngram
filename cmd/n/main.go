package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ty-cooper/ngram/internal/cli"
)

func main() {
	checkNameCollision()

	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// checkNameCollision warns if another `n` binary exists in PATH
// that isn't the ngram binary (e.g., Node.js version manager).
func checkNameCollision() {
	self, err := os.Executable()
	if err != nil {
		return
	}

	// Find all `n` binaries in PATH.
	path := os.Getenv("PATH")
	for _, dir := range strings.Split(path, ":") {
		candidate := dir + "/n"
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		// Resolve symlinks for comparison.
		resolvedSelf, _ := resolveLink(self)
		resolvedCandidate, _ := resolveLink(candidate)

		if resolvedCandidate != resolvedSelf && resolvedCandidate != "" {
			// Check if it's the Node.js version manager.
			out, _ := exec.Command(candidate, "--version").CombinedOutput()
			if strings.Contains(string(out), "node") || strings.Contains(string(out), "v") {
				fmt.Fprintf(os.Stderr, "warn: another 'n' binary found at %s (Node.js version manager?)\n", candidate)
				fmt.Fprintf(os.Stderr, "      consider aliasing ngram: alias n=%s\n", self)
				return
			}
		}
	}
}

func resolveLink(path string) (string, error) {
	resolved, err := os.Readlink(path)
	if err != nil {
		return path, nil // Not a symlink, return as-is.
	}
	return resolved, nil
}
