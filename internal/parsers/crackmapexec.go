package parsers

import (
	"fmt"
	"regexp"
	"strings"
)

type CrackmapexecParser struct{}

func (p *CrackmapexecParser) Name() string { return "crackmapexec" }

var (
	// CME output: PROTO  IP  PORT  HOST  [+] or [-] message
	cmeResultRE = regexp.MustCompile(`^(\S+)\s+(\d+\.\d+\.\d+\.\d+)\s+(\d+)\s+(\S+)\s+\[([+-])\]\s+(.+)$`)
	// Share enumeration: SHARE  READ/WRITE access
	cmeShareRE = regexp.MustCompile(`^\s+(\S+)\s+(READ|WRITE|READ,\s*WRITE|NO ACCESS)\s*(.*)$`)
)

func (p *CrackmapexecParser) Parse(raw string) (*ParseResult, error) {
	raw = StripANSI(raw)
	lines := strings.Split(raw, "\n")

	var findings []Finding
	var md strings.Builder
	var validCreds, invalidCreds int
	var shares []string

	md.WriteString("### Credential Results\n\n")
	md.WriteString("| Host | Protocol | Result | Detail |\n")
	md.WriteString("|------|----------|--------|--------|\n")

	hasCredResults := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if m := cmeResultRE.FindStringSubmatch(line); m != nil {
			proto := m[1]
			ip := m[2]
			// port := m[3]
			// host := m[4]
			success := m[5]
			detail := m[6]

			result := "✗"
			fType := "credential"
			if success == "+" {
				result = "✓"
				validCreds++
			} else {
				invalidCreds++
			}

			findings = append(findings, Finding{
				Type:     fType,
				Severity: severityFromCME(success, detail),
				Data: map[string]string{
					"ip":       ip,
					"protocol": proto,
					"success":  success,
					"detail":   detail,
				},
				Raw: line,
			})

			md.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", ip, proto, result, detail))
			hasCredResults = true
			continue
		}

		if m := cmeShareRE.FindStringSubmatch(line); m != nil {
			share := m[1]
			access := m[2]
			shares = append(shares, fmt.Sprintf("%s (%s)", share, access))

			findings = append(findings, Finding{
				Type:     "path",
				Severity: "info",
				Data:     map[string]string{"share": share, "access": access},
			})
		}
	}

	if !hasCredResults {
		md.Reset()
	} else {
		md.WriteString("\n")
	}

	if len(shares) > 0 {
		md.WriteString("### Shares\n\n")
		for _, s := range shares {
			md.WriteString(fmt.Sprintf("- %s\n", s))
		}
		md.WriteString("\n")
	}

	var parts []string
	if validCreds > 0 {
		parts = append(parts, fmt.Sprintf("%d valid", validCreds))
	}
	if invalidCreds > 0 {
		parts = append(parts, fmt.Sprintf("%d invalid", invalidCreds))
	}
	if len(shares) > 0 {
		parts = append(parts, fmt.Sprintf("%d shares", len(shares)))
	}

	summary := "crackmapexec — no results"
	if len(parts) > 0 {
		summary = fmt.Sprintf("crackmapexec — %s", strings.Join(parts, ", "))
	}

	return &ParseResult{
		Tool:     "crackmapexec",
		Findings: findings,
		Summary:  summary,
		Markdown: md.String(),
	}, nil
}

func severityFromCME(success, detail string) string {
	if success == "+" {
		lower := strings.ToLower(detail)
		if strings.Contains(lower, "admin") || strings.Contains(lower, "pwn") {
			return "critical"
		}
		return "high"
	}
	return "info"
}
