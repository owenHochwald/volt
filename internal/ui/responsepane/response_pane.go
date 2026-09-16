package responsepane

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// ResponsePane is the component responsible for displaying HTTP responses and load test statistics
type ResponsePane struct {
	Response      *http.Response
	LoadTestStats *http.LoadTestStats
	isLoadTest    bool
	height, width int

	// SentRequest is the actual, fully-resolved request that produced
	// Response (path/body params substituted, server headers merged in,
	// etc. - see RequestPane.resolvedRequest) - shown on the Request tab so
	// what was actually sent over the wire is inspectable, not just what's
	// still being edited in the request pane.
	SentRequest *http.Request

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

// SetResponse updates the response pane with a new HTTP response and the
// request that produced it (see SentRequest). Deliberately leaves activeTab
// as-is (re-sending a request while on the Headers tab, say, stays there)
// rather than forcing it back to Body - updateViewportForActiveTab fills
// the viewport with whatever that tab actually shows, so the two can never
// drift out of sync the way duplicating Body's own content-selection logic
// here once did (see updateViewportForActiveTab's TabRequest case, added
// after Body/Headers/Timing/Request all started needing the same handling
// once every tab became scrollable - see renderActiveTabContent).
func (m *ResponsePane) SetResponse(response *http.Response, sentRequest *http.Request) {
	m.Response = response
	m.SentRequest = sentRequest
	m.isLoadTest = false
	m.updateViewportForActiveTab()
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
