package parsers

import (
	"fmt"
	"regexp"
	"strings"
)

type NmapParser struct{}

func (p *NmapParser) Name() string { return "nmap" }

var (
	nmapPortRE = regexp.MustCompile(`^(\d+)/(tcp|udp)\s+(open|closed|filtered)\s+(\S+)\s*(.*)$`)
	nmapHostRE = regexp.MustCompile(`^Nmap scan report for\s+(.+)$`)
	nmapOSRE   = regexp.MustCompile(`^OS details?:\s+(.+)$`)
	nmapMACRE  = regexp.MustCompile(`^MAC Address:\s+(\S+)\s*(.*)$`)
)

type nmapHost struct {
	addr  string
	os    string
	mac   string
	ports []nmapPort
}

type nmapPort struct {
	port    string
	proto   string
	state   string
	service string
	version string
}

func (p *NmapParser) Parse(raw string) (*ParseResult, error) {
	raw = StripANSI(raw)
	lines := strings.Split(raw, "\n")

	var hosts []nmapHost
	var current *nmapHost

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if m := nmapHostRE.FindStringSubmatch(line); m != nil {
			hosts = append(hosts, nmapHost{addr: m[1]})
			current = &hosts[len(hosts)-1]
			continue
		}

		if current == nil {
			// Ports before any host header — create implicit host.
			if nmapPortRE.MatchString(line) {
				hosts = append(hosts, nmapHost{addr: "unknown"})
				current = &hosts[len(hosts)-1]
			} else {
				continue
			}
		}

		if m := nmapPortRE.FindStringSubmatch(line); m != nil {
			current.ports = append(current.ports, nmapPort{
				port:    m[1],
				proto:   m[2],
				state:   m[3],
				service: m[4],
				version: strings.TrimSpace(m[5]),
			})
		}

		if m := nmapOSRE.FindStringSubmatch(line); m != nil {
			current.os = m[1]
		}

		if m := nmapMACRE.FindStringSubmatch(line); m != nil {
			current.mac = m[1]
		}
	}

	if len(hosts) == 0 {
		return &ParseResult{Tool: "nmap", Summary: "nmap scan — no hosts found"}, nil
	}

	var findings []Finding
	var md strings.Builder
	totalPorts := 0

	for _, h := range hosts {
		if h.os != "" {
			findings = append(findings, Finding{
				Type:     "host",
				Severity: "info",
				Data:     map[string]string{"ip": h.addr, "os": h.os},
			})
		}

		md.WriteString(fmt.Sprintf("### %s\n\n", h.addr))
		if h.os != "" {
			md.WriteString(fmt.Sprintf("**OS**: %s\n\n", h.os))
		}

		if len(h.ports) > 0 {
			md.WriteString("| Port | State | Service | Version |\n")
			md.WriteString("|------|-------|---------|---------|\n")
			for _, port := range h.ports {
				findings = append(findings, Finding{
					Type:     "port",
					Severity: "info",
					Data: map[string]string{
						"ip":      h.addr,
						"port":    port.port,
						"proto":   port.proto,
						"state":   port.state,
						"service": port.service,
						"version": port.version,
					},
				})
				md.WriteString(fmt.Sprintf("| %s/%s | %s | %s | %s |\n",
					port.port, port.proto, port.state, port.service, port.version))
				totalPorts++
			}
			md.WriteString("\n")
		}
	}

	summary := fmt.Sprintf("nmap scan — %d host(s), %d open port(s)", len(hosts), totalPorts)

	return &ParseResult{
		Tool:     "nmap",
		Findings: findings,
		Summary:  summary,
		Markdown: md.String(),
	}, nil
}
