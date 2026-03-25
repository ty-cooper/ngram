package taxonomy

import (
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

type DomainEntry struct {
	Aliases []string `yaml:"aliases"`
}

type TagEntry struct {
	Aliases    []string `yaml:"aliases"`
	DomainHint string   `yaml:"domain_hint"`
}

type Taxonomy struct {
	Domains map[string]DomainEntry `yaml:"domains"`
	Tags    map[string]TagEntry    `yaml:"tags"`

	// Precomputed reverse lookup maps.
	domainAliases map[string]string
	tagAliases    map[string]string
}

// Load reads and parses _meta/taxonomy.yml from the vault.
// Returns an empty Taxonomy (not an error) if the file does not exist.
func Load(vaultPath string) (*Taxonomy, error) {
	path := filepath.Join(vaultPath, "_meta", "taxonomy.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Taxonomy{
				Domains:       make(map[string]DomainEntry),
				Tags:          make(map[string]TagEntry),
				domainAliases: make(map[string]string),
				tagAliases:    make(map[string]string),
			}, nil
		}
		return nil, err
	}

	var t Taxonomy
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if t.Domains == nil {
		t.Domains = make(map[string]DomainEntry)
	}
	if t.Tags == nil {
		t.Tags = make(map[string]TagEntry)
	}

	t.buildAliasMap()
	return &t, nil
}

func (t *Taxonomy) buildAliasMap() {
	t.domainAliases = make(map[string]string)
	for canonical, entry := range t.Domains {
		for _, alias := range entry.Aliases {
			t.domainAliases[strings.ToLower(alias)] = canonical
		}
	}

	t.tagAliases = make(map[string]string)
	for canonical, entry := range t.Tags {
		for _, alias := range entry.Aliases {
			t.tagAliases[strings.ToLower(alias)] = canonical
		}
	}
}

// ResolveDomain maps a raw domain string to its canonical form.
// Returns the input unchanged if no match.
func (t *Taxonomy) ResolveDomain(raw string) string {
	lower := strings.ToLower(raw)
	// Check if already canonical.
	if _, ok := t.Domains[lower]; ok {
		return lower
	}
	if canonical, ok := t.domainAliases[lower]; ok {
		return canonical
	}
	return raw
}

// ResolveTag maps a raw tag to its canonical form.
func (t *Taxonomy) ResolveTag(raw string) string {
	lower := strings.ToLower(raw)
	if _, ok := t.Tags[lower]; ok {
		return lower
	}
	if canonical, ok := t.tagAliases[lower]; ok {
		return canonical
	}
	return raw
}

// ResolveTags resolves a slice of raw tags, deduplicating.
func (t *Taxonomy) ResolveTags(raw []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, tag := range raw {
		resolved := t.ResolveTag(tag)
		if !seen[resolved] {
			seen[resolved] = true
			result = append(result, resolved)
		}
	}
	return result
}

// CanonicalTagList returns all canonical tag names.
func (t *Taxonomy) CanonicalTagList() []string {
	tags := make([]string, 0, len(t.Tags))
	for k := range t.Tags {
		tags = append(tags, k)
	}
	return tags
}

// CanonicalDomainList returns all canonical domain names.
func (t *Taxonomy) CanonicalDomainList() []string {
	domains := make([]string, 0, len(t.Domains))
	for k := range t.Domains {
		domains = append(domains, k)
	}
	return domains
}

// IsKnownTag returns true if the tag exists as a canonical name or alias.
func (t *Taxonomy) IsKnownTag(tag string) bool {
	lower := strings.ToLower(tag)
	if _, ok := t.Tags[lower]; ok {
		return true
	}
	_, ok := t.tagAliases[lower]
	return ok
}

// RegisterTags adds any unknown tags to the taxonomy as new canonical entries
// and persists the updated taxonomy to disk. First-come-first-serve.
func (t *Taxonomy) RegisterTags(tags []string, vaultPath string) {
	dirty := false
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if !t.IsKnownTag(tag) {
			t.Tags[strings.ToLower(tag)] = TagEntry{}
			dirty = true
		}
	}
	if dirty {
		t.buildAliasMap()
		t.save(vaultPath)
	}
}

// RegisterDomain adds an unknown domain to the taxonomy and persists.
func (t *Taxonomy) RegisterDomain(domain string, vaultPath string) {
	if domain == "" {
		return
	}
	lower := strings.ToLower(domain)
	if _, ok := t.Domains[lower]; ok {
		return
	}
	if _, ok := t.domainAliases[lower]; ok {
		return
	}
	t.Domains[lower] = DomainEntry{}
	t.buildAliasMap()
	t.save(vaultPath)
}

func (t *Taxonomy) save(vaultPath string) {
	path := filepath.Join(vaultPath, "_meta", "taxonomy.yml")
	data, err := yaml.Marshal(t)
	if err != nil {
		return
	}
	header := []byte("# Canonical tags and domains for Ngram vault.\n# Auto-populated as notes are processed. First tag wins as canonical.\n\n")
	os.WriteFile(path, append(header, data...), 0o644)
}
