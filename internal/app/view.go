package app

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/utils"
)

func (m Model) View() string {
	sidebarWidth := m.sidebarWidth()
	mainWidth := m.mainWidth()
	mainHeight := m.mainHeight()

	requestHeight := m.requestHeight()
	requestWidth := m.requestWidth()
	responseHeight := int(float64(mainHeight)/2.2) - 2

	sidebar := m.sidebarView(mainHeight, sidebarWidth)

	// Conditional rendering for custom request pane border color
	var request string
	var response string
	if m.requestPane.LoadTestMode {
		// Load test style already has yellow border, background, and bold - don't apply focus
		request = ui.LoadTestBorderStyle.Width(requestWidth).
			Height(requestHeight).
			Render(m.requestView(requestHeight, requestWidth))
		response = ui.ApplyFocus(ui.ResponseStyle, m.focusedPanel == 2).Width(mainWidth - 10).
			Height(responseHeight - 3).
			Render(m.responseView(responseHeight, mainWidth-10))
	} else {
		request = ui.ApplyFocus(ui.RequestStyle, m.focusedPanel == 1).Width(requestWidth).
			Height(requestHeight).
			Render(m.requestView(requestHeight, requestWidth))
		response = ui.ApplyFocus(ui.ResponseStyle, m.focusedPanel == 2).Width(mainWidth - 10).
			Height(responseHeight).
			Render(m.responseView(responseHeight, mainWidth-10))
	}

	rightSide := lipgloss.JoinVertical(lipgloss.Right, request, response)
	bottomPanels := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightSide)
	mainView := lipgloss.JoinVertical(lipgloss.Top, m.headerView(m.width), bottomPanels)

	// If a modal is open, overlay it on top (only one can be open at once)
	if m.showHelpModal {
		return m.overlayHelpModal()
	}
	if m.showGroupsModal {
		return m.overlayModal(m.groupsPane.View())
	}
	if m.showImportModal {
		return m.overlayModal(m.importPane.View())
	}

	return mainView
}

func (m Model) headerView(width int) string {
	header := ui.HeaderStyle.Width(width).Render(m.headerPane.View())
	return header
}

func (m Model) sidebarView(height, width int) string {
	sidebar := ui.ApplyFocus(ui.SidebarStyle, m.focusedPanel == 0).Width(width).
		Height(height - 4).
		Render(m.sidebarPane.View())
	return sidebar
}

func (m Model) requestView(height, width int) string {
	m.requestPane.SetFocused(m.focusedPanel == utils.RequestPanel)
	//m.requestPane.LoadTestMode
	m.requestPane.SetHeight(height)
	m.requestPane.SetWidth(width)

	return m.requestPane.View()
}

func (m Model) responseView(height, width int) string {
	m.responsePane.SetHeight(height)
	m.responsePane.SetWidth(width)

	return m.responsePane.View()
}

// overlayHelpModal renders the help modal centered over the main view
func (m Model) overlayHelpModal() string {
	return m.overlayModal(m.shortcutPane.View())
}

// overlayModal centers any modal's rendered content over a dimmed backdrop
// covering the whole terminal, matching the help modal's presentation.
func (m Model) overlayModal(modal string) string {
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}
