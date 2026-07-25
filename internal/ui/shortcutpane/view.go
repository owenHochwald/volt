package shortcutpane

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m ShortcutPane) View() string {
	// Modal container style
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(m.width).
		Height(m.height).
		Bold(true)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render("Keyboard Shortcuts")

	tabBar := m.renderTabs()
	content := m.renderShortcutList()

	// Footer hint
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(m.keys.CloseHelp.Help().Key + " " + m.keys.CloseHelp.Help().Desc)

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		tabBar,
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

	// Tab name header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render(tab.Name + ":")
	lines = append(lines, header, "")

	// Shortcut list
	for _, shortcut := range tab.Shortcuts {
		keyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

		key := keyStyle.Render(lipgloss.NewStyle().Width(keyColumnWidth).Render(shortcut.Key))
		desc := lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Render(shortcut.Description)

		line := lipgloss.JoinHorizontal(lipgloss.Left, "  ", key, desc)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
