package shortcutpane

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

func (m ShortcutPane) View() string {
	styles := design.NewStyles(design.DefaultTheme())
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Panel.Focused.GetBorderLeftForeground()).
		Padding(1, 2).
		Width(m.width).
		Height(m.height).
		Bold(true)

	title := styles.Text.Logo.Render("Keyboard Shortcuts")

	tabBar := m.renderTabs()
	content := m.renderShortcutList()

	footer := styles.Text.Muted.Render(m.keys.CloseHelp.Help().Key + " " + m.keys.CloseHelp.Help().Desc)

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

	styles := design.NewStyles(design.DefaultTheme())
	header := styles.Text.Logo.Render(tab.Name + ":")
	lines = append(lines, header, "")

	// Shortcut list
	for _, shortcut := range tab.Shortcuts {
		key := styles.Text.Logo.Render(lipgloss.NewStyle().Width(keyColumnWidth).Render(shortcut.Key))
		desc := styles.Text.Value.Render(shortcut.Description)

		line := lipgloss.JoinHorizontal(lipgloss.Left, "  ", key, desc)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
