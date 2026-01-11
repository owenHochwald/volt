package requestpane

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/volt/internal/ui"
	"github.com/owenHochwald/volt/internal/ui/keybindings"
)

// LoadTestMode implements the load test mode
type LoadTestMode struct{}

// HandleInput handles keyboard input in load test mode
func (ltm *LoadTestMode) HandleInput(m *RequestPane, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Check for special keybindings FIRST before delegating to text components
	if keybindings.Matches(msg, m.keys.SendRequest) {
		return ltm.handleSubmit(m, msg)
	}
	if keybindings.Matches(msg, m.keys.ToggleLoadTest) {
		// Don't pass to text components, will be handled in update.go
		return m, nil
	}
	if keybindings.Matches(msg, m.keys.SaveRequest) {
		// Don't pass to text components, will be handled in update.go
		return m, nil
	}

	switch FieldIndex(m.FocusManager.CurrentIndex()) {
	case FieldMethodSelector:
		if keybindings.Matches(msg, m.keys.ChangeMethodNext) {
			m.MethodSelector.Next()
		}
		if keybindings.Matches(msg, m.keys.ChangeMethodPrev) {
			m.MethodSelector.Prev()
		}
	case FieldURL:
		var cmd tea.Cmd
		*m.URLInput, cmd = m.URLInput.Update(msg)
		cmds = append(cmds, cmd)
	case FieldName:
		var cmd tea.Cmd
		*m.NameInput, cmd = m.NameInput.Update(msg)
		cmds = append(cmds, cmd)
	case FieldHeaders:
		var cmd tea.Cmd
		*m.Headers, cmd = m.Headers.Update(msg)
		cmds = append(cmds, cmd)
	case FieldBody:
		var cmd tea.Cmd
		*m.Body, cmd = m.Body.Update(msg)
		cmds = append(cmds, cmd)
	case FieldLTConcurrency:
		var cmd tea.Cmd
		*m.LoadTestConcurrency, cmd = m.LoadTestConcurrency.Update(msg)
		cmds = append(cmds, cmd)
	case FieldLTTotalReqs:
		var cmd tea.Cmd
		*m.LoadTestTotalReqs, cmd = m.LoadTestTotalReqs.Update(msg)
		cmds = append(cmds, cmd)
	case FieldLTQPS:
		var cmd tea.Cmd
		*m.LoadTestQPS, cmd = m.LoadTestQPS.Update(msg)
		cmds = append(cmds, cmd)
	case FieldLTTimeout:
		var cmd tea.Cmd
		*m.LoadTestTimeout, cmd = m.LoadTestTimeout.Update(msg)
		cmds = append(cmds, cmd)
	case FieldLTSubmit:
		return ltm.handleSubmit(m, msg)
	}

	return m, tea.Batch(cmds...)
}

// handleSubmit handles the submit button in load test mode
func (ltm *LoadTestMode) handleSubmit(m *RequestPane, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == tea.KeyEnter.String() || keybindings.Matches(msg, m.keys.SendRequest) {
		if m.RequestInProgress {
			return m, nil
		}

		m.syncRequest()
		m.RequestInProgress = true

		config, err := m.buildJobConfig()
		if err != nil {
			m.ParseErrors = append(m.ParseErrors, "Load test config error: "+err.Error())
			m.RequestInProgress = false
			return m, nil
		}
		return m, ui.StartLoadTestCmd(config)
	}
	return m, nil
}

// GetFocusManager returns the focus manager for load test mode
func (ltm *LoadTestMode) GetFocusManager(m *RequestPane) *ui.FocusManager {
	unifiedComponents := []ui.Focusable{
		m.MethodSelector,
		m.URLInput,
		m.NameInput,
		m.Headers,
		m.Body,
		m.LoadTestConcurrency,
		m.LoadTestTotalReqs,
		m.LoadTestQPS,
		m.LoadTestTimeout,
		m.SubmitButton,
	}
	return ui.NewFocusManager(unifiedComponents)
}

// GetFocusManagerWithIndex returns the focus manager for load test mode starting at a specific index
func (ltm *LoadTestMode) GetFocusManagerWithIndex(m *RequestPane, index int) *ui.FocusManager {
	unifiedComponents := []ui.Focusable{
		m.MethodSelector,
		m.URLInput,
		m.NameInput,
		m.Headers,
		m.Body,
		m.LoadTestConcurrency,
		m.LoadTestTotalReqs,
		m.LoadTestQPS,
		m.LoadTestTimeout,
		m.SubmitButton,
	}
	return ui.NewFocusManagerWithIndex(unifiedComponents, index)
}

// OnEnter is called when entering load test mode
func (ltm *LoadTestMode) OnEnter(m *RequestPane) {}

// OnExit is called when exiting load test mode
func (ltm *LoadTestMode) OnExit(m *RequestPane) {}
