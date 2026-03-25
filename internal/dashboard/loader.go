package dashboard

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ty-cooper/ngram/internal/search"
)

// Load populates DashboardData from Meilisearch and the filesystem.
func Load(vaultPath, boxName string, client *search.Client) (*DashboardData, error) {
	data := &DashboardData{Box: boxName}

	// Load coverage from filesystem.
	boxDir := filepath.Join(vaultPath, "boxes", boxName)
	data.Coverage = loadCoverage(boxDir)

	if client == nil {
		return data, nil
	}

	// Query all notes for this box.
	results, err := client.FindSimilarFiltered("", 250, `box = "`+boxName+`"`)
	if err != nil {
		return data, nil // graceful
	}

	data.NoteCount = len(results)
	hostMap := map[string]*HostInfo{}

	for _, r := range results {
		// Build timeline.
		data.Timeline = append(data.Timeline, TimelineEntry{
			NoteID: r.ID,
			Title:  r.Title,
		})

		// Parse findings from body for severity counts.
		lower := strings.ToLower(r.Body)
		if strings.Contains(lower, "critical") {
			data.Findings.Critical++
		}
		if strings.Contains(lower, "high") && strings.Contains(lower, "severity") {
			data.Findings.High++
		}

		// Extract host/port info from parsed output tables.
		extractHostPorts(r.Body, hostMap)
	}

	for _, h := range hostMap {
		data.Hosts = append(data.Hosts, *h)
	}
	sort.Slice(data.Hosts, func(i, j int) bool {
		return data.Hosts[i].IP < data.Hosts[j].IP
	})

	// Sort timeline by note ID (which encodes order).
	sort.Slice(data.Timeline, func(i, j int) bool {
		return data.Timeline[i].NoteID < data.Timeline[j].NoteID
	})

	// Limit timeline to last 10.
	if len(data.Timeline) > 10 {
		data.Timeline = data.Timeline[len(data.Timeline)-10:]
	}

	return data, nil
}

func loadCoverage(boxDir string) CoverageChecklist {
	var c CoverageChecklist
	c.Recon = dirHasNotes(filepath.Join(boxDir, "_recon"))
	c.Enum = dirHasNotes(filepath.Join(boxDir, "_enum"))
	c.Exploit = dirHasNotes(filepath.Join(boxDir, "_exploit"))
	c.Post = dirHasNotes(filepath.Join(boxDir, "_post"))
	c.Loot = dirHasNotes(filepath.Join(boxDir, "_loot"))
	return c
}

func dirHasNotes(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			return true
		}
	}
	return false
}

// extractHostPorts parses markdown tables for host/port data.
func extractHostPorts(body string, hosts map[string]*HostInfo) {
	lines := strings.Split(body, "\n")
	currentHost := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect host header: ### 10.10.10.1
		if strings.HasPrefix(line, "### ") {
			currentHost = strings.TrimPrefix(line, "### ")
			if _, ok := hosts[currentHost]; !ok {
				hosts[currentHost] = &HostInfo{IP: currentHost}
			}
			continue
		}

		// Parse table row: | 22/tcp | open | ssh | OpenSSH 8.9 |
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "---") {
			continue
		}
		cols := strings.Split(line, "|")
		if len(cols) < 5 {
			continue
		}
		portProto := strings.TrimSpace(cols[1])
		state := strings.TrimSpace(cols[2])
		service := strings.TrimSpace(cols[3])
		version := strings.TrimSpace(cols[4])

		if portProto == "Port" || portProto == "" {
			continue
		}

		parts := strings.SplitN(portProto, "/", 2)
		port := parts[0]
		proto := ""
		if len(parts) > 1 {
			proto = parts[1]
		}

		if currentHost == "" {
			currentHost = "unknown"
		}
		if _, ok := hosts[currentHost]; !ok {
			hosts[currentHost] = &HostInfo{IP: currentHost}
		}

		hosts[currentHost].Ports = append(hosts[currentHost].Ports, PortInfo{
			Number:  port,
			Proto:   proto,
			State:   state,
			Service: service,
			Version: version,
		})
	}
}

// Ensure time is imported for TimelineEntry.
var _ = time.Now
