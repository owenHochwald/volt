package shortcutpane

import (
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// ShortcutPane is the component responsible for displaying shortcuts
type ShortcutPane struct {
	activeTab     int
	height, width int
	tabs          []ShortcutTab

	Focused bool
	keys    keybindings.KeyMap
	styles  design.Styles
}

func (m ShortcutPane) Init() tea.Cmd {
	return nil
}

func (m *ShortcutPane) SetFocused(focused bool) {
	m.Focused = focused
}

func (m *ShortcutPane) SetHeight(height int) {
	m.height = height
}

func (m *ShortcutPane) SetWidth(width int) {
	m.width = width
}

func (m *ShortcutPane) SetContext(context keybindings.Context) {
	for i, tab := range m.tabs {
		if tab.Context == context {
			m.activeTab = i
			return
		}
	}
}

func (m ShortcutPane) ActiveContext() keybindings.Context {
	if m.activeTab < 0 || m.activeTab >= len(m.tabs) {
		return ""
	}
	return m.tabs[m.activeTab].Context
}
