package taxonomy

import (
	"os"
	"path/filepath"
	"testing"
)

const testTaxonomy = `domains:
  pentest:
    aliases: [pentesting, pen-testing, offensive-security]
  distributed-systems:
    aliases: [dist-sys, distributed]
  cooking:
    aliases: [recipes]

tags:
  nmap:
    aliases: [nmap-scan, network-mapper]
    domain_hint: pentest
  raft:
    aliases: [raft-consensus]
    domain_hint: distributed-systems
  proxychains:
    aliases: [proxychains4, proxy-chains]
    domain_hint: pentest
`

func setupTaxonomy(t *testing.T) (*Taxonomy, string) {
	t.Helper()
	dir := t.TempDir()
	metaDir := filepath.Join(dir, "_meta")
	os.MkdirAll(metaDir, 0o755)
	os.WriteFile(filepath.Join(metaDir, "taxonomy.yml"), []byte(testTaxonomy), 0o644)

	tax, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return tax, dir
}

func TestLoad(t *testing.T) {
	tax, _ := setupTaxonomy(t)
	if len(tax.Domains) != 3 {
		t.Errorf("expected 3 domains, got %d", len(tax.Domains))
	}
	if len(tax.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tax.Tags))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	tax, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("expected no error for missing taxonomy, got %v", err)
	}
	if len(tax.Domains) != 0 {
		t.Errorf("expected empty domains, got %d", len(tax.Domains))
	}
}

func TestResolveDomain(t *testing.T) {
	tax, _ := setupTaxonomy(t)

	tests := []struct {
		input string
		want  string
	}{
		{"pentest", "pentest"},
		{"pentesting", "pentest"},
		{"pen-testing", "pentest"},
		{"dist-sys", "distributed-systems"},
		{"distributed", "distributed-systems"},
		{"unknown-domain", "unknown-domain"},
	}

	for _, tt := range tests {
		if got := tax.ResolveDomain(tt.input); got != tt.want {
			t.Errorf("ResolveDomain(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveTag(t *testing.T) {
	tax, _ := setupTaxonomy(t)

	tests := []struct {
		input string
		want  string
	}{
		{"nmap", "nmap"},
		{"nmap-scan", "nmap"},
		{"network-mapper", "nmap"},
		{"proxychains4", "proxychains"},
		{"unknown-tag", "unknown-tag"},
	}

	for _, tt := range tests {
		if got := tax.ResolveTag(tt.input); got != tt.want {
			t.Errorf("ResolveTag(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveTags_Dedup(t *testing.T) {
	tax, _ := setupTaxonomy(t)

	input := []string{"nmap", "nmap-scan", "raft", "unknown"}
	got := tax.ResolveTags(input)

	if len(got) != 3 {
		t.Errorf("expected 3 unique tags, got %d: %v", len(got), got)
	}
}

func TestCanonicalLists(t *testing.T) {
	tax, _ := setupTaxonomy(t)

	domains := tax.CanonicalDomainList()
	if len(domains) != 3 {
		t.Errorf("expected 3 domains, got %d", len(domains))
	}

	tags := tax.CanonicalTagList()
	if len(tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(tags))
	}
}
