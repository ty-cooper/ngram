package query

import (
	"fmt"
	"strings"
)

// Known filterable fields and their Meilisearch attribute names.
var filterFields = map[string]string{
	"domain":     "domain",
	"box":        "box",
	"phase":      "phase",
	"tag":        "tags",
	"type":       "content_type",
	"engagement": "engagement",
}

// Parsed holds the separated free-text query and Meilisearch filter expression.
type Parsed struct {
	Query  string
	Filter string
}

// Parse splits a raw search string into free-text and field:value filters.
//
// Examples:
//
//	"proxy setup domain:pentest" → Query="proxy setup", Filter=`domain = "pentest"`
//	"creds box:optimum tag:smb"  → Query="creds", Filter=`box = "optimum" AND tags = "smb"`
//	"suid privesc"               → Query="suid privesc", Filter=""
func Parse(raw string) Parsed {
	words := strings.Fields(raw)
	var textParts []string
	var filters []string

	for _, w := range words {
		idx := strings.Index(w, ":")
		if idx <= 0 || idx == len(w)-1 {
			textParts = append(textParts, w)
			continue
		}

		field := strings.ToLower(w[:idx])
		value := w[idx+1:]

		attr, ok := filterFields[field]
		if !ok {
			// Not a known filter field; treat as search text.
			textParts = append(textParts, w)
			continue
		}

		filters = append(filters, fmt.Sprintf(`%s = "%s"`, attr, value))
	}

	return Parsed{
		Query:  strings.Join(textParts, " "),
		Filter: strings.Join(filters, " AND "),
	}
}
