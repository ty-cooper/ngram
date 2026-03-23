package query

import (
	"fmt"
	"strings"
	"time"
)

// Known filterable fields that map directly to equality filters.
var equalityFields = map[string]string{
	"domain":       "domain",
	"box":          "box",
	"phase":        "phase",
	"tag":          "tags",
	"type":         "content_type",
	"content_type": "content_type",
	"engagement":   "engagement",
	"state":        "retention_state",
	"source":       "source",
	"source_type":  "source_type",
	"cluster":      "topic_cluster",
}

// Parsed holds the separated free-text query and Meilisearch filter expression.
type Parsed struct {
	Query  string
	Filter string
}

// Parse splits a raw search string into free-text and field:value filters.
//
// Supports:
//
//	field:value        → equality filter
//	score:<60          → retention_score < 60
//	score:>80          → retention_score > 80
//	lapse:>2           → lapse_count > 2
//	streak:>5          → streak > 5
//	due:overdue        → next_review < today
//	due:today          → next_review = today
func Parse(raw string) Parsed {
	words := strings.Fields(raw)
	var textParts []string
	var filters []string

	today := time.Now().Format("2006-01-02")

	for _, w := range words {
		idx := strings.Index(w, ":")
		if idx <= 0 || idx == len(w)-1 {
			textParts = append(textParts, w)
			continue
		}

		field := strings.ToLower(w[:idx])
		value := w[idx+1:]

		// Check comparison fields (score, lapse, streak).
		if f := parseComparison(field, value); f != "" {
			filters = append(filters, f)
			continue
		}

		// Check date filters.
		if field == "due" {
			switch strings.ToLower(value) {
			case "overdue":
				filters = append(filters, fmt.Sprintf(`next_review < "%s"`, today))
			case "today":
				filters = append(filters, fmt.Sprintf(`next_review = "%s"`, today))
			default:
				// Treat as a specific date.
				filters = append(filters, fmt.Sprintf(`next_review = "%s"`, value))
			}
			continue
		}

		// Check equality fields.
		if attr, ok := equalityFields[field]; ok {
			filters = append(filters, fmt.Sprintf(`%s = "%s"`, attr, value))
			continue
		}

		// Unknown field; treat as search text.
		textParts = append(textParts, w)
	}

	return Parsed{
		Query:  strings.Join(textParts, " "),
		Filter: strings.Join(filters, " AND "),
	}
}

// comparisonFields maps shorthand to Meilisearch attribute names
// for fields that support <, > operators.
var comparisonFields = map[string]string{
	"score":  "retention_score",
	"lapse":  "lapse_count",
	"streak": "streak",
}

func parseComparison(field, value string) string {
	attr, ok := comparisonFields[field]
	if !ok {
		return ""
	}

	if len(value) < 2 {
		return ""
	}

	switch value[0] {
	case '<':
		return fmt.Sprintf(`%s < %s`, attr, value[1:])
	case '>':
		return fmt.Sprintf(`%s > %s`, attr, value[1:])
	default:
		// Plain equality.
		return fmt.Sprintf(`%s = %s`, attr, value)
	}
}
