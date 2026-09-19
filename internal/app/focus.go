package app

import (
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ui/shortcutpane"
	"github.com/owenHochwald/Volt/internal/utils"
)

// panelCount returns how many panels ctrl+w/ctrl+e (see movePanel) cycle
// through - 4 (including utils.WSPanel) only while the WebSocket pane is
// docked, since it isn't a focusable panel at all otherwise (see
// utils.WSPanel's own doc).
func (m Model) panelCount() int {
	if m.wsDocked {
		return 4
	}
	return 3
}

func (m *Model) setFocusedPanel(panel utils.Panel) {
	maxPanel := utils.ResponsePanel
	if m.wsDocked {
		maxPanel = utils.WSPanel
	}
	if panel < utils.SidebarPanel || panel > maxPanel {
		return
	}

	m.sidebarPane.SetFocused(panel == utils.SidebarPanel)
	m.requestPane.SetFocused(panel == utils.RequestPanel)
	m.responsePane.SetFocused(panel == utils.ResponsePanel)
	m.wsPane.SetFocused(panel == utils.WSPanel)
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
	count := m.panelCount()
	next := (int(m.focusedPanel) + delta + count) % count
	m.setFocusedPanel(utils.Panel(next))
}

func (m Model) focusedContext() keybindings.Context {
	switch m.focusedPanel {
	case utils.RequestPanel:
		return keybindings.ContextRequest
	case utils.ResponsePanel:
		return keybindings.ContextResponse
	case utils.WSPanel:
		return keybindings.ContextWebSocket
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

// toggleWSDock switches the WebSocket pane between a floating modal and a
// real fourth panel (see utils.WSPanel, wsDocked). Docking supersedes the
// floating overlay - showing the same session in both places at once
// would be redundant - so docking on always closes it and jumps focus
// straight to the newly-docked panel; undocking, if it was the focused
// panel, falls back to the response panel rather than leaving focus
// pointing at a panel that's no longer part of the cycle.
func (m *Model) toggleWSDock() {
	m.wsDocked = !m.wsDocked
	m.wsPane.SetDocked(m.wsDocked)
	if m.wsDocked {
		m.showWSModal = false
		m.setFocusedPanel(utils.WSPanel)
	} else if m.focusedPanel == utils.WSPanel {
		m.setFocusedPanel(utils.ResponsePanel)
	}
	m.resizeContentPanels()
}

// closeWS hides the WebSocket modal - unlike closeHelp, it deliberately
// doesn't touch wsPane's session or log (see wspane's package doc: esc
// only hides, it never disconnects), so this is safe to call from
// anywhere esc might reach it, including before the modal's own Update
// ever sees the keypress (see update.go's Quit-handling special case).
func (m *Model) closeWS() {
	m.showWSModal = false
	m.wsPane.SetFocused(false)
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
