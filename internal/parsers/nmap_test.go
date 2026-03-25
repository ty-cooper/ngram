package parsers

import (
	"strings"
	"testing"
)

func TestNmapParser(t *testing.T) {
	raw := `Starting Nmap 7.94 ( https://nmap.org ) at 2024-01-15 10:30 UTC
Nmap scan report for 10.10.10.1
Host is up (0.042s latency).

PORT     STATE SERVICE VERSION
22/tcp   open  ssh     OpenSSH 8.9p1 Ubuntu
80/tcp   open  http    Apache httpd 2.4.49
443/tcp  open  https   nginx 1.18.0
3306/tcp closed mysql
OS details: Linux 5.4

Nmap done: 1 IP address (1 host up) scanned in 12.34 seconds`

	p := &NmapParser{}
	result, err := p.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	if result.Tool != "nmap" {
		t.Errorf("tool = %q, want nmap", result.Tool)
	}

	// 4 port findings + 1 host finding (OS)
	if len(result.Findings) != 5 {
		t.Errorf("findings = %d, want 5", len(result.Findings))
	}

	// Check port findings.
	ports := 0
	for _, f := range result.Findings {
		if f.Type == "port" {
			ports++
		}
	}
	if ports != 4 {
		t.Errorf("port findings = %d, want 4", ports)
	}

	// Check markdown has table.
	if !strings.Contains(result.Markdown, "| 22/tcp |") {
		t.Error("markdown missing port 22 row")
	}
	if !strings.Contains(result.Markdown, "Apache httpd 2.4.49") {
		t.Error("markdown missing Apache version")
	}

	if !strings.Contains(result.Summary, "4 open port") {
		t.Errorf("summary = %q, expected open port count", result.Summary)
	}
}

func TestNmapParserEmpty(t *testing.T) {
	p := &NmapParser{}
	result, err := p.Parse("Starting Nmap...\nNmap done: 0 hosts up")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("findings = %d, want 0", len(result.Findings))
	}
}
