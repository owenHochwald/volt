package shortcutpane

import (
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ui"
)

const (
	Global TabIndex = iota
	Sidebar
	Request
	Response
)

// TabIndex represents a tab position
type TabIndex int

// renderTabs renders the tab bar with shortcut categories
func (m ShortcutPane) renderTabs() string {
	var tabNames []string
	keys := m.keys.DirectHelpTab.Keys()
	for i, tab := range m.tabs {
		keyLabel := ""
		if i < len(keys) {
			keyLabel = "[" + keys[i] + "] "
		}
		tabNames = append(tabNames, lipgloss.NewStyle().Render(keyLabel+tab.Name))
	}
	return m.renderTabBar(tabNames)
}

// renderTabBar is a helper that renders a tab bar with active/inactive styling
func (m ShortcutPane) renderTabBar(tabs []string) string {
	renderedTabs := make([]string, 0, len(tabs)+1)

	for i, tab := range tabs {
		if i == m.activeTab {
			renderedTabs = append(renderedTabs, ui.ActiveTab.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, ui.InactiveTab.Render(tab))
		}
	}
	renderedTabs = append(renderedTabs, "\n")

	return lipgloss.JoinHorizontal(lipgloss.Left, renderedTabs...)
}

// getMaxTabs returns the max number of tabs for central control
func (m ShortcutPane) getMaxTabs() int {
	return len(m.tabs)
}
