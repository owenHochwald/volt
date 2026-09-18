package responsepane

import (
	"charm.land/lipgloss/v2"
)

// Status styles
var (
	// errorTextStyle renders a plain-text error explanation (e.g. the
	// Timing tab's "request failed" note) - unlike m.styles.Badge.Error,
	// which is a filled badge meant for a short status word like "ERROR",
	// not a full sentence.
	errorTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	// scrollIndicatorStyle renders the "more above/below" hint shown when a
	// tab's content overflows the viewport - see ResponsePane.scrollIndicator.
	scrollIndicatorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Italic(true)

	// successTextStyle renders a plain-text success confirmation (e.g.
	// "saved to file.json") - the mirror of errorTextStyle above, for a
	// good outcome instead of a bad one.
	successTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	// footerHelpStyle renders this pane's pinned one-line key hint (e.g.
	// "r: raw/pretty · s: save to file · y: copy"), dim so it stays out of
	// the way of the actual response content above it.
	footerHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// currentMatchStyle/otherMatchStyle highlight a search's matching
	// lines (see highlightMatches) - currentMatchStyle for whichever one
	// "n"/"b" would move away from, otherMatchStyle (more subdued) for
	// every other match, so the current one is unambiguous at a glance.
	currentMatchStyle = lipgloss.NewStyle().Background(lipgloss.Color("208")).Foreground(lipgloss.Color("0")).Bold(true)
	otherMatchStyle   = lipgloss.NewStyle().Background(lipgloss.Color("58")).Foreground(lipgloss.Color("255"))
)
