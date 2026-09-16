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
			return m, ui.SaveRequestCmd(m.DB, m.Request)
		}
		if keybindings.Matches(msg, m.keys.NextField) {
			m.FocusManager.Next()
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.PrevField) {
			m.FocusManager.Prev()
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.ScrollFieldsUp) {
			// SetContent first: FieldsViewport.lines are only ever
			// populated inside View() (value receiver - never persisted
			// here), so without this refresh every scroll would see empty
			// content, read maxYOffset()==0, and PageUp/PageDown would be
			// a permanent no-op.
			m.FieldsViewport.SetContent(m.fieldsContent())
			m.FieldsViewport.PageUp()
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.ScrollFieldsDown) {
			m.FieldsViewport.SetContent(m.fieldsContent())
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
	m.syncPathParams()
	m.syncBodyParams()
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
