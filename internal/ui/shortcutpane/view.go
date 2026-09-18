package shortcutpane

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m ShortcutPane) View() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Panel.Focused.GetBorderLeftForeground()).
		Padding(1, 2).
		Width(m.width).
		Height(m.height).
		MaxWidth(m.width).
		MaxHeight(m.height).
		Bold(true)

	title := m.styles.Text.Logo.Render("VOLT COMMAND CENTER")
	surfaceTabs := m.renderSurfaceTabs()

	var content, footerText string
	switch m.surface {
	case SurfaceHelp:
		m.helpViewport.SetContent(m.renderShortcutList())
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderTabs(),
			"",
			m.helpViewport.View(),
		)
		footerText = "h/l section • j/k category • 1-6 jump • pgup/pgdn scroll • q/? close"
	case SurfaceGroups:
		content = m.groupsPane.ViewContent()
		footerText = "h/l section • " + m.groupsPane.FooterHint()
	default:
		content = m.renderSettings()
		footerText = "h/l section • j/k select • enter save • esc cancel"
	}
	footer := m.styles.Text.Muted.Render(footerText)

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		surfaceTabs,
		"",
		content,
		"",
		footer,
	)

	return modalStyle.Render(modalContent)
}

// renderShortcutList renders the shortcuts for the active tab
func (m ShortcutPane) renderShortcutList() string {
	if m.activeTab >= len(m.tabs) {
		return "No shortcuts available"
	}

	tab := m.tabs[m.activeTab]
	var lines []string
	keyColumnWidth := 15
	for _, shortcut := range tab.Shortcuts {
		keyColumnWidth = max(keyColumnWidth, lipgloss.Width(shortcut.Key))
	}
	keyColumnWidth = min(keyColumnWidth, max(m.width/2, 15))

	header := m.styles.Text.Logo.Render(tab.Name + ":")
	lines = append(lines, header, "")

	// Shortcut list
	for _, shortcut := range tab.Shortcuts {
		key := m.styles.Text.Logo.Render(lipgloss.NewStyle().Width(keyColumnWidth).Render(shortcut.Key))
		desc := m.styles.Text.Value.Render(shortcut.Description)

		line := lipgloss.JoinHorizontal(lipgloss.Left, "  ", key, desc)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
