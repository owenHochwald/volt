package requestpane

import (
	tea "charm.land/bubbletea/v2"
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
				return m, ui.NotifyCmd(ui.NotificationError, err.Error())
			}
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
}

// reinitRequestPane reinits the request pane with a new request
func (m *RequestPane) reinitRequestPane(request *http.Request) {
	m.Request = request

	m.MethodSelector.SetCurrentIndex(request.Method)
	m.URLInput.SetValue(request.URL)
	m.NameInput.SetValue(request.Name)
	m.Headers.SetValue(utils.ParseMapToString(request.Headers))
	m.Body.SetValue(request.Body)
}
