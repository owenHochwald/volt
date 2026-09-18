package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ui"
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

	// If a modal is open, overlay it on top (only one can be open at once).
	switch {
	case m.showHelpModal:
		content = m.overlayHelpModal()
	case m.showGroupsModal:
		content = m.overlayModal(m.groupsPane.View())
	case m.showImportModal:
		content = m.overlayModal(m.importPane.View())
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
	sidebar := renderPanel(
		ui.SidebarStyle,
		m.focusedPanel == utils.SidebarPanel,
		layout.sidebarWidth,
		layout.contentHeight,
		m.sidebarPane.View(),
	)

	response := renderPanel(
		ui.ResponseStyle,
		m.focusedPanel == utils.ResponsePanel,
		layout.mainWidth,
		layout.responseHeight,
		m.responsePane.View(),
	)

	// Zoomed (see zoomedResponse/ToggleZoomResponse), the response panel
	// is the only box in the right column - the request panel isn't
	// rendered at all, not just shrunk to nothing, since even a
	// zero-height bordered box would still draw its own border lines.
	// Picking a body style (see below) always un-zooms first (see
	// update.go's PickBodyStyle handling), so this and that are mutually
	// exclusive in practice - the picker takes priority here too, in case
	// that invariant ever changes, since it needs the request slot to
	// show its list in.
	var rightSide string
	if m.zoomedResponse && !m.responsePane.IsPickingStyle() {
		rightSide = response
	} else {
		var request string
		switch {
		case m.responsePane.IsPickingStyle():
			// The style picker's fuzzy-filterable list (see PickBodyStyle)
			// goes here, in the request pane's own slot, instead of the
			// request pane itself - response (already rendered above)
			// keeps showing the real response content alongside it,
			// live-previewing whatever style the list's cursor is on (see
			// responsepane.ResponsePane.effectiveBodyStyle).
			request = renderPanel(ui.RequestStyle, true, layout.mainWidth, layout.requestHeight, m.responsePane.StylePickerListView())
		case m.requestPane.LoadTestMode:
			// Load test style already has yellow border, background, and
			// bold - don't apply the usual focus highlight on top of it.
			request = renderPanel(ui.LoadTestBorderStyle, false, layout.mainWidth, layout.requestHeight, m.requestPane.View())
		default:
			request = renderPanel(ui.RequestStyle, m.focusedPanel == utils.RequestPanel, layout.mainWidth, layout.requestHeight, m.requestPane.View())
		}
		rightSide = lipgloss.JoinVertical(lipgloss.Left, request, response)
	}

	panels := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightSide)
	return lipgloss.JoinVertical(lipgloss.Left, header, panels, m.statusView(layout.width))
}

func (m Model) focusedView(layout terminalLayout, header string) string {
	tabs := m.panelTabs(layout.width)
	var panel string
	switch m.focusedPanel {
	case utils.RequestPanel:
		style := ui.RequestStyle
		if m.requestPane.LoadTestMode {
			style = ui.LoadTestBorderStyle
		}
		panel = renderPanel(style, true, layout.width, layout.contentHeight, m.requestPane.View())
	case utils.ResponsePanel:
		panel = renderPanel(ui.ResponseStyle, true, layout.width, layout.contentHeight, m.responsePane.View())
	default:
		panel = renderPanel(ui.SidebarStyle, true, layout.width, layout.contentHeight, m.sidebarPane.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, tabs, panel, m.statusView(layout.width))
}

func (m Model) statusView(width int) string {
	if m.notification.Text != "" {
		return m.notification.View(width)
	}
	help := m.keys.CompactHelp(m.focusedContext(), 5)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(width).
		MaxWidth(width).
		MaxHeight(1).
		Render(" " + help)
}

func (m Model) headerView(layout terminalLayout) string {
	return renderPanel(
		ui.HeaderStyle,
		false,
		layout.width,
		layout.headerHeight,
		m.headerPane.View(),
	)
}

func (m Model) panelTabs(width int) string {
	names := []string{" SIDEBAR ", " REQUEST ", " RESPONSE "}
	rendered := make([]string, 0, len(names))
	for i, name := range names {
		if utils.Panel(i) == m.focusedPanel {
			rendered = append(rendered, ui.ActiveTab.Render(name))
		} else {
			rendered = append(rendered, ui.InactiveTab.Render(name))
		}
	}
	content := lipgloss.JoinHorizontal(lipgloss.Left, rendered...)
	return lipgloss.NewStyle().Width(width).MaxWidth(width).MaxHeight(1).Render(content)
}

func renderPanel(style lipgloss.Style, focused bool, width, height int, content string) string {
	if focused {
		style = ui.ApplyFocus(style, true)
	}
	contentWidth, contentHeight := contentSize(style, width, height)
	return style.
		Width(contentWidth).
		Height(contentHeight).
		MaxWidth(contentWidth).
		MaxHeight(contentHeight).
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
		lipgloss.WithWhitespaceStyle(
			lipgloss.NewStyle().Foreground(lipgloss.Color("236")),
		),
	)
}
