package parsers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type NucleiParser struct{}

func (p *NucleiParser) Name() string { return "nuclei" }

type nucleiResult struct {
	TemplateID string `json:"template-id"`
	Info       struct {
		Name        string   `json:"name"`
		Severity    string   `json:"severity"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	} `json:"info"`
	MatchedAt string `json:"matched-at"`
	Host      string `json:"host"`
}

func (p *NucleiParser) Parse(raw string) (*ParseResult, error) {
	raw = StripANSI(raw)
	lines := strings.Split(raw, "\n")

	var results []nucleiResult
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}
		var r nucleiResult
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		results = append(results, r)
	}

	if len(results) == 0 {
		return &ParseResult{Tool: "nuclei", Summary: "nuclei — no findings"}, nil
	}

	// Sort by severity.
	sevOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3, "info": 4}
	sort.Slice(results, func(i, j int) bool {
		return sevOrder[results[i].Info.Severity] < sevOrder[results[j].Info.Severity]
	})

	var findings []Finding
	var md strings.Builder

	md.WriteString("| Severity | Template | Finding | Target |\n")
	md.WriteString("|----------|----------|---------|--------|\n")

	sevCount := map[string]int{}
	for _, r := range results {
		sev := strings.ToLower(r.Info.Severity)
		if sev == "" {
			sev = "info"
		}
		sevCount[sev]++

		findings = append(findings, Finding{
			Type:     "vuln",
			Severity: sev,
			Data: map[string]string{
				"template_id": r.TemplateID,
				"name":        r.Info.Name,
				"matched_at":  r.MatchedAt,
				"host":        r.Host,
				"description": r.Info.Description,
			},
		})

		target := r.MatchedAt
		if target == "" {
			target = r.Host
		}
		md.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			sev, r.TemplateID, r.Info.Name, target))
	}
	md.WriteString("\n")

	var parts []string
	for _, sev := range []string{"critical", "high", "medium", "low", "info"} {
		if n := sevCount[sev]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, sev))
		}
	}
	summary := fmt.Sprintf("nuclei — %s", strings.Join(parts, ", "))

	return &ParseResult{
		Tool:     "nuclei",
		Findings: findings,
		Summary:  summary,
		Markdown: md.String(),
	}, nil
}
