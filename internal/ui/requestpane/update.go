package requestpane

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/utils"
)

// Update handles updates to the request pane
func (m RequestPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Stopwatch, cmd = m.Stopwatch.Update(msg)

	switch msg := msg.(type) {
	case ui.SetRequestPaneRequestMsg:
		m.reinitRequestPane(msg.Request)
		return m, nil

	case tea.KeyMsg:
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
		if ptr, ok := model.(*RequestPane); ok {
			return *ptr, cmd
		}
		return m, cmd
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
	m.Request = request

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
	m.BodyParams = nil
	m.ArrayParams = nil
	m.rebuildServerSelector()
	m.syncPathParams()
	m.syncBodyParams()
	// Restores whatever was typed/selected into request's fields the last
	// time it was saved/sent (see http.Request.ParamValues) - after
	// syncPathParams/syncBodyParams have (re)built the fields to apply
	// them onto, and before the rebuildFocusManager below, since applying
	// array item counts can itself change the focus cycle's shape.
	m.applyParamValues(request.ParamValues)
	// rebuildServerSelector always replaces m.ServerSelector with a fresh
	// instance, but syncPathParams/syncBodyParams only rebuild the focus
	// cycle when their own field set actually changed (e.g. a request with
	// no path or body params is a no-op for both) - so without this,
	// loading such a request would leave FocusManager still pointing at
	// the *previous*, now-orphaned ServerSelector instead of the new one,
	// and cycling it (h/l) would silently do nothing.
	m.rebuildFocusManager()
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
