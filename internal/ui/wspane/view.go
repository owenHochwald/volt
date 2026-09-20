package wspane

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ws"
)

// View renders the floating overlay - its border and every accent color
// come from the active theme (m.styles), the same as every other panel,
// rather than a hardcoded color of its own.
func (m WSPane) View() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Panel.Focused.GetBorderLeftForeground()).
		Padding(1, 2).
		Width(m.width).
		Height(m.height)

	header := m.renderHeader()
	footer := m.renderFooter()

	// The message log gets whatever's left after everything else - it's
	// the one part of this modal whose content can grow unbounded, so it
	// has to be measured and fit rather than just laid out and hoping it
	// doesn't overflow the modal's fixed height the way importpane's
	// fixed-shape form never needs to worry about.
	const chrome = 4 // Border(2) + Padding(1,2)'s vertical padding(2)
	const blankLinesAroundLog = 2
	logHeight := max(m.height-chrome-lipgloss.Height(header)-lipgloss.Height(footer)-blankLinesAroundLog, 1)

	content := lipgloss.JoinVertical(lipgloss.Left, header, "", m.renderLog(logHeight), "", footer)
	return modalStyle.Render(content)
}

func (m WSPane) renderHeader() string {
	buttonLabel := "[ Connect ]"
	if m.status == statusConnected {
		buttonLabel = "[ Disconnect ]"
	}
	if m.focus == focusConnectButton {
		buttonLabel = m.styles.Action.Focused.Render(buttonLabel)
	} else {
		buttonLabel = m.styles.Action.Primary.Render(buttonLabel)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Text.Logo.Render("WebSocket"),
		"",
		m.styles.Text.Label.Render("URL:"),
		m.urlInput.View(),
		"",
		buttonLabel+"   "+m.statusBadge(),
	)
}

func (m WSPane) statusBadge() string {
	switch m.status {
	case statusConnected:
		return m.styles.Badge.Success.Render("● Connected")
	case statusConnecting:
		return m.styles.Badge.Warning.Render("● Connecting…")
	default:
		return m.styles.Text.Muted.Render("● Disconnected")
	}
}

func (m WSPane) renderFooter() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Text.Label.Render("Send:"),
		m.composeInput.View(),
		"",
		m.styles.Text.Faint.Render(m.footerHint()),
	)
}

func (m WSPane) footerHint() string {
	return "tab: next field · enter: connect/disconnect or send · esc: hide (keeps session running)"
}

// renderLog renders the last height entries of the message log - older
// entries simply scroll off the top, no interactive scrolling (the log
// always tails the most recent activity, the way a live session's
// history matters far more than its beginning).
func (m WSPane) renderLog(height int) string {
	if len(m.log) == 0 {
		return m.styles.Text.Faint.Render("(no messages yet)")
	}
	entries := m.log
	if len(entries) > height {
		entries = entries[len(entries)-height:]
	}
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, m.renderLogLine(e))
	}
	return strings.Join(lines, "\n")
}

func (m WSPane) renderLogLine(e ws.LogEntry) string {
	ts := e.At.Format("15:04:05")
	switch e.Kind {
	case ws.EntrySent:
		return m.styles.Notice.Info.Render(fmt.Sprintf("%s → %s", ts, e.Text))
	case ws.EntryReceived:
		return m.styles.Notice.Success.Render(fmt.Sprintf("%s ← %s", ts, e.Text))
	default:
		style := m.styles.Text.Muted
		if strings.HasPrefix(e.Text, "Error") {
			style = m.styles.Notice.Error
		}
		return style.Render(fmt.Sprintf("%s • %s", ts, e.Text))
	}
}
