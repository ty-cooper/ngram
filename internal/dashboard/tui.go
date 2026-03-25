package dashboard

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ty-cooper/ngram/internal/search"
)

type tickMsg time.Time

// Model is the Bubbletea model for the dashboard.
type Model struct {
	data       *DashboardData
	vaultPath  string
	boxName    string
	search     *search.Client
	width      int
	height     int
	err        error
	lastUpdate time.Time
}

// New creates a new dashboard model.
func New(vaultPath, boxName string, client *search.Client) Model {
	data, _ := Load(vaultPath, boxName, client)
	return Model{
		data:       data,
		vaultPath:  vaultPath,
		boxName:    boxName,
		search:     client,
		lastUpdate: time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.data, _ = Load(m.vaultPath, m.boxName, m.search)
			m.lastUpdate = time.Now()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		m.data, _ = Load(m.vaultPath, m.boxName, m.search)
		m.lastUpdate = time.Now()
		return m, tickCmd()
	}
	return m, nil
}

func (m Model) View() tea.View {
	if m.data == nil {
		return tea.NewView("Loading...")
	}
	d := m.data

	var b strings.Builder

	// Header.
	header := fmt.Sprintf(" %s [%s]", d.Box, d.Phase)
	if d.IP != "" {
		header += " — " + d.IP
	}
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n\n")

	// Three columns: Hosts | Findings | Coverage
	hostsCol := m.renderHosts()
	findingsCol := m.renderFindings()
	coverageCol := m.renderCoverage()

	colWidth := 30
	if m.width > 0 {
		colWidth = (m.width - 10) / 3
		if colWidth < 20 {
			colWidth = 20
		}
	}

	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(colWidth).Render(hostsCol),
		lipgloss.NewStyle().Width(colWidth).Render(findingsCol),
		lipgloss.NewStyle().Width(colWidth).Render(coverageCol),
	)
	b.WriteString(cols)
	b.WriteString("\n\n")

	// Timeline.
	b.WriteString(m.renderTimeline())
	b.WriteString("\n")

	// Footer.
	footer := fmt.Sprintf(" %d notes — q:quit r:refresh — updated %s",
		d.NoteCount, m.lastUpdate.Format("15:04:05"))
	b.WriteString(dimStyle.Render(footer))

	return tea.NewView(borderStyle.Render(b.String()))
}

func (m Model) renderHosts() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("HOSTS"))
	b.WriteString("\n")

	if len(m.data.Hosts) == 0 {
		b.WriteString(dimStyle.Render("  (none discovered)"))
		return b.String()
	}

	for _, h := range m.data.Hosts {
		b.WriteString(fmt.Sprintf("  %s\n", h.IP))
		for _, p := range h.Ports {
			svc := p.Service
			if p.Version != "" {
				svc += " " + p.Version
			}
			b.WriteString(fmt.Sprintf("    %s/%s %s\n", p.Number, p.Proto, svc))
		}
	}
	return b.String()
}

func (m Model) renderFindings() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("FINDINGS"))
	b.WriteString("\n")

	f := m.data.Findings
	if f.Total() == 0 {
		b.WriteString(dimStyle.Render("  (none)"))
		return b.String()
	}

	if f.Critical > 0 {
		b.WriteString(criticalStyle.Render("■") + fmt.Sprintf(" %d critical\n", f.Critical))
	}
	if f.High > 0 {
		b.WriteString(highStyle.Render("■") + fmt.Sprintf(" %d high\n", f.High))
	}
	if f.Medium > 0 {
		b.WriteString(mediumStyle.Render("■") + fmt.Sprintf(" %d medium\n", f.Medium))
	}
	if f.Low > 0 {
		b.WriteString(lowStyle.Render("□") + fmt.Sprintf(" %d low\n", f.Low))
	}
	if f.Info > 0 {
		b.WriteString(infoStyle.Render("□") + fmt.Sprintf(" %d info\n", f.Info))
	}
	return b.String()
}

func (m Model) renderCoverage() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("COVERAGE"))
	b.WriteString("\n")

	c := m.data.Coverage
	checks := []struct {
		done  bool
		label string
	}{
		{c.Recon, "recon"},
		{c.Enum, "enum"},
		{c.Exploit, "exploit"},
		{c.Post, "post"},
		{c.Loot, "loot"},
	}
	for _, ch := range checks {
		if ch.done {
			b.WriteString(checkStyle.Render("[x]"))
		} else {
			b.WriteString(uncheckStyle.Render("[ ]"))
		}
		b.WriteString(" " + ch.label + "\n")
	}
	return b.String()
}

func (m Model) renderTimeline() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("RECENT ACTIVITY"))
	b.WriteString("\n")

	if len(m.data.Timeline) == 0 {
		b.WriteString(dimStyle.Render("  (no activity)"))
		return b.String()
	}

	for _, e := range m.data.Timeline {
		title := e.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		b.WriteString(fmt.Sprintf("  %s  %s\n", e.NoteID[:8], title))
	}
	return b.String()
}
