package parsers

import (
	"path/filepath"
	"strings"
)

var registry = map[string]Parser{}

func Register(p Parser) { registry[p.Name()] = p }

func Get(name string) (Parser, bool) {
	p, ok := registry[strings.ToLower(name)]
	return p, ok
}

// DetectTool extracts the tool name from a command string and checks the registry.
func DetectTool(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	// Strip path: /usr/bin/nmap → nmap
	bin := filepath.Base(fields[0])
	if _, ok := registry[bin]; ok {
		return bin
	}
	return ""
}

// Parse runs the named parser on raw output. Returns nil if no parser found.
func Parse(tool, raw string) (*ParseResult, error) {
	p, ok := Get(tool)
	if !ok {
		return nil, nil
	}
	return p.Parse(raw)
}

func init() {
	Register(&NmapParser{})
	Register(&GobusterParser{})
	Register(&FfufParser{})
	Register(&NucleiParser{})
	Register(&CrackmapexecParser{})
	Register(&NiktoParser{})
}
