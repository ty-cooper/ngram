package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type BoxContext struct {
	Box        string
	IP         string
	Phase      string
	Engagement string
	Model      string
}

// FindBoxRC walks from dir up to root looking for a .boxrc file.
// Returns nil if none found.
func FindBoxRC(dir string) (*BoxContext, error) {
	for {
		path := filepath.Join(dir, ".boxrc")
		if _, err := os.Stat(path); err == nil {
			return ParseBoxRC(path)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
	}
}

// ParseBoxRC reads a KEY=VALUE file and returns a BoxContext.
func ParseBoxRC(path string) (*BoxContext, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ctx := &BoxContext{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = stripQuotes(val)

		switch key {
		case "BOX":
			ctx.Box = val
		case "IP":
			ctx.IP = val
		case "PHASE":
			ctx.Phase = val
		case "ENGAGEMENT":
			ctx.Engagement = val
		case "MODEL":
			ctx.Model = val
		}
	}

	return ctx, scanner.Err()
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
