package parsers

import (
	"fmt"
	"regexp"
	"strings"
)

type NiktoParser struct{}

func (p *NiktoParser) Name() string { return "nikto" }

var (
	niktoFindingRE = regexp.MustCompile(`^\+\s+(.+)$`)
	niktoOSVDBRE   = regexp.MustCompile(`OSVDB-(\d+)`)
	niktoTargetRE  = regexp.MustCompile(`^\+\s+Target IP:\s+(\S+)`)
	niktoHostRE    = regexp.MustCompile(`^\+\s+Target Hostname:\s+(\S+)`)
	niktoPortRE    = regexp.MustCompile(`^\+\s+Target Port:\s+(\d+)`)
)

func (p *NiktoParser) Parse(raw string) (*ParseResult, error) {
	raw = StripANSI(raw)
	lines := strings.Split(raw, "\n")

	var findings []Finding
	var md strings.Builder
	var targetIP, targetHost, targetPort string

	md.WriteString("| Finding | OSVDB |\n")
	md.WriteString("|---------|-------|\n")

	hasFindings := false

	skipPrefixes := []string{
		"Target IP:", "Target Hostname:", "Target Port:",
		"Start Time:", "End Time:", "Server:",
		"Nikto", "-----------",
	}
	skipContains := []string{
		"host(s) tested", "item(s) reported",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if m := niktoTargetRE.FindStringSubmatch(line); m != nil {
			targetIP = m[1]
			continue
		}
		if m := niktoHostRE.FindStringSubmatch(line); m != nil {
			targetHost = m[1]
			continue
		}
		if m := niktoPortRE.FindStringSubmatch(line); m != nil {
			targetPort = m[1]
			continue
		}

		m := niktoFindingRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		finding := m[1]

		skip := false
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(finding, prefix) {
				skip = true
				break
			}
		}
		if !skip {
			for _, substr := range skipContains {
				if strings.Contains(finding, substr) {
					skip = true
					break
				}
			}
		}
		if skip {
			continue
		}

		osvdb := ""
		if om := niktoOSVDBRE.FindStringSubmatch(finding); om != nil {
			osvdb = "OSVDB-" + om[1]
		}

		findings = append(findings, Finding{
			Type:     "vuln",
			Severity: "info",
			Data: map[string]string{
				"finding": finding,
				"osvdb":   osvdb,
				"ip":      targetIP,
				"host":    targetHost,
				"port":    targetPort,
			},
			Raw: line,
		})

		md.WriteString(fmt.Sprintf("| %s | %s |\n", finding, osvdb))
		hasFindings = true
	}

	if !hasFindings {
		md.Reset()
	} else {
		md.WriteString("\n")
	}

	summary := fmt.Sprintf("nikto — %d finding(s)", len(findings))
	if targetHost != "" {
		summary = fmt.Sprintf("nikto %s — %d finding(s)", targetHost, len(findings))
	}

	return &ParseResult{
		Tool:     "nikto",
		Findings: findings,
		Summary:  summary,
		Markdown: md.String(),
	}, nil
}
