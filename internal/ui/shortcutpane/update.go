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
		// While the Groups surface's own create/rename/edit form is
		// capturing keystrokes, every key below (the surface switch, tab
		// jumps, the whole-modal close) must reach it untouched instead of
		// being intercepted - typing e.g. "h" or "q" into a name field
		// would otherwise switch surfaces or close the modal instead of
		// entering the literal character.
		editing := m.surface == SurfaceGroups && m.groupsPane.IsEditing()

		if !editing {
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
				return m, nil
			}

			if keybindings.Matches(msg, m.keys.PrevTab) {
				m.SetSurface(m.surface.previous())
				return m, nil
			}
			if keybindings.Matches(msg, m.keys.NextTab) {
				m.SetSurface(m.surface.next())
				return m, nil
			}
		}

		// The Groups surface owns its own j/k (list navigation) and
		// enter/n/r/d/q (create/rename/delete/close) - see
		// GroupsPane.Update - rather than the tab-cycling/theme-picking
		// meaning those keys have on the Help/Settings surfaces below.
		if m.surface == SurfaceGroups {
			var cmd tea.Cmd
			m.groupsPane, cmd = m.groupsPane.Update(msg)
			return m, cmd
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
