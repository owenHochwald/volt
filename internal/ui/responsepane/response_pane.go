package responsepane

import (
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/authjar"
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

	// SentRequest is the actual, fully-resolved request that produced
	// Response (path/body params substituted, server headers merged in,
	// etc. - see RequestPane.resolvedRequest) - shown on the Request tab so
	// what was actually sent over the wire is inspectable, not just what's
	// still being edited in the request pane.
	SentRequest *http.Request

	latencySamples    []time.Duration
	loadTestStart     time.Time
	previousCompleted int
	previousDuration  time.Duration

	viewport  viewport.Model
	activeTab int

	// rawBody, toggled by ToggleRawBody ("r"), switches the Body tab
	// between the normal formatted/highlighted rendering
	// (formatContentByType) and the response body exactly as received -
	// see updateViewportForActiveTab's TabBody case.
	rawBody bool

	// wrapBody, toggled by ToggleWrap ("w"), switches long lines between
	// wrapping (see setViewportContent's use of ansi.Wrap) and the
	// viewport's own default behavior otherwise - truncating them to the
	// pane's width with horizontal scroll available (see bubbles
	// viewport's own Cut-based visibleLines, driven by ScrollLeft/
	// ScrollRight) - off by default, preserving that existing behavior for
	// anyone who hasn't asked for wrapping.
	wrapBody bool

	// savingToFile/filenameInput drive the "save raw body to file"
	// prompt (see SaveResponseRaw ("s")) - a mode+textinput overlay
	// mirroring groupspane's create/rename prompt. saveStatusMsg/
	// saveStatusErr report the outcome (success or failure) of the last
	// save attempt, shown until the next one replaces it.
	savingToFile  bool
	filenameInput textinput.Model
	saveStatusMsg string
	saveStatusErr bool

	// currentContent is whatever's currently in the viewport - kept
	// alongside it (see setViewportContent) since viewport.Model doesn't
	// expose its own content back out, and search (searchMatchLines) needs
	// the raw string to search against, exactly as displayed (raw or
	// pretty, whichever tab) - see search.go.
	currentContent string

	// Search (search.go): searching/searchInput drive the "/" prompt, the
	// same mode+textinput overlay pattern as savingToFile/filenameInput
	// above. searchQuery/searchMatches/searchMatchIdx hold the last
	// confirmed search's state (matches are individual occurrences into
	// currentContent - a line with the query more than once contributes
	// one searchMatch per occurrence, not one for the whole line - in
	// ascending order) so SearchNext/SearchPrev ("n"/"b") can cycle
	// through them without re-searching. searchStatusMsg reports "no
	// matches for ..." the same way saveStatusMsg reports a save outcome.
	searching       bool
	searchInput     textinput.Model
	searchQuery     string
	searchMatches   []searchMatch
	searchMatchIdx  int
	searchStatusMsg string

	// searchOrigin snapshots search state from right before "/" was
	// pressed (see openSearch) - restored verbatim by cancelSearch (esc),
	// and what liveSearch's incremental matching searches forward from.
	searchOrigin searchSnapshot

	// gPending/gotoLineBuffer implement vim/less-style "gg" (go to top) and
	// "g{n}g" (go to line n) - see navigation.go. gPending is armed by a
	// bare "g" keypress (see GotoTop) and cleared by whatever key resolves
	// or cancels the sequence, or after gotoSequenceTimeout of inactivity
	// (see gotoSequence) - otherwise a "g" the user forgot about (or
	// walked away after) would silently keep digit keys captured as a
	// goto-line buffer indefinitely instead of switching tabs the way
	// ActionDirectTab normally would.
	gPending       bool
	gotoLineBuffer string
	gotoSequence   uint64

	// bodyStyleName is the chroma style used for the Body (and sent-
	// request) tab's pretty rendering (see formatContentByType) -
	// defaultBodyStyle ("onedark") unless a different one was ever picked
	// (see style_picker.go's "t" picker) and persisted (see
	// design.SaveBodyStyle/LoadBodyStyle, loaded once at SetupResponsePane).
	bodyStyleName string

	// Style picker (style_picker.go): pickingStyle/styleList drive the "t"
	// picker - a fuzzy-filterable list of every style chroma ships (see
	// chroma/v2/styles.Names), rendered by app/view.go in the request
	// pane's own box (see StylePickerListView) rather than inside this
	// pane - so it never has to fight this pane's own fixed-size box for
	// room the way an earlier version, cramming a live-preview snippet in
	// here too, once did. The live preview instead just reuses this pane's
	// own normal Body rendering (see effectiveBodyStyle), which already
	// knows how to safely handle arbitrarily long real response lines
	// (unlike that snippet, which didn't and could blow out the layout on
	// something like a minified HTML page). Browsing the list only ever
	// changes what effectiveBodyStyle returns, never bodyStyleName itself
	// - so esc cancelling has nothing to undo, and only enter (see
	// confirmStylePicker) actually applies and persists (see
	// design.SaveBodyStyle) the selection. stylePickerWidth/Height are the
	// request pane's own box
	// dimensions (set by app/update.go's WindowSizeMsg handling, the same
	// way requestPane.SetHeight/SetWidth already are - see
	// SetStylePickerSize), since that's the box the list is sized to fill,
	// not this pane's own width/height.
	pickingStyle                        bool
	styleList                           list.Model
	stylePickerWidth, stylePickerHeight int

	// jar remembers cookies and authorization headers per host, in memory
	// only (see internal/authjar) - nil-safe, like DB below, so tests that
	// never call SetJar just get no capture/injection at all rather than
	// a nil-pointer panic. autoRememberCookies/autoRememberAuthHeaders
	// (see SetAutoRememberCookies/SetAutoRememberAuthHeaders) gate whether
	// SetResponse captures automatically on every response - independent
	// of CaptureAuth ("c" - see jar_capture.go), which always captures
	// regardless of these. viewingJar/jarList/jarViewerWidth/Height drive
	// ViewAuthJar ("v" - see jar_viewer.go), the same request-pane-slot
	// overlay pattern pickingStyle/styleList/stylePickerWidth/Height above
	// already use for the style picker.
	jar                             *authjar.Store
	autoRememberCookies             bool
	autoRememberAuthHeaders         bool
	viewingJar                      bool
	jarList                         list.Model
	jarViewerWidth, jarViewerHeight int

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

// IsSavingToFile reports whether the save-to-file filename prompt (see
// SaveResponseRaw) is currently open - checked by app/update.go before
// treating esc as "switch back to the request panel", so esc still closes
// this prompt first (its own, more specific meaning) instead of the
// panel-switch swallowing it before the prompt ever sees it.
func (m *ResponsePane) IsSavingToFile() bool {
	return m.savingToFile
}

// IsSearching reports whether the "/" search prompt (see SearchResponse)
// is currently open - checked by app/update.go for the same reason as
// IsSavingToFile: so esc closes this prompt first instead of the
// panel-switch swallowing it before the prompt ever sees it.
func (m *ResponsePane) IsSearching() bool {
	return m.searching
}

// IsPickingStyle reports whether the body style picker (see
// style_picker.go's "t") is currently open - checked by app/update.go for
// the same reason as IsSavingToFile/IsSearching: so esc closes the
// picker first instead of the panel-switch swallowing it before the
// picker ever sees it.
func (m *ResponsePane) IsPickingStyle() bool {
	return m.pickingStyle
}

// IsGotoPending reports whether a "gg"/"g{n}g" sequence (see navigation.go)
// is currently armed - checked by app/update.go for the same reason as
// IsSavingToFile/IsSearching/IsPickingStyle: so esc reaches this pane's own
// Update (see continueGoto's default case) and cancels the sequence
// instead of the panel-switch/quit-arm swallowing it first, otherwise
// leaving gPending stuck true (still capturing digit keys as a goto-line
// buffer) even after focus has moved elsewhere.
func (m *ResponsePane) IsGotoPending() bool {
	return m.gPending
}

// IsViewingJar reports whether the saved-cookies/auth-headers viewer (see
// ViewAuthJar, "v") is currently open - checked by app/update.go for the
// same reason as IsSavingToFile/IsSearching/IsPickingStyle/IsGotoPending:
// so esc closes the viewer first instead of the panel-switch swallowing
// it before the viewer ever sees it.
func (m *ResponsePane) IsViewingJar() bool {
	return m.viewingJar
}

// SetJar wires the shared in-memory cookie/auth-header jar (see
// internal/authjar) into this pane - called once from app.SetupModel with
// the same *authjar.Store handed to requestpane.RequestPane.SetJar, so
// what's captured here is what actually gets injected into future
// requests. Left nil (the zero value) in tests that never call this is
// safe - every authjar.Store method is nil-receiver-safe.
func (m *ResponsePane) SetJar(jar *authjar.Store) {
	m.jar = jar
}

// SetAutoRememberCookies/SetAutoRememberAuthHeaders gate whether
// SetResponse captures a response's Set-Cookie headers / the sent
// request's Authorization header automatically on every response, or
// only when the user explicitly presses CaptureAuth ("c") - see
// jar_capture.go. Called once at startup with the persisted setting (see
// design.LoadAutoRememberCookies/LoadAutoRememberAuthHeaders) and again
// every time it's toggled from the Command Center's Settings surface.
func (m *ResponsePane) SetAutoRememberCookies(enabled bool) {
	m.autoRememberCookies = enabled
}

func (m *ResponsePane) SetAutoRememberAuthHeaders(enabled bool) {
	m.autoRememberAuthHeaders = enabled
}

func (m *ResponsePane) SetStyles(styles design.Styles) {
	m.styles = styles
	if m.isLoadTest {
		m.updateViewportForActiveTab()
	}
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
	m.resetLoadTestSeries(time.Time{})
	if m.autoRememberCookies {
		m.captureCookies()
	}
	if m.autoRememberAuthHeaders {
		m.captureAuthHeader()
	}
	m.updateViewportForActiveTab()
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
	// Viewport needs to be smaller to account for the status bar, tab
	// bar, and the pinned footer hint renderNormalView always appends
	// below it (see footerHint) - the scroll indicator and save-status
	// line are each only sometimes present, so they aren't chased
	// precisely here; View()'s own MaxHeight clip is the backstop against
	// those (or this estimate) ever pushing the pane's rendered height
	// past what's actually been allocated for it, which - unlike a plain
	// lipgloss.Height() - only pads short content and would otherwise
	// silently corrupt the whole layout around it.
	m.viewport.SetHeight(max(m.height-4, 1))
}

// SetWidth sets the width of the response pane
func (m *ResponsePane) SetWidth(width int) {
	m.width = max(width, 1)
	m.viewport.SetWidth(m.width)
}

// SetStylePickerSize sizes the style picker's list (see
// StylePickerListView) to width/height - the request pane's own box
// dimensions (app/view.go renders the list there while picking, not in
// this pane's own box - see IsPickingStyle), called from app/update.go's
// WindowSizeMsg handling the same way requestPane.SetHeight/SetWidth
// already are. Stored regardless of whether the picker is open right
// now, so openStylePicker has a real size to start from instead of
// whatever the zero value would size the list to.
func (m *ResponsePane) SetStylePickerSize(width, height int) {
	m.stylePickerWidth = width
	m.stylePickerHeight = height
	if m.pickingStyle {
		m.resizeStyleList()
	}
}

// SetJarViewerSize sizes the saved-cookies/auth-headers viewer's list (see
// JarViewerListView) the same way SetStylePickerSize sizes the style
// picker's - the request pane's own box dimensions, called from
// app/update.go's WindowSizeMsg handling.
func (m *ResponsePane) SetJarViewerSize(width, height int) {
	m.jarViewerWidth = width
	m.jarViewerHeight = height
	if m.viewingJar {
		m.resizeJarList()
	}
}

func (m *ResponsePane) resetLoadTestSeries(start time.Time) {
	m.latencySamples = nil
	m.loadTestStart = start
	m.previousCompleted = 0
	m.previousDuration = 0
}
