package parsers

import (
	"strings"
	"testing"
)

func TestCrackmapexecParser(t *testing.T) {
	raw := `SMB         10.10.10.1      445    DC01             [+] domain\admin:password123 (Pwn3d!)
SMB         10.10.10.1      445    DC01             [-] domain\user1:wrong
SMB         10.10.10.1      445    DC01             [+] domain\user2:pass123`

	p := &CrackmapexecParser{}
	result, err := p.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Findings) != 3 {
		t.Errorf("findings = %d, want 3", len(result.Findings))
	}

	// First should be critical (admin + Pwn3d).
	if result.Findings[0].Severity != "critical" {
		t.Errorf("admin finding severity = %q, want critical", result.Findings[0].Severity)
	}

	if !strings.Contains(result.Summary, "2 valid") {
		t.Errorf("summary = %q, expected 2 valid", result.Summary)
	}
}
