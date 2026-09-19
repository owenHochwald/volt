package requestpane

import (
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/apperror"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/utils"
)

// Update handles updates to the request pane
func (m RequestPane) Update(msg tea.Msg) (RequestPane, tea.Cmd) {
	var cmd tea.Cmd
	m.Stopwatch, cmd = m.Stopwatch.Update(msg)

	switch msg := msg.(type) {
	case ui.SetRequestPaneRequestMsg:
		m.reinitRequestPane(msg.Request)
		// Unlike reinitRequestPane's own default (method+URL - see
		// defaultFocusIndex), a request picked from the sidebar was
		// picked *for* its method and URL, already visible right there in
		// the list - so focus skips past them to whatever's actually left
		// to fill in (see focusIndexAfterURL).
		m.FocusManager.FocusIndex(m.focusIndexAfterURL())
		m.scrollToFocused()
		return m, nil

	case tea.KeyPressMsg:
		if !m.PanelFocused {
			return m, nil
		}

		// Global shortcuts
		if keybindings.Matches(msg, m.keys.ToggleLoadTest) {
			m.toggleLoadTestMode()
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.SaveRequest) {
			m.syncRequest()
			if err := m.validateRequest(); err != nil {
				return m, ui.NotifyErrorCmd(apperror.FromApplication(err))
			}
			// Snapshot (not m.Request directly) so the currently
			// typed/selected field values (see http.Request.ParamValues)
			// are persisted along with the schema/template - otherwise
			// reopening a saved request would only ever show blank/
			// default fields again, never what was actually filled in.
			snapshot := m.Snapshot()
			return m, ui.SaveRequestCmd(m.DB, &snapshot)
		}
		if keybindings.Matches(msg, m.keys.ResetRequest) {
			m.resetToBlank()
			// Also clears whatever was persisted as "the last sent
			// request" (see ui.SaveLastSentRequestCmd) - otherwise a
			// reset pane would come right back on the next app start,
			// defeating the point of resetting it now.
			if m.DB != nil {
				return m, ui.ClearLastSentRequestCmd(m.DB)
			}
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.ClearParams) {
			m.clearParamValues()
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.NextField) {
			m.FocusManager.Next()
			m.scrollToFocused()
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.PrevField) {
			m.FocusManager.Prev()
			m.scrollToFocused()
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.ScrollFieldsUp) {
			// SetContent first: FieldsViewport.lines are only ever
			// populated inside View() (value receiver - never persisted
			// here), so without this refresh every scroll would see empty
			// content, read maxYOffset()==0, and PageUp/PageDown would be
			// a permanent no-op.
			content, _ := m.fieldsContent()
			m.FieldsViewport.SetContent(content)
			m.FieldsViewport.PageUp()
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.ScrollFieldsDown) {
			content, _ := m.fieldsContent()
			m.FieldsViewport.SetContent(content)
			m.FieldsViewport.PageDown()
			return m, nil
		}

		// Delegate to current mode strategy
		model, cmd := m.currentMode.HandleInput(&m, msg)
		return *model, cmd
	}

	m.syncRequest()
	return m, cmd
}

// toggleLoadTestMode toggles between normal and load test mode
func (m *RequestPane) toggleLoadTestMode() {
	if m.FocusManager != nil {
		m.FocusManager.Current().Blur()
	}

	// Save curr focus index for maintaining positioning
	currentIndex := 0
	if m.FocusManager != nil {
		currentIndex = m.FocusManager.CurrentIndex()
	}

	m.LoadTestMode = !m.LoadTestMode

	if m.LoadTestMode {
		m.currentMode = &LoadTestMode{}
	} else {
		m.currentMode = &NormalMode{}
	}

	// Create new focus manager, preserving index (clamped to valid range)
	m.FocusManager = m.currentMode.GetFocusManagerWithIndex(m, currentIndex)
	// Headers/Body's editorHeight (see SetHeight) branches on LoadTestMode,
	// just flipped above - re-applying the current size recalculates it for
	// the new mode instead of leaving it sized for whichever mode was
	// active when SetHeight was last called from a real resize.
	m.SetWidth(m.Width)
	m.SetHeight(m.Height)
}

// resetToBlank discards whatever request is currently loaded/being
// edited - including every typed field value - and returns the pane to
// the exact same blank, unsaved-request state SetupRequestPane starts the
// app on (see http.NewDefaultRequest). Also drops back to normal mode:
// like a fresh app start, there's no reason a reset request should stay
// in load-test mode.
func (m *RequestPane) resetToBlank() {
	m.LoadTestMode = false
	m.currentMode = &NormalMode{}
	m.RequestInProgress = false
	m.ParseErrors = nil
	m.reinitRequestPane(http.NewDefaultRequest())
}

// reinitRequestPane reinits the request pane with a new request
func (m *RequestPane) reinitRequestPane(request *http.Request) {
	// Blur whatever's currently focused before rebuilding FocusManager
	// below - NameInput/URLInput/etc. are reused across requests, not
	// recreated, so skipping this left the previously focused field
	// still marked focused (cursor still blinking) even after focus
	// moved to method+URL, showing as two simultaneously blinking
	// cursors once a different request was loaded/reset while a field
	// like Name was focused.
	if m.FocusManager != nil {
		m.FocusManager.Current().Blur()
	}
	m.Request = request
	// A request loaded/reset/cleared fresh hasn't had a send attempted
	// against *it* yet - carrying this over from whatever was loaded before
	// would show every still-empty required field as an error before the
	// user has done anything with this request.
	m.ShowRequiredErrors = false

	m.MethodSelector.SetCurrentIndex(request.Method)
	m.URLInput.SetValue(request.URL)
	m.NameInput.SetValue(request.Name)
	m.Headers.SetValue(utils.ParseMapToString(request.Headers))
	m.Body.SetValue(bodyInnerContent(request.Body))

	// Discard any param values from whatever request was loaded before, so
	// switching requests doesn't leak one request's {id} or body field
	// value into another's - only re-editing the *same* request should
	// keep values as you type.
	m.PathParams = nil
	m.QueryParams = nil
	m.BodyParams = nil
	m.ArrayParams = nil
	m.rebuildServerSelector()
	m.syncPathParams()
	m.syncQueryParams()
	m.syncBodyParams()
	// Restores whatever was typed/selected into request's fields the last
	// time it was saved/sent (see http.Request.ParamValues) - after
	// syncPathParams/syncQueryParams/syncBodyParams have (re)built the
	// fields to apply them onto, and before the rebuildFocusManager below,
	// since applying array item counts can itself change the focus
	// cycle's shape.
	m.applyParamValues(request.ParamValues)
	// Every reinitRequestPane call starts a request fresh - not just
	// rebuildFocusManager's usual "keep focus where it currently is"
	// (used when a request's own field set merely changes shape, e.g. an
	// array item added) - so focus always lands on method+URL (see
	// defaultFocusIndex), regardless of whatever was focused in whatever
	// was loaded before. This also covers what rebuildFocusManager exists
	// for elsewhere: rebuildServerSelector always replaces
	// m.ServerSelector with a fresh instance, so FocusManager must be
	// rebuilt against it too, or cycling it (h/l) would silently do
	// nothing.
	m.FocusManager = m.currentMode.GetFocusManagerWithIndex(m, m.defaultFocusIndex())
	// FieldsViewport is a persistent field, not rebuilt per request - without
	// this, loading a short request after scrolling deep into a long one
	// left FieldsViewport still scrolled to that old position, hiding the
	// new request's fields (e.g. Name) above the fold until manually
	// scrolled back up.
	m.FieldsViewport.GotoTop()
}

// bodyInnerContent strips the outer braces from a request body stored as a
// JSON object (e.g. `{"a":"b"}` -> `"a":"b"`), matching how syncRequest
// re-wraps the body textarea's key=value pairs into JSON. Bodies too short
// to contain braces - "" for a request that was never synced through the
// editor, such as one saved directly by `volt import` - render blank
// instead of panicking on the slice.
func bodyInnerContent(body string) string {
	if len(body) < 2 {
		return ""
	}
	return body[1 : len(body)-1]
}
