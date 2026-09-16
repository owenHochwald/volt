package responsepane

import (
	"github.com/charmbracelet/lipgloss"
)

// Tab styles
var (
	inactiveTab = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("240")) // dimGray

	activeTab = lipgloss.NewStyle().
			Padding(0, 2).
			Background(lipgloss.Color("98")). // darkPurple
			Foreground(lipgloss.Color("255")).
			Bold(true)
)

// Content styles
var (
	responseKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true) // focusColor/Pink for keys
	responseValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))            // Light gray for values
	responseLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("93")).Bold(true)  // deepViolet for labels
)

// Status styles
var (
	errorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("196")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			Bold(true)

	loadTestStatusStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230")).
				Padding(0, 1)

	faintStyle = lipgloss.NewStyle().Faint(true)

	// errorTextStyle renders a plain-text error explanation (e.g. the
	// Timing tab's "request failed" note) - unlike errorStyle above, which
	// is a filled badge meant for a short status word like "ERROR", not a
	// full sentence.
	errorTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	// scrollIndicatorStyle renders the "more above/below" hint shown when a
	// tab's content overflows the viewport - see ResponsePane.scrollIndicator.
	scrollIndicatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Italic(true)
)
