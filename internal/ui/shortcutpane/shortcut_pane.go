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
	m.applySizing()
}

func (m *ShortcutPane) SetWidth(width int) {
	m.width = width
	m.applySizing()
}

// applySizing gives the Groups surface and the Help surface's shortcut
// list viewport each exactly the content area they actually have to
// render into, recomputed from the modal's own chrome via contentArea -
// called whenever width or height changes, since either can change how
// much of that chrome wraps onto extra lines.
func (m *ShortcutPane) applySizing() {
	helpWidth, helpHeight := m.contentArea(helpFooterText)
	// The Help surface's own category tab bar (see renderTabs) sits above
	// the shortcut list, inside the content area contentArea already
	// carved out - wrapped at the same width, since a narrow enough modal
	// wraps it onto more than one line too, and the list only gets
	// whatever's left after it (plus the blank line separating them).
	tabBarHeight := lipgloss.Height(lipgloss.NewStyle().Width(helpWidth).Render(m.renderTabs()))
	m.helpViewport.SetWidth(helpWidth)
	m.helpViewport.SetHeight(max(helpHeight-tabBarHeight-1, 1))

	// The Groups surface's own footer changes with its mode (a create/
	// rename/edit form's hint is shorter than browse's own), but browse's
	// is always the longest - using whatever's current is safe: modes are
	// transient, so by the time a real resize lands, it's back in browse
	// almost always, and every other mode only ever needs *less* room.
	groupsWidth, groupsHeight := m.contentArea(m.groupsPane.FooterHint())
	m.groupsPane.SetWidth(groupsWidth)
	m.groupsPane.SetHeight(groupsHeight)
}

// contentArea returns the width/height actually left over for a surface's
// own content, after the outer modal's border+padding (see View's
// modalStyle) and the title/surface-tabs/footer chrome every surface
// shares - each measured at its own *wrapped* height, not assumed to be
// exactly one line, since a long footer hint or a narrow terminal can
// wrap either onto more than one line. Treating them as fixed single
// lines silently overflowed the outer modal's declared height (clipping
// its bottom border) instead of leaving an accurate remaining height for
// whatever's rendered below them.
func (m ShortcutPane) contentArea(footerText string) (width, height int) {
	const horizontalFrame = 6 // modalStyle's Border(2) + Padding(0,2)*2(4)
	const verticalFrame = 4   // modalStyle's Border(2) + Padding(1,0)*2(2)
	width = max(m.width-horizontalFrame, 1)

	wrappedHeight := func(s string) int {
		return lipgloss.Height(lipgloss.NewStyle().Width(width).Render(s))
	}
	fixed := 1 + 1 + wrappedHeight(m.renderSurfaceTabs()) + 1 + 1 + wrappedHeight(footerText)
	height = max(m.height-verticalFrame-fixed, 1)
	return width, height
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
