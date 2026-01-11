package responsepane

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/volt/internal/http"
	"github.com/owenHochwald/volt/internal/ui/keybindings"
)

// ResponsePane is the component responsible for displaying HTTP responses and load test statistics
type ResponsePane struct {
	Response      *http.Response
	LoadTestStats *http.LoadTestStats
	isLoadTest    bool
	height, width int

	viewport  viewport.Model
	activeTab int

	keys keybindings.KeyMap
}

// Init initializes the response pane
func (m ResponsePane) Init() tea.Cmd {
	return nil
}

// SetFocused sets the focused state of the response pane
func (m *ResponsePane) SetFocused(focused bool) {
	// Response pane doesn't currently use focus state, but implements interface for consistency
}

// SetResponse updates the response pane with a new HTTP response
func (m *ResponsePane) SetResponse(response *http.Response) {
	m.Response = response
	m.isLoadTest = false

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
	m.LoadTestStats = stats
	m.isLoadTest = true
	m.activeTab = int(TabLoadTestOverview) // Reset to Overview tab
	m.updateViewportForActiveTab()
}

// ClearLoadTestStats clears load test data and switches back to normal mode
func (m *ResponsePane) ClearLoadTestStats() {
	m.LoadTestStats = nil
	m.isLoadTest = false
}

// SetHeight sets the height of the response pane
func (m *ResponsePane) SetHeight(height int) {
	m.height = height
	// Viewport needs to be smaller to account for status bar, tabs, etc.
	m.viewport.Height = height - 5
}

// SetWidth sets the width of the response pane
func (m *ResponsePane) SetWidth(width int) {
	m.width = width
	m.viewport.Width = width
}
