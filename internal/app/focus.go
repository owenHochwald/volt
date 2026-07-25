package app

import (
	"github.com/owenHochwald/Volt/internal/ui/design"
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
	m.shortcutPane.BeginSession()
	m.shortcutPane.SetContext(context)
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
