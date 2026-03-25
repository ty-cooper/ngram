package dashboard

import "charm.land/lipgloss/v2"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	criticalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).Bold(true)

	highStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208"))

	mediumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	lowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	checkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	uncheckStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)
