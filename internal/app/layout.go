package app

import (
	"charm.land/lipgloss/v2"
)

const (
	minTerminalWidth   = 60
	minTerminalHeight  = 20
	wideTerminalWidth  = 110
	wideTerminalHeight = 30
)

type layoutMode uint8

const (
	layoutTooSmall layoutMode = iota
	layoutFocused
	layoutWide
)

type terminalLayout struct {
	mode layoutMode

	width  int
	height int

	headerHeight  int
	tabBarHeight  int
	statusHeight  int
	contentHeight int

	sidebarWidth   int
	mainWidth      int
	requestHeight  int
	responseHeight int

	modalWidth  int
	modalHeight int
}

func calculateLayout(width, height int) terminalLayout {
	layout := terminalLayout{
		width:       max(width, 0),
		height:      max(height, 0),
		modalWidth:  clamp(width-8, 30, 72),
		modalHeight: clamp(height-4, 12, 30),
	}
	if width < minTerminalWidth || height < minTerminalHeight {
		layout.mode = layoutTooSmall
		layout.modalWidth = max(min(width, layout.modalWidth), 0)
		layout.modalHeight = max(min(height, layout.modalHeight), 0)
		return layout
	}

	if width >= wideTerminalWidth && height >= wideTerminalHeight {
		layout.mode = layoutWide
		layout.headerHeight = 8
		layout.statusHeight = 1
		layout.contentHeight = max(height-layout.headerHeight-layout.statusHeight, 0)
		layout.sidebarWidth = clamp(width/4, 24, 34)
		layout.mainWidth = max(width-layout.sidebarWidth, 0)
		layout.requestHeight = max(layout.contentHeight*45/100, 1)
		layout.responseHeight = max(layout.contentHeight-layout.requestHeight, 1)
		return layout
	}

	layout.mode = layoutFocused
	layout.headerHeight = 3
	layout.tabBarHeight = 1
	layout.statusHeight = 1
	layout.contentHeight = max(height-layout.headerHeight-layout.tabBarHeight-layout.statusHeight, 0)
	layout.sidebarWidth = width
	layout.mainWidth = width
	layout.requestHeight = layout.contentHeight
	layout.responseHeight = layout.contentHeight
	return layout
}

func clamp(value, minimum, maximum int) int {
	if maximum < minimum {
		return minimum
	}
	return min(max(value, minimum), maximum)
}

func (m *Model) applyLayout(layout terminalLayout) {
	m.headerPane.SetSize(layout.width)
	m.shortcutPane.SetWidth(layout.modalWidth)
	m.shortcutPane.SetHeight(layout.modalHeight)

	if layout.mode == layoutTooSmall {
		return
	}

	if layout.mode == layoutWide {
		sidebarWidth, sidebarHeight := contentSize(m.styles.Panel.Sidebar, layout.sidebarWidth, layout.contentHeight)
		requestWidth, requestHeight := contentSize(m.styles.Panel.Base, layout.mainWidth, layout.requestHeight)
		responseWidth, responseHeight := contentSize(m.styles.Panel.Base, layout.mainWidth, layout.responseHeight)
		m.sidebarPane.SetSize(sidebarWidth, sidebarHeight)
		m.requestPane.SetSize(requestWidth, requestHeight)
		m.responsePane.SetWidth(responseWidth)
		m.responsePane.SetHeight(responseHeight)
		return
	}

	sidebarWidth, sidebarHeight := contentSize(m.styles.Panel.Sidebar, layout.width, layout.contentHeight)
	requestWidth, requestHeight := contentSize(m.styles.Panel.Base, layout.width, layout.contentHeight)
	responseWidth, responseHeight := contentSize(m.styles.Panel.Base, layout.width, layout.contentHeight)
	m.sidebarPane.SetSize(sidebarWidth, sidebarHeight)
	m.requestPane.SetSize(requestWidth, requestHeight)
	m.responsePane.SetWidth(responseWidth)
	m.responsePane.SetHeight(responseHeight)
}

func contentSize(style lipgloss.Style, outerWidth, outerHeight int) (int, int) {
	return max(outerWidth-style.GetHorizontalFrameSize(), 1),
		max(outerHeight-style.GetVerticalFrameSize(), 1)
}
