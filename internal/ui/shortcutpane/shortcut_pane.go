package shortcutpane

import (
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

type Surface uint8

const (
	SurfaceHelp Surface = iota
	SurfaceSettings
)

type ThemeOption struct {
	Name        string
	Description string
	Source      string
	Theme       design.Theme
}

// ShortcutPane is the component responsible for displaying shortcuts
type ShortcutPane struct {
	activeTab     int
	height, width int
	tabs          []ShortcutTab
	surface       Surface
	themeOptions  []ThemeOption
	themeIndex    int
	savedTheme    int
	previewing    bool

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

func (m *ShortcutPane) SetStyles(styles design.Styles) {
	m.styles = styles
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
