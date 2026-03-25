package parsers

import (
	"strings"
	"testing"
)

func TestNucleiParser(t *testing.T) {
	raw := `{"template-id":"cve-2021-41773","info":{"name":"Apache Path Traversal","severity":"critical","description":"Path traversal in Apache 2.4.49","tags":["cve","apache"]},"matched-at":"http://10.10.10.1/cgi-bin/.%2e/.%2e/etc/passwd","host":"http://10.10.10.1"}
{"template-id":"http-missing-security-headers","info":{"name":"Missing X-Frame-Options","severity":"info","description":"","tags":["headers"]},"matched-at":"http://10.10.10.1","host":"http://10.10.10.1"}
{"template-id":"cve-2023-1234","info":{"name":"Some Vuln","severity":"high","description":"A high vuln","tags":["cve"]},"matched-at":"http://10.10.10.1/api","host":"http://10.10.10.1"}`

	p := &NucleiParser{}
	result, err := p.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Findings) != 3 {
		t.Errorf("findings = %d, want 3", len(result.Findings))
	}

	// Should be sorted by severity: critical, high, info.
	if result.Findings[0].Severity != "critical" {
		t.Errorf("first finding severity = %q, want critical", result.Findings[0].Severity)
	}
	if result.Findings[1].Severity != "high" {
		t.Errorf("second finding severity = %q, want high", result.Findings[1].Severity)
	}

	if !strings.Contains(result.Summary, "1 critical") {
		t.Errorf("summary = %q, expected critical count", result.Summary)
	}
}

func TestNucleiParserMixedOutput(t *testing.T) {
	raw := `[INF] Running nuclei engine...
[INF] Loaded 100 templates
{"template-id":"test","info":{"name":"Test","severity":"low","description":"","tags":[]},"matched-at":"http://target","host":"http://target"}
[INF] Scan complete`

	p := &NucleiParser{}
	result, err := p.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Findings) != 1 {
		t.Errorf("findings = %d, want 1", len(result.Findings))
	}
}
