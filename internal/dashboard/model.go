package dashboard

import "time"

// DashboardData holds all data for the engagement dashboard.
type DashboardData struct {
	Box        string
	IP         string
	Phase      string
	Hosts      []HostInfo
	Findings   FindingsSummary
	Coverage   CoverageChecklist
	Timeline   []TimelineEntry
	NoteCount  int
}

// HostInfo represents a discovered host.
type HostInfo struct {
	IP       string
	Hostname string
	Ports    []PortInfo
	OS       string
}

// PortInfo represents an open port.
type PortInfo struct {
	Number  string
	Proto   string
	State   string
	Service string
	Version string
}

// FindingsSummary aggregates findings by severity.
type FindingsSummary struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Info     int
}

func (f FindingsSummary) Total() int {
	return f.Critical + f.High + f.Medium + f.Low + f.Info
}

// CoverageChecklist tracks which phases have content.
type CoverageChecklist struct {
	Recon   bool
	Enum    bool
	Exploit bool
	Post    bool
	Loot    bool
}

// TimelineEntry is a single event in the engagement timeline.
type TimelineEntry struct {
	Timestamp time.Time
	NoteID    string
	Title     string
	Tool      string
}
