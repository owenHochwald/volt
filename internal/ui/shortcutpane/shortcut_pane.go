package shortcutpane

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

	// helpViewport scrolls the Help surface's shortcut list (see
	// renderShortcutList) when it's taller than the modal has room for
	// (e.g. the Response category's ~20 bindings on a small terminal) -
	// see PageUp/PageDown handling in update.go. Sized by SetWidth/
	// SetHeight (see helpContentWidth/helpListHeight below), since View()
	// has a value receiver and can't persist its own size adjustments
	// back onto the real model the way Update() can - the same reason
	// requestpane's FieldsViewport is sized from its own SetWidth/
	// SetHeight rather than from within View().
	helpViewport viewport.Model

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
	m.groupsPane.SetStyles(styles)
}

func (m *ShortcutPane) SetHeight(height int) {
	m.height = height
	m.groupsPane.SetHeight(height)
	m.helpViewport.SetHeight(m.helpListHeight())
}

func (m *ShortcutPane) SetWidth(width int) {
	m.width = width
	m.groupsPane.SetWidth(width)
	m.helpViewport.SetWidth(m.helpContentWidth())
	// The tab bar's own height (see renderTabs) can change with width -
	// a narrow enough modal wraps it onto more lines - so the viewport's
	// available height needs recomputing here too, not just in
	// SetHeight.
	m.helpViewport.SetHeight(m.helpListHeight())
}

// helpContentWidth/helpListHeight compute the Help surface's shortcut
// list viewport size from the modal's own chrome - the outer border+
// padding (see View's modalStyle), the fixed title/surface-tabs/footer
// rows every surface shares, and the Help surface's own category tab bar
// above the list - so the viewport gets exactly the room that's actually
// left over, not a guess.
func (m ShortcutPane) helpContentWidth() int {
	const horizontalFrame = 6 // modalStyle's Border(2) + Padding(0,2)*2(4)
	return max(m.width-horizontalFrame, 1)
}

func (m ShortcutPane) helpListHeight() int {
	const verticalFrame = 4 // modalStyle's Border(2) + Padding(1,0)*2(2)
	const fixedChrome = 6   // title, blank, surface tabs, blank, blank, footer (see View)
	const listSeparator = 1 // blank line between the category tab bar and the list
	tabBarHeight := lipgloss.Height(m.renderTabs())
	return max(m.height-verticalFrame-fixedChrome-tabBarHeight-listSeparator, 1)
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
