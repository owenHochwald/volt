package wspane

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ws"
)

// Local, package-scoped styles - mirrors importpane's own view.go, the
// closest existing precedent for a standalone modal pane that predates
// design.Styles injection.
var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	// focusedFieldStyle highlights the connect/disconnect button when it's
	// the focused field - it has no textinput.Model of its own to lean on
	// for that (unlike urlInput/composeInput, which show focus via their
	// own cursor), so this fills in the same visual cue directly.
	focusedFieldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	sentStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
	receivedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

// View renders the floating overlay, complete with its own modal border -
// used while !docked (see app/view.go's overlayModal). DockedView below
// renders the same content sized for a real panel box instead.
func (m WSPane) View() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(m.width).
		Height(m.height)

	const chrome = 4 // Border(2) + Padding(1,2)'s vertical padding(2)
	return modalStyle.Render(m.renderContent(chrome))
}

// DockedView renders the same content as View but without its floating
// modal border/padding - app/view.go's renderPanel supplies the box
// border and focus highlighting instead, the same way it already does
// for the request/response panes, using whatever SetWidth/SetHeight was
// last called with (app/layout.go points those at the docked box's own
// dimensions instead of the floating modal's while docked).
func (m WSPane) DockedView() string {
	return m.renderContent(0)
}

// renderContent lays out the header, message log, and footer - shared by
// View and DockedView, which differ only in how much of m.height chrome
// (a border and/or padding) eats into before the log gets what's left.
func (m WSPane) renderContent(chrome int) string {
	header := m.renderHeader()
	footer := m.renderFooter()

	// The message log gets whatever's left after everything else - it's
	// the one part of this pane whose content can grow unbounded, so it
	// has to be measured and fit rather than just laid out and hoping it
	// doesn't overflow the available height the way importpane's
	// fixed-shape form never needs to worry about.
	const blankLinesAroundLog = 2
	logHeight := max(m.height-chrome-lipgloss.Height(header)-lipgloss.Height(footer)-blankLinesAroundLog, 1)

	return lipgloss.JoinVertical(lipgloss.Left, header, "", m.renderLog(logHeight), "", footer)
}

func (m WSPane) renderHeader() string {
	buttonLabel := "[ Connect ]"
	if m.status == statusConnected {
		buttonLabel = "[ Disconnect ]"
	}
	if m.focus == focusConnectButton {
		buttonLabel = focusedFieldStyle.Render(buttonLabel)
	} else {
		buttonLabel = labelStyle.Render(buttonLabel)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("WebSocket"),
		"",
		labelStyle.Render("URL:"),
		m.urlInput.View(),
		"",
		buttonLabel+"   "+m.statusBadge(),
	)
}

func (m WSPane) statusBadge() string {
	switch m.status {
	case statusConnected:
		return okStyle.Render("● Connected")
	case statusConnecting:
		return warnStyle.Render("● Connecting…")
	default:
		return hintStyle.Render("● Disconnected")
	}
}

func (m WSPane) renderFooter() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		labelStyle.Render("Send:"),
		m.composeInput.View(),
		"",
		hintStyle.Render(m.footerHint()),
	)
}

func (m WSPane) footerHint() string {
	if m.docked {
		return "tab: next field · enter: connect/disconnect or send"
	}
	return "tab: next field · enter: connect/disconnect or send · esc: hide (keeps session running)"
}

// renderLog renders the last height entries of the message log - older
// entries simply scroll off the top, no interactive scrolling (the log
// always tails the most recent activity, the way a live session's
// history matters far more than its beginning).
func (m WSPane) renderLog(height int) string {
	if len(m.log) == 0 {
		return hintStyle.Render("(no messages yet)")
	}
	entries := m.log
	if len(entries) > height {
		entries = entries[len(entries)-height:]
	}
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, renderLogLine(e))
	}
	return strings.Join(lines, "\n")
}

func renderLogLine(e ws.LogEntry) string {
	ts := e.At.Format("15:04:05")
	switch e.Kind {
	case ws.EntrySent:
		return sentStyle.Render(fmt.Sprintf("%s → %s", ts, e.Text))
	case ws.EntryReceived:
		return receivedStyle.Render(fmt.Sprintf("%s ← %s", ts, e.Text))
	default:
		style := hintStyle
		if strings.HasPrefix(e.Text, "Error") {
			style = errStyle
		}
		return style.Render(fmt.Sprintf("%s • %s", ts, e.Text))
	}
}
