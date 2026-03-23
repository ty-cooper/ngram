package vault

import (
	"fmt"
	"os"
	"path/filepath"
)

// Required vault directories.
var vaultDirs = []string{
	"_inbox",
	"_archive",
	"_processing",
	"_trash",
	"_meta",
	"_config",
	"_templates",
	"knowledge",
	"boxes",
	"tools",
}

// Init creates the full vault directory structure and seed files.
// Safe to call on an existing vault — only creates what's missing.
func Init(vaultPath string) error {
	for _, dir := range vaultDirs {
		if err := EnsureDir(filepath.Join(vaultPath, dir)); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	// Seed _meta files if they don't exist.
	seeds := map[string]string{
		"_meta/taxonomy.yml":       seedTaxonomy,
		"_meta/topic-clusters.yml": seedTopicClusters,
		"_templates/knowledge-note.md": templateKnowledgeNote,
		"_templates/box.md":            templateBox,
	}

	for rel, content := range seeds {
		path := filepath.Join(vaultPath, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", rel, err)
			}
		}
	}

	return nil
}

const seedTaxonomy = `# Canonical tags and domains for Ngram vault.
# The AI structuring pipeline resolves aliases to canonical names.
# Add new tags/domains here. Unknown tags proposed by the AI are used
# immediately and flagged for review via 'n tags --proposed'.

domains: {}
tags: {}
`

const seedTopicClusters = `# Topic clusters group related notes within a domain.
# Auto-extends as new domains appear during note processing.

auto_extend: true
review_new_domains: true

domains: {}
`

const templateKnowledgeNote = `---
id: ""
title: ""
content_type: "knowledge"
created: ""
modified: ""
source: ""
source_type: ""
domain: ""
topic_cluster: ""
tags: []
related: []
retention:
  state: "new"
  ease_factor: 2.5
  interval_days: 0
  repetition_count: 0
  last_reviewed: null
  next_review: null
  total_reviews: 0
  total_correct: 0
  retention_score: 0
  difficulty_rating: null
  streak: 0
  lapse_count: 0
---

`

const templateBox = `---
box: ""
ip: ""
os: ""
engagement: ""
phase: "recon"
---

## Overview

## Findings

## Notes
`
