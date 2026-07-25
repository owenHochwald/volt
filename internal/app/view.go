package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/utils"
)

func (m Model) View() tea.View {
	layout := m.currentLayout()
	if layout.mode == layoutTooSmall {
		return newView(m.tooSmallView(layout))
	}

	header := m.headerView(layout)
	var content string
	if layout.mode == layoutWide {
		content = m.wideView(layout, header)
	} else {
		content = m.focusedView(layout, header)
	}

	if m.showHelpModal {
		content = m.overlayHelpModal()
	}
	return newView(content)
}

func newView(content string) tea.View {
	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "Volt"
	return view
}

func (m Model) wideView(layout terminalLayout, header string) string {
	sidebar := m.renderPanel(
		m.styles.Panel.Sidebar,
		m.focusedPanel == utils.SidebarPanel,
		false,
		layout.sidebarWidth,
		layout.contentHeight,
		m.sidebarPane.View(),
	)

	request := m.renderPanel(
		m.styles.Panel.Base,
		m.focusedPanel == utils.RequestPanel && !m.requestPane.LoadTestMode,
		m.requestPane.RequestInProgress,
		layout.mainWidth,
		layout.requestHeight,
		m.requestPane.View(),
	)
	response := m.renderPanel(
		m.styles.Panel.Base,
		m.focusedPanel == utils.ResponsePanel,
		false,
		layout.mainWidth,
		layout.responseHeight,
		m.responsePane.View(),
	)

	rightSide := lipgloss.JoinVertical(lipgloss.Left, request, response)
	panels := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightSide)
	return lipgloss.JoinVertical(lipgloss.Left, header, panels, m.statusView(layout.width))
}

func (m Model) focusedView(layout terminalLayout, header string) string {
	tabs := m.panelTabs(layout.width)
	var panel string
	switch m.focusedPanel {
	case utils.RequestPanel:
		panel = m.renderPanel(
			m.styles.Panel.Base,
			true,
			m.requestPane.RequestInProgress,
			layout.width,
			layout.contentHeight,
			m.requestPane.View(),
		)
	case utils.ResponsePanel:
		panel = m.renderPanel(m.styles.Panel.Base, true, false, layout.width, layout.contentHeight, m.responsePane.View())
	default:
		panel = m.renderPanel(m.styles.Panel.Sidebar, true, false, layout.width, layout.contentHeight, m.sidebarPane.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, panel, m.statusView(layout.width))
}

func (m Model) statusView(width int) string {
	if m.notification.Text != "" {
		return m.notification.View(width, m.styles)
	}
	help := m.keys.CompactHelp(m.focusedContext(), 5)
	return m.styles.Text.Muted.
		Width(width).
		MaxWidth(width).
		MaxHeight(1).
		Render(" " + help)
}

func (m Model) headerView(layout terminalLayout) string {
	activePanel := "SIDEBAR"
	switch m.focusedPanel {
	case utils.RequestPanel:
		activePanel = "REQUEST"
	case utils.ResponsePanel:
		activePanel = "RESPONSE"
	}
	mode := "NORMAL"
	if m.requestPane.LoadTestMode || m.loadTestUpdates != nil {
		mode = "LOAD TEST"
	}
	m.headerPane.SetContext(activePanel, mode)
	return lipgloss.NewStyle().
		Width(layout.width).
		Height(layout.headerHeight).
		MaxWidth(layout.width).
		MaxHeight(layout.headerHeight).
		Render(m.headerPane.View())
}

func (m Model) panelTabs(width int) string {
	names := []string{" SIDEBAR ", " REQUEST ", " RESPONSE "}
	rendered := make([]string, 0, len(names))
	for i, name := range names {
		if utils.Panel(i) == m.focusedPanel {
			rendered = append(rendered, m.styles.Tabs.Active.Render(name))
		} else {
			rendered = append(rendered, m.styles.Tabs.Inactive.Render(name))
		}
	}
	content := lipgloss.JoinHorizontal(lipgloss.Left, rendered...)
	return lipgloss.NewStyle().Width(width).MaxWidth(width).MaxHeight(1).Render(content)
}

func (m Model) renderPanel(style lipgloss.Style, focused, running bool, width, height int, content string) string {
	style = m.styles.Panel.Apply(style, focused, running)
	contentWidth, contentHeight := contentSize(style, width, height)
	content = lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		MaxWidth(contentWidth).
		MaxHeight(contentHeight).
		Render(content)
	return style.
		Width(contentWidth).
		Height(contentHeight).
		Render(content)
}

func (m Model) tooSmallView(layout terminalLayout) string {
	if layout.width == 0 || layout.height == 0 {
		return ""
	}
	message := fmt.Sprintf(
		"Volt needs at least %d×%d\nCurrent terminal: %d×%d",
		minTerminalWidth,
		minTerminalHeight,
		layout.width,
		layout.height,
	)
	message = lipgloss.NewStyle().
		Width(layout.width).
		MaxWidth(layout.width).
		Align(lipgloss.Center).
		Render(message)
	return lipgloss.Place(
		layout.width,
		layout.height,
		lipgloss.Center,
		lipgloss.Center,
		message,
	)
}

func (m Model) overlayHelpModal() string {
	helpModal := m.shortcutPane.View()
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		helpModal,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceStyle(
			lipgloss.NewStyle().Foreground(m.theme.Colors.SurfaceRaised),
		),
	)
}
