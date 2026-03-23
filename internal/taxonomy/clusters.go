package taxonomy

import (
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// ClusterEntry defines a topic cluster within a domain.
type ClusterEntry struct {
	Keywords []string `yaml:"keywords"`
}

// DomainClusters defines clusters for a single domain.
type DomainClusters struct {
	Description string                  `yaml:"description"`
	Clusters    map[string]ClusterEntry `yaml:"clusters"`
}

// TopicClusters is the _meta/topic-clusters.yml schema.
type TopicClusters struct {
	AutoExtend       bool                      `yaml:"auto_extend"`
	ReviewNewDomains bool                      `yaml:"review_new_domains"`
	Domains          map[string]DomainClusters `yaml:"domains"`
}

// LoadClusters reads _meta/topic-clusters.yml from the vault.
// Returns an empty TopicClusters if the file does not exist.
func LoadClusters(vaultPath string) (*TopicClusters, error) {
	path := filepath.Join(vaultPath, "_meta", "topic-clusters.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &TopicClusters{
				AutoExtend: true,
				Domains:    make(map[string]DomainClusters),
			}, nil
		}
		return nil, err
	}

	var tc TopicClusters
	if err := yaml.Unmarshal(data, &tc); err != nil {
		return nil, err
	}
	if tc.Domains == nil {
		tc.Domains = make(map[string]DomainClusters)
	}
	return &tc, nil
}

// AddDomain adds a new domain with an optional cluster. Writes back to disk.
func (tc *TopicClusters) AddDomain(vaultPath, domain, cluster string) error {
	if _, ok := tc.Domains[domain]; !ok {
		tc.Domains[domain] = DomainClusters{
			Clusters: make(map[string]ClusterEntry),
		}
	}

	if cluster != "" {
		d := tc.Domains[domain]
		if d.Clusters == nil {
			d.Clusters = make(map[string]ClusterEntry)
		}
		if _, ok := d.Clusters[cluster]; !ok {
			d.Clusters[cluster] = ClusterEntry{}
		}
		tc.Domains[domain] = d
	}

	return tc.Save(vaultPath)
}

// Save writes the topic clusters back to _meta/topic-clusters.yml.
func (tc *TopicClusters) Save(vaultPath string) error {
	data, err := yaml.Marshal(tc)
	if err != nil {
		return err
	}
	path := filepath.Join(vaultPath, "_meta", "topic-clusters.yml")
	return os.WriteFile(path, data, 0o644)
}
