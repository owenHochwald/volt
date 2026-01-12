package shortcutpane

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// CloseHelpModalMsg signals the app to close the help modal
type CloseHelpModalMsg struct{}

func (m ShortcutPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Direct tab access - check for numbers
		switch msg.String() {
		case "1":
			m.activeTab = int(Global)
		case "2":
			m.activeTab = int(Sidebar)
		case "3":
			m.activeTab = int(Request)
		case "4":
			m.activeTab = int(Response)
		}

		// Tab navigation
		if keybindings.Matches(msg, m.keys.PrevTab) {
			m.activeTab = (m.activeTab - 1 + m.getMaxTabs()) % m.getMaxTabs()
		}
		if keybindings.Matches(msg, m.keys.NextTab) {
			m.activeTab = (m.activeTab + 1) % m.getMaxTabs()
		}

		// Close modal
		if keybindings.Matches(msg, m.keys.CloseHelp) {
			return m, func() tea.Msg {
				return CloseHelpModalMsg{}
			}
		}
	}

	return m, nil
}
