package shortcutpane

import (
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// CloseHelpModalMsg signals the app to close the help modal
type CloseHelpModalMsg struct{}

func (m ShortcutPane) Update(msg tea.Msg) (ShortcutPane, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Direct tab access - check for numbers
		if m.surface == SurfaceHelp && keybindings.Matches(msg, m.keys.DirectHelpTab) {
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
		}

		if keybindings.Matches(msg, m.keys.PrevTab) {
			m.surface = SurfaceHelp
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.NextTab) {
			m.surface = SurfaceSettings
			return m, nil
		}

		if keybindings.Matches(msg, m.keys.NavUp) {
			if m.surface == SurfaceSettings {
				return m, m.moveTheme(-1)
			}
			m.activeTab = (m.activeTab - 1 + m.getMaxTabs()) % m.getMaxTabs()
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.NavDown) {
			if m.surface == SurfaceSettings {
				return m, m.moveTheme(1)
			}
			m.activeTab = (m.activeTab + 1) % m.getMaxTabs()
			return m, nil
		}

		if m.surface == SurfaceSettings && keybindings.Matches(msg, m.keys.ActivateControl) {
			return m, m.saveTheme()
		}

		// Close modal
		if keybindings.Matches(msg, m.keys.CloseHelp) {
			if m.previewing {
				return m, m.cancelThemePreview()
			}
			return m, func() tea.Msg {
				return CloseHelpModalMsg{}
			}
		}
	}

	return m, nil
}
