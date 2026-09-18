package app

import (
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
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
	// (see zoomedResponse/ToggleZoomResponse).
	if panel != utils.ResponsePanel {
		m.zoomedResponse = false
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
	m.shortcutPane.SetContext(context)
	m.shortcutPane.SetFocused(true)
}

func (m *Model) closeHelp() {
	m.showHelpModal = false
	m.shortcutPane.SetFocused(false)
}
