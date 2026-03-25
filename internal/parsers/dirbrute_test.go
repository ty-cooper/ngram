package parsers

import (
	"strings"
	"testing"
)

func TestGobusterParser(t *testing.T) {
	raw := `===============================================================
Gobuster v3.6
===============================================================
/admin (Status: 200) [Size: 1234]
/login (Status: 302) [Size: 56]
/api (Status: 403) [Size: 287]
/robots.txt (Status: 200) [Size: 42]
===============================================================`

	p := &GobusterParser{}
	result, err := p.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Findings) != 4 {
		t.Errorf("findings = %d, want 4", len(result.Findings))
	}

	if !strings.Contains(result.Markdown, "/admin") {
		t.Error("markdown missing /admin")
	}

	if !strings.Contains(result.Summary, "4 path") {
		t.Errorf("summary = %q, expected 4 paths", result.Summary)
	}
}

func TestFfufParser(t *testing.T) {
	raw := `admin [Status: 200, Size: 1234, Words: 100, Lines: 50]
login [Status: 302, Size: 56, Words: 2, Lines: 1]
secret [Status: 403, Size: 287, Words: 10, Lines: 5]`

	p := &FfufParser{}
	result, err := p.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Findings) != 3 {
		t.Errorf("findings = %d, want 3", len(result.Findings))
	}
}
