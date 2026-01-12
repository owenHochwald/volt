package requestpane

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// NormalMode implements the standard request mode
type NormalMode struct{}

// HandleInput handles keyboard input in normal mode
func (nm *NormalMode) HandleInput(m *RequestPane, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for special keybindings FIRST before delegating to text components
	if keybindings.Matches(msg, m.keys.SendRequest) || msg.String() == tea.KeyEnter.String() {
		return nm.handleSubmit(m, msg)
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
		return m, cmd
	case FieldName:
		var cmd tea.Cmd
		*m.NameInput, cmd = m.NameInput.Update(msg)
		return m, cmd
	case FieldHeaders:
		var cmd tea.Cmd
		*m.Headers, cmd = m.Headers.Update(msg)
		return m, cmd
	case FieldBody:
		var cmd tea.Cmd
		*m.Body, cmd = m.Body.Update(msg)
		return m, cmd
	case FieldSubmitButton:
		return nm.handleSubmit(m, msg)
	}
	return m, nil
}

// handleSubmit handles the submit button in normal mode
func (nm *NormalMode) handleSubmit(m *RequestPane, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == tea.KeyEnter.String() || keybindings.Matches(msg, m.keys.SendRequest) {
		if m.RequestInProgress {
			return m, nil
		}

		m.syncRequest()
		m.RequestInProgress = true

		m.Stopwatch.Reset()
		stopwatchCmd := m.Stopwatch.Start()

		return m, tea.Batch(stopwatchCmd, sendRequestCmd(m.Client, m.Request))
	}
	return m, nil
}

// GetFocusManager returns the focus manager for normal mode
func (nm *NormalMode) GetFocusManager(m *RequestPane) *ui.FocusManager {
	components := []ui.Focusable{
		m.MethodSelector,
		m.URLInput,
		m.NameInput,
		m.Headers,
		m.Body,
		m.SubmitButton,
	}
	return ui.NewFocusManager(components)
}

// GetFocusManagerWithIndex returns the focus manager for normal mode starting at a specific index
func (nm *NormalMode) GetFocusManagerWithIndex(m *RequestPane, index int) *ui.FocusManager {
	components := []ui.Focusable{
		m.MethodSelector,
		m.URLInput,
		m.NameInput,
		m.Headers,
		m.Body,
		m.SubmitButton,
	}
	return ui.NewFocusManagerWithIndex(components, index)
}

// OnEnter is called when entering normal mode
func (nm *NormalMode) OnEnter(m *RequestPane) {}

// OnExit is called when exiting normal mode
func (nm *NormalMode) OnExit(m *RequestPane) {}
