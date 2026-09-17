package responsepane

import (
	"github.com/charmbracelet/bubbles/textinput"
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

	// rawBody, toggled by ToggleRawBody ("r"), switches the Body tab
	// between the normal formatted/highlighted rendering
	// (formatContentByType) and the response body exactly as received -
	// see updateViewportForActiveTab's TabBody case.
	rawBody bool

	// savingToFile/filenameInput drive the "save raw body to file"
	// prompt (see SaveResponseRaw ("s")) - a mode+textinput overlay
	// mirroring groupspane's create/rename prompt. saveStatusMsg/
	// saveStatusErr report the outcome (success or failure) of the last
	// save attempt, shown until the next one replaces it.
	savingToFile  bool
	filenameInput textinput.Model
	saveStatusMsg string
	saveStatusErr bool

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

// IsSavingToFile reports whether the save-to-file filename prompt (see
// SaveResponseRaw) is currently open - checked by app/update.go before
// treating esc as "switch back to the request panel", so esc still closes
// this prompt first (its own, more specific meaning) instead of the
// panel-switch swallowing it before the prompt ever sees it.
func (m *ResponsePane) IsSavingToFile() bool {
	return m.savingToFile
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
	// Viewport needs to be smaller to account for the status bar, blank
	// separator, tab bar, and the pinned footer hint renderNormalView
	// always appends below it (see footerHint) - the scroll indicator and
	// save-status line are each only sometimes present, so they aren't
	// chased precisely here; View()'s own MaxHeight clip is the backstop
	// against those (or this estimate) ever pushing the pane's rendered
	// height past what's actually been allocated for it, which - unlike a
	// plain lipgloss.Height() - only pads short content and would
	// otherwise silently corrupt the whole layout around it.
	m.viewport.Height = max(height-6, 1)
}

// SetWidth sets the width of the response pane
func (m *ResponsePane) SetWidth(width int) {
	m.width = width
	m.viewport.Width = width
}
