package query

import (
	"testing"
)

func TestParse(t *testing.T) {
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
			// Unknown field treated as search text.
			input:      "author:tyler something",
			wantQuery:  "author:tyler something",
			wantFilter: "",
		},
		{
			// Colon at start or end of word is not a filter.
			input:      ":leading trailing:",
			wantQuery:  ":leading trailing:",
			wantFilter: "",
		},
		{
			input:      "",
			wantQuery:  "",
			wantFilter: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Parse(tt.input)
			if got.Query != tt.wantQuery {
				t.Errorf("Query = %q, want %q", got.Query, tt.wantQuery)
			}
			if got.Filter != tt.wantFilter {
				t.Errorf("Filter = %q, want %q", got.Filter, tt.wantFilter)
			}
		})
	}
}
