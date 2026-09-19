package app

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

const (
	startupHoldDuration  = 2 * time.Second
	startupFrameDuration = 120 * time.Millisecond
)

type startupAdvanceMsg struct {
	frame int
}

// currentLayout computes the layout for the model's current terminal size,
// then adjusts it for state calculateLayout can't see on its own (a pure
// function of terminal size alone): the startup animation's current frame
// (headerHeight/resizeContentForHeader), whether a load test is running
// (resizeContentForLoadTest), whether the WebSocket pane is docked
// (resizeContentForWSDock), and zoomedResponse (see ToggleZoomResponse,
// "z"), which hands the request column's share to the response instead of
// splitting it - checked last so a zoomed response always wins over
// whatever the load-test/WS-dock split above it computed.
func (m Model) currentLayout() terminalLayout {
	layout := calculateLayout(m.width, m.height)
	if layout.mode == layoutTooSmall {
		return layout
	}

	layout.headerHeight = 2
	if m.startupFrame == ui.HeaderFrameFull {
		layout.headerHeight = 7
	}

	resizeContentForHeader(&layout)
	m.resizeContentForLoadTest(&layout)
	m.resizeContentForWSDock(&layout)

	if m.zoomedResponse {
		layout.requestHeight = 0
		layout.wsHeight = 0
		layout.responseHeight = layout.contentHeight
	}

	return layout
}

func resizeContentForHeader(layout *terminalLayout) {
	layout.contentHeight = max(
		layout.height-layout.headerHeight-layout.tabBarHeight-layout.statusHeight,
		0,
	)
	if layout.mode == layoutWide {
		layout.requestHeight = max(layout.contentHeight*45/100, 1)
		layout.responseHeight = max(layout.contentHeight-layout.requestHeight, 1)
		return
	}
	layout.requestHeight = layout.contentHeight
	layout.responseHeight = layout.contentHeight
}

func (m Model) resizeContentForLoadTest(layout *terminalLayout) {
	if layout.mode != layoutWide || !m.requestPane.LoadTestMode {
		return
	}

	if m.requestPane.RequestInProgress || m.loadTestUpdates != nil {
		layout.requestHeight = clamp(layout.contentHeight*35/100, 8, layout.contentHeight-8)
	} else {
		layout.requestHeight = clamp(layout.contentHeight*60/100, 20, 24)
		layout.requestHeight = min(layout.requestHeight, max(layout.contentHeight-7, 1))
	}
	layout.responseHeight = max(layout.contentHeight-layout.requestHeight, 1)
}

// resizeContentForWSDock splits the wide layout's content column three
// ways - 25% request / 50% response / 25% WebSocket - instead of
// resizeContentForHeader's plain request/response split, whenever the
// WebSocket pane is docked (see Model.wsDocked, "ctrl+t"). Runs after
// resizeContentForLoadTest so docking always wins over the load-test
// split (simpler and more predictable than trying to blend both), and
// does nothing in the focused layout, where a docked WebSocket pane is
// just a fourth full-size tab (see app/view.go's panelTabs/focusedView)
// rather than a box sharing height with siblings.
func (m Model) resizeContentForWSDock(layout *terminalLayout) {
	if layout.mode != layoutWide || !m.wsDocked {
		return
	}
	layout.requestHeight = max(layout.contentHeight*25/100, 1)
	layout.responseHeight = max(layout.contentHeight*50/100, 1)
	layout.wsHeight = max(layout.contentHeight-layout.requestHeight-layout.responseHeight, 1)
}

func startupAdvanceCmd(frame int, motion design.MotionMode) tea.Cmd {
	delay := startupFrameDuration
	if frame == ui.HeaderFrameCompressed {
		delay = startupHoldDuration
	}
	if motion == design.MotionReduced {
		frame = ui.HeaderFrameCompact
		delay = startupHoldDuration
	}
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return startupAdvanceMsg{frame: frame}
	})
}

func (m *Model) setStartupFrame(frame int) {
	frame = min(max(frame, ui.HeaderFrameFull), ui.HeaderFrameCompact)
	if frame <= m.startupFrame {
		return
	}
	m.startupFrame = frame
	m.headerPane.SetStartupFrame(frame)
	m.applyLayout(m.currentLayout())
}
