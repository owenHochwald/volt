package app

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/utils"
)

func (m Model) View() string {
	sidebarWidth := m.sidebarWidth()
	mainHeight := m.mainHeight()

	responseHeight := m.responseHeight()
	responseWidth := m.responseWidth()

	sidebar := m.sidebarView(mainHeight, sidebarWidth)

	response := ui.ApplyFocus(ui.ResponseStyle, m.focusedPanel == 2).Width(responseWidth).
		Height(responseHeight).
		Render(m.responseView(responseHeight, responseWidth))

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
		requestHeight := m.requestHeight()
		requestWidth := m.requestWidth()

		// Conditional rendering for custom request pane border color. Both
		// branches use the same requestHeight (see layout.go) - previously
		// the load-test branch also subtracted an extra, undocumented 3
		// lines from the response box, which (combined with the two boxes'
		// heights never actually summing to the space available beside the
		// sidebar - see rightColumnHeight) was part of why the right column
		// visibly fell short of the sidebar, leaving blank space beneath it.
		var request string
		switch {
		case m.responsePane.IsPickingStyle():
			// The style picker's fuzzy-filterable list (see PickBodyStyle)
			// goes here, in the request pane's own slot, instead of the
			// request pane itself - response (already rendered above)
			// keeps showing the real response content alongside it,
			// live-previewing whatever style the list's cursor is on
			// (see responsepane.ResponsePane.effectiveBodyStyle) - safe
			// against arbitrarily long real response lines the same way
			// response's own normal rendering always has been, unlike an
			// earlier version that crammed a separate preview snippet
			// into this pane's own fixed-size box instead.
			request = ui.ApplyFocus(ui.RequestStyle, true).Width(requestWidth).
				Height(requestHeight).
				Render(m.responsePane.StylePickerListView())
		case m.requestPane.LoadTestMode:
			// Load test style already has yellow border, background, and bold - don't apply focus
			request = ui.LoadTestBorderStyle.Width(requestWidth).
				Height(requestHeight).
				Render(m.requestView(requestHeight, requestWidth))
		default:
			request = ui.ApplyFocus(ui.RequestStyle, m.focusedPanel == 1).Width(requestWidth).
				Height(requestHeight).
				Render(m.requestView(requestHeight, requestWidth))
		}
		rightSide = lipgloss.JoinVertical(lipgloss.Right, request, response)
	}

	bottomPanels := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, rightSide)

	// Hidden (see showHeader/ToggleHeader, "f4"), the header isn't
	// rendered at all - not just shrunk to nothing, since even a
	// zero-height bordered box would still draw its own border lines -
	// and mainHeight/sidebarContentHeight (see layout.go's headerHeight)
	// have already handed the panels below the space it used to occupy.
	var mainView string
	if m.showHeader {
		mainView = lipgloss.JoinVertical(lipgloss.Top, m.headerView(m.width), bottomPanels)
	} else {
		mainView = bottomPanels
	}

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
