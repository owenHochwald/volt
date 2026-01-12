package shortcutpane

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// ShortcutPane is the component responsible for displaying shortcuts
type ShortcutPane struct {
	activeTab     int
	height, width int
	tabs          []ShortcutTab

	Focused bool
	keys    keybindings.KeyMap
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
