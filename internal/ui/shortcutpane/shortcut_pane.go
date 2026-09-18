package shortcutpane

import (
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/groupspane"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

type Surface uint8

const (
	SurfaceHelp Surface = iota
	SurfaceSettings
	SurfaceGroups

	surfaceCount = 3
)

// next/previous cycle through every surface in order - h/l (see
// keys.PrevTab/NextTab in update.go) move backward/forward through all of
// them, not just toggle between two.
func (s Surface) next() Surface {
	return Surface((int(s) + 1) % surfaceCount)
}

func (s Surface) previous() Surface {
	return Surface((int(s) - 1 + surfaceCount) % surfaceCount)
}

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

	// groupsPane hosts the Groups & Servers manager as this pane's Groups
	// surface - migrated in from what used to be app.Model's own
	// standalone ctrl+g modal, so Groups/Servers/Help/Settings all live
	// behind one modal instead of two.
	groupsPane groupspane.GroupsPane

	Focused bool
	keys    keybindings.KeyMap
	styles  design.Styles
}

func (m ShortcutPane) Init() tea.Cmd {
	return nil
}

func (m *ShortcutPane) SetFocused(focused bool) {
	m.Focused = focused
	m.groupsPane.SetFocused(focused)
}

func (m *ShortcutPane) SetStyles(styles design.Styles) {
	m.styles = styles
}

func (m *ShortcutPane) SetHeight(height int) {
	m.height = height
	m.groupsPane.SetHeight(height)
}

func (m *ShortcutPane) SetWidth(width int) {
	m.width = width
	m.groupsPane.SetWidth(width)
}

// SetSurface switches which surface is active, reloading the Groups
// surface's data every time it becomes active - it may have gone stale
// since it was last shown (e.g. edited via `volt import`/`volt group`/
// `volt server` from a separate terminal, or a prior visit to this same
// surface) and it always starts back in its own browse state, matching
// GroupsPane.Open's original "reload every time this is shown" contract.
func (m *ShortcutPane) SetSurface(surface Surface) {
	m.surface = surface
	if surface == SurfaceGroups {
		m.groupsPane.Open()
	}
}

// IsEditing reports whether the Groups surface's own create/rename/edit
// form is currently capturing keystrokes - see GroupsPane.IsEditing.
func (m ShortcutPane) IsEditing() bool {
	return m.surface == SurfaceGroups && m.groupsPane.IsEditing()
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
