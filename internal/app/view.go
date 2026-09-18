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
	sidebar := m.renderPanel(
		m.styles.Panel.Sidebar,
		m.focusedPanel == utils.SidebarPanel,
		false,
		layout.sidebarWidth,
		layout.contentHeight,
		m.sidebarPane.View(),
	)

	response := m.renderPanel(
		m.styles.Panel.Base,
		m.focusedPanel == utils.ResponsePanel,
		false,
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
			request = m.renderPanel(m.styles.Panel.Base, true, false, layout.mainWidth, layout.requestHeight, m.responsePane.StylePickerListView())
		case m.requestPane.LoadTestMode:
			// Load test mode gets its own fixed yellow border regardless
			// of focus or in-progress state, not the usual focus/running
			// highlight - see renderLoadTestPanel.
			request = m.renderLoadTestPanel(layout.mainWidth, layout.requestHeight, m.requestPane.View())
		default:
			request = m.renderPanel(
				m.styles.Panel.Base,
				m.focusedPanel == utils.RequestPanel,
				m.requestPane.RequestInProgress,
				layout.mainWidth,
				layout.requestHeight,
				m.requestPane.View(),
			)
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
		if m.requestPane.LoadTestMode {
			panel = m.renderLoadTestPanel(layout.width, layout.contentHeight, m.requestPane.View())
		} else {
			panel = m.renderPanel(
				m.styles.Panel.Base,
				true,
				m.requestPane.RequestInProgress,
				layout.width,
				layout.contentHeight,
				m.requestPane.View(),
			)
		}
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
	return m.renderPanel(
		m.styles.Panel.Header,
		false,
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
	// The outer style's own Width/Height (which draws the border around
	// content already sized to contentWidth/contentHeight above) still
	// needs its own MaxWidth/MaxHeight here too - without them, re-flowing
	// content that's already exactly contentWidth cells wide through a
	// second Width() call can occasionally push a trailing word (e.g. a
	// bullet in the footer's help hint) onto an extra line, one row taller
	// than contentHeight reserved for it.
	return style.
		Width(contentWidth).
		Height(contentHeight).
		MaxWidth(width).
		MaxHeight(height).
		Render(content)
}

// renderLoadTestPanel renders the request pane while in load-test mode
// with a fixed yellow border, independent of focus or in-progress state -
// unlike renderPanel's style, which m.styles.Panel.Apply always repaints
// with the focused/running border color, this mode has its own distinct
// visual language and keeps its border regardless.
func (m Model) renderLoadTestPanel(width, height int, content string) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("226")). // Yellow
		Bold(true)
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
		MaxWidth(width).
		MaxHeight(height).
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
			lipgloss.NewStyle().Foreground(m.theme.Colors.SurfaceRaised),
		),
	)
}
