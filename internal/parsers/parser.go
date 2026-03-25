package parsers

// Finding is a single extracted data point from tool output.
type Finding struct {
	Type     string            // "host", "port", "vuln", "path", "credential"
	Severity string            // "critical", "high", "medium", "low", "info"
	Data     map[string]string // tool-specific key/value pairs
	Raw      string            // original output line(s) that produced this
}

// ParseResult is what a parser returns.
type ParseResult struct {
	Tool     string
	Findings []Finding
	Summary  string // auto-generated one-liner
	Markdown string // pre-formatted structured markdown sections
}

// Parser extracts structured data from raw tool output.
type Parser interface {
	Name() string
	Parse(raw string) (*ParseResult, error)
}
