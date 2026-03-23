package query

import (
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	today := time.Now().Format("2006-01-02")

	tests := []struct {
		input      string
		wantQuery  string
		wantFilter string
	}{
		{
			input:      "proxy setup",
			wantQuery:  "proxy setup",
			wantFilter: "",
		},
		{
			input:      "proxy setup domain:pentest",
			wantQuery:  "proxy setup",
			wantFilter: `domain = "pentest"`,
		},
		{
			input:      "creds box:optimum tag:smb",
			wantQuery:  "creds",
			wantFilter: `box = "optimum" AND tags = "smb"`,
		},
		{
			input:      "domain:distributed-systems raft leader",
			wantQuery:  "raft leader",
			wantFilter: `domain = "distributed-systems"`,
		},
		{
			input:      "type:knowledge phase:exploit box:optimum privesc",
			wantQuery:  "privesc",
			wantFilter: `content_type = "knowledge" AND phase = "exploit" AND box = "optimum"`,
		},
		{
			// State filter.
			input:      "state:learning",
			wantQuery:  "",
			wantFilter: `retention_state = "learning"`,
		},
		{
			// Score comparison.
			input:      "score:<60",
			wantQuery:  "",
			wantFilter: `retention_score < 60`,
		},
		{
			input:      "score:>80",
			wantQuery:  "",
			wantFilter: `retention_score > 80`,
		},
		{
			// Lapse comparison.
			input:      "lapse:>2",
			wantQuery:  "",
			wantFilter: `lapse_count > 2`,
		},
		{
			// Due overdue.
			input:     "due:overdue",
			wantQuery: "",
		},
		{
			// Due today.
			input:     "due:today",
			wantQuery: "",
		},
		{
			// Combined.
			input:      "firewall domain:pentest state:learning score:<60",
			wantQuery:  "firewall",
			wantFilter: `domain = "pentest" AND retention_state = "learning" AND retention_score < 60`,
		},
		{
			// Unknown field treated as search text.
			input:      "author:tyler something",
			wantQuery:  "author:tyler something",
			wantFilter: "",
		},
		{
			// Colon at start or end is not a filter.
			input:      ":leading trailing:",
			wantQuery:  ":leading trailing:",
			wantFilter: "",
		},
		{
			input:      "",
			wantQuery:  "",
			wantFilter: "",
		},
		{
			// Source filter.
			input:      "source:DDIA",
			wantQuery:  "",
			wantFilter: `source = "DDIA"`,
		},
		{
			// Cluster filter.
			input:      "cluster:consensus",
			wantQuery:  "",
			wantFilter: `topic_cluster = "consensus"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Parse(tt.input)
			if got.Query != tt.wantQuery {
				t.Errorf("Query = %q, want %q", got.Query, tt.wantQuery)
			}
			if tt.wantFilter != "" && got.Filter != tt.wantFilter {
				t.Errorf("Filter = %q, want %q", got.Filter, tt.wantFilter)
			}
			// For date-dependent filters, just check they contain the date.
			if tt.input == "due:overdue" {
				if !strings.Contains(got.Filter, today) {
					t.Errorf("due:overdue filter should contain today's date, got %q", got.Filter)
				}
				if !strings.Contains(got.Filter, "next_review <") {
					t.Errorf("due:overdue should use < operator, got %q", got.Filter)
				}
			}
			if tt.input == "due:today" {
				if !strings.Contains(got.Filter, today) {
					t.Errorf("due:today filter should contain today's date, got %q", got.Filter)
				}
			}
		})
	}
}
