package parsers

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// GobusterParser handles gobuster dir/vhost output.
type GobusterParser struct{}

func (p *GobusterParser) Name() string { return "gobuster" }

// FfufParser handles ffuf output.
type FfufParser struct{}

func (p *FfufParser) Name() string { return "ffuf" }

var (
	// Gobuster: /admin (Status: 200) [Size: 1234]
	gobusterRE = regexp.MustCompile(`^(/\S*)\s+\(Status:\s*(\d+)\)\s+\[Size:\s*(\d+)\]`)
	// Ffuf: path [Status: 200, Size: 1234, Words: 56, Lines: 12]
	ffufRE = regexp.MustCompile(`^(\S+)\s+\[Status:\s*(\d+),\s*Size:\s*(\d+)`)
)

type dirEntry struct {
	path   string
	status string
	size   string
}

func (p *GobusterParser) Parse(raw string) (*ParseResult, error) {
	return parseDirBrute("gobuster", raw, gobusterRE)
}

func (p *FfufParser) Parse(raw string) (*ParseResult, error) {
	return parseDirBrute("ffuf", raw, ffufRE)
}

func parseDirBrute(tool, raw string, re *regexp.Regexp) (*ParseResult, error) {
	raw = StripANSI(raw)
	lines := strings.Split(raw, "\n")

	var entries []dirEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		m := re.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		entries = append(entries, dirEntry{
			path:   m[1],
			status: m[2],
			size:   m[3],
		})
	}

	if len(entries) == 0 {
		return &ParseResult{Tool: tool, Summary: fmt.Sprintf("%s — no paths discovered", tool)}, nil
	}

	// Group by status code.
	grouped := map[string][]dirEntry{}
	for _, e := range entries {
		grouped[e.status] = append(grouped[e.status], e)
	}

	var findings []Finding
	var md strings.Builder

	// Sort status codes.
	var codes []string
	for c := range grouped {
		codes = append(codes, c)
	}
	sort.Strings(codes)

	md.WriteString("| Path | Status | Size |\n")
	md.WriteString("|------|--------|------|\n")

	for _, code := range codes {
		for _, e := range grouped[code] {
			findings = append(findings, Finding{
				Type:     "path",
				Severity: "info",
				Data:     map[string]string{"path": e.path, "status": e.status, "size": e.size},
			})
			md.WriteString(fmt.Sprintf("| %s | %s | %s |\n", e.path, e.status, e.size))
		}
	}
	md.WriteString("\n")

	summary := fmt.Sprintf("%s — %d path(s) discovered", tool, len(entries))

	return &ParseResult{
		Tool:     tool,
		Findings: findings,
		Summary:  summary,
		Markdown: md.String(),
	}, nil
}
