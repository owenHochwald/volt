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
