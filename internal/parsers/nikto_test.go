package parsers

import (
	"strings"
	"testing"
)

func TestNiktoParser(t *testing.T) {
	raw := `- Nikto v2.5.0
---------------------------------------------------------------------------
+ Target IP:          10.10.10.1
+ Target Hostname:    target.htb
+ Target Port:        80
+ Start Time:         2024-01-15 10:30:00 (GMT0)
---------------------------------------------------------------------------
+ Server: Apache/2.4.49 (Ubuntu)
+ OSVDB-3092: /admin/: This might be interesting...
+ OSVDB-3233: /icons/README: Apache default file found.
+ /login.php: Admin login page/section found.
+ 7 host(s) tested`

	p := &NiktoParser{}
	result, err := p.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Findings) != 3 {
		t.Errorf("findings = %d, want 3", len(result.Findings))
	}

	// Check OSVDB extraction.
	found := false
	for _, f := range result.Findings {
		if f.Data["osvdb"] == "OSVDB-3092" {
			found = true
		}
	}
	if !found {
		t.Error("missing OSVDB-3092 finding")
	}

	if !strings.Contains(result.Summary, "target.htb") {
		t.Errorf("summary = %q, expected target hostname", result.Summary)
	}
}
