package app

import (
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ui/shortcutpane"
	"github.com/owenHochwald/Volt/internal/utils"
)

const panelCount = 3

func (m *Model) setFocusedPanel(panel utils.Panel) {
	if panel < utils.SidebarPanel || panel > utils.ResponsePanel {
		return
	}

	m.sidebarPane.SetFocused(panel == utils.SidebarPanel)
	m.requestPane.SetFocused(panel == utils.RequestPanel)
	m.responsePane.SetFocused(panel == utils.ResponsePanel)
	m.focusedPanel = panel
	// Leaving the response panel any way (not just explicitly toggling
	// zoom off - esc, a panel switch, loading a request from the sidebar,
	// a response arriving while zoomed, ...) reliably un-zooms it, so a
	// zoomed response never silently stays zoomed once focus has moved on
	// (see zoomedResponse/ToggleZoomResponse). Un-zooming has to
	// re-apply the layout too, the same way ToggleZoomResponse's own "z"
	// handler already does - otherwise the request pane's own Height
	// stayed stuck at the near-zero value applyLayout gave it while
	// zoomed (currentLayout sets requestHeight to 0 whenever
	// zoomedResponse is true), showing almost nothing of it - just its
	// first field - until some *other* resize happened to come along.
	if panel != utils.ResponsePanel && m.zoomedResponse {
		m.zoomedResponse = false
		m.resizeContentPanels()
	}
}

func (m *Model) movePanel(delta int) {
	next := (int(m.focusedPanel) + delta + panelCount) % panelCount
	m.setFocusedPanel(utils.Panel(next))
}

func (m Model) focusedContext() keybindings.Context {
	switch m.focusedPanel {
	case utils.RequestPanel:
		return keybindings.ContextRequest
	case utils.ResponsePanel:
		return keybindings.ContextResponse
	default:
		return keybindings.ContextSidebar
	}
}

func (m *Model) openHelp(context keybindings.Context) {
	m.showHelpModal = true
	m.themeSession = design.ThemeLoadResult{
		Theme:  m.theme,
		Source: m.themeSource,
	}
	m.themeSessionOpen = true
	m.shortcutPane.SetThemeOptions(m.settingsThemeOptions(), m.themeSource)
	m.shortcutPane.SetPreferences(m.autoRememberCookies, m.autoRememberAuthHeaders)
	m.shortcutPane.BeginSession()
	m.shortcutPane.SetContext(context)
	m.shortcutPane.SetFocused(true)
}

// openGroups opens the Command Center directly onto its Groups surface -
// the same modal openHelp opens, just landing somewhere other than Help.
// Kept as ctrl+g's binding so muscle memory from the old standalone
// groups modal still works.
func (m *Model) openGroups() {
	m.showHelpModal = true
	m.themeSession = design.ThemeLoadResult{
		Theme:  m.theme,
		Source: m.themeSource,
	}
	m.themeSessionOpen = true
	m.shortcutPane.SetThemeOptions(m.settingsThemeOptions(), m.themeSource)
	m.shortcutPane.SetPreferences(m.autoRememberCookies, m.autoRememberAuthHeaders)
	m.shortcutPane.BeginSession()
	m.shortcutPane.SetSurface(shortcutpane.SurfaceGroups)
	m.shortcutPane.SetFocused(true)
}

func (m *Model) closeHelp() {
	if m.themeSessionOpen {
		m.applyTheme(m.themeSession.Theme, m.themeSession.Source)
		m.themeSessionOpen = false
	}
	m.showHelpModal = false
	m.shortcutPane.SetFocused(false)
}

func (m *Model) saveThemeSession(theme design.Theme, source string) {
	m.applyTheme(theme, source)
	m.themeSessionOpen = false
	m.showHelpModal = false
	m.shortcutPane.SetFocused(false)
}
