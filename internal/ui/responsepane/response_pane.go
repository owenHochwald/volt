package responsepane

import (
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

const maxLatencySamples = 32

// ResponsePane is the component responsible for displaying HTTP responses and load test statistics
type ResponsePane struct {
	Response      *http.Response
	LoadTestStats *http.LoadTestStats
	isLoadTest    bool
	height, width int

	latencySamples    []time.Duration
	loadTestStart     time.Time
	previousCompleted int
	previousDuration  time.Duration

	viewport  viewport.Model
	activeTab int

	keys   keybindings.KeyMap
	styles design.Styles
}

// Init initializes the response pane
func (m ResponsePane) Init() tea.Cmd {
	return nil
}

// SetFocused sets the focused state of the response pane
func (m *ResponsePane) SetFocused(focused bool) {
	// Response pane doesn't currently use focus state, but implements interface for consistency
}

func (m *ResponsePane) SetStyles(styles design.Styles) {
	m.styles = styles
	if m.isLoadTest {
		m.updateViewportForActiveTab()
	}
}

// SetResponse updates the response pane with a new HTTP response
func (m *ResponsePane) SetResponse(response *http.Response) {
	m.Response = response
	m.isLoadTest = false
	m.resetLoadTestSeries(time.Time{})

	if m.Response != nil {
		if m.Response.Error != "" {
			m.viewport.SetContent(m.Response.Error)
			return
		}

		contentType := m.Response.ParseContentType()
		content := formatContentByType(m.Response.Body, contentType)
		m.viewport.SetContent(content)
	}
}

// SetLoadTestStats updates the response pane with load test statistics
func (m *ResponsePane) SetLoadTestStats(stats *http.LoadTestStats) {
	if stats == nil {
		return
	}
	newRun := !m.isLoadTest ||
		!stats.StartTime.Equal(m.loadTestStart) ||
		stats.CompletedRequests < m.previousCompleted
	if newRun {
		m.resetLoadTestSeries(stats.StartTime)
		m.activeTab = int(TabLoadTestOverview)
	}

	completedDelta := stats.CompletedRequests - m.previousCompleted
	durationDelta := stats.TotalDuration - m.previousDuration
	if completedDelta > 0 && durationDelta >= 0 {
		m.latencySamples = append(
			m.latencySamples,
			durationDelta/time.Duration(completedDelta),
		)
		if len(m.latencySamples) > maxLatencySamples {
			m.latencySamples = m.latencySamples[len(m.latencySamples)-maxLatencySamples:]
		}
	}
	m.previousCompleted = stats.CompletedRequests
	m.previousDuration = stats.TotalDuration
	m.LoadTestStats = stats
	m.isLoadTest = true
	m.updateViewportForActiveTab()
}

// ClearLoadTestStats clears load test data and switches back to normal mode
func (m *ResponsePane) ClearLoadTestStats() {
	m.LoadTestStats = nil
	m.isLoadTest = false
	m.resetLoadTestSeries(time.Time{})
}

func (m *ResponsePane) SetLoadTestPending(totalRequests int) {
	m.SetLoadTestStats(http.NewLoadTestStats(totalRequests))
}

// SetHeight sets the height of the response pane
func (m *ResponsePane) SetHeight(height int) {
	m.height = max(height, 1)
	// Viewport needs to be smaller to account for status bar, tabs, etc.
	m.viewport.SetHeight(max(m.height-5, 1))
}

// SetWidth sets the width of the response pane
func (m *ResponsePane) SetWidth(width int) {
	m.width = max(width, 1)
	m.viewport.SetWidth(m.width)
}

func (m *ResponsePane) resetLoadTestSeries(start time.Time) {
	m.latencySamples = nil
	m.loadTestStart = start
	m.previousCompleted = 0
	m.previousDuration = 0
}
