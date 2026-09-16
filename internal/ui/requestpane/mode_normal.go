package requestpane

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
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

	// Dispatch by the focused component's own type rather than a fixed
	// position, since the number of fields (PathParams) varies with the
	// URL. URLInput/NameInput/PathParam inputs are all *textinput.Model,
	// so this one case handles all of them.
	switch c := m.FocusManager.Current().(type) {
	case *ui.MethodSelector:
		if keybindings.Matches(msg, m.keys.ChangeMethodNext) {
			c.Next()
		}
		if keybindings.Matches(msg, m.keys.ChangeMethodPrev) {
			c.Prev()
		}
	case *textinput.Model:
		var cmd tea.Cmd
		*c, cmd = c.Update(msg)
		return m, cmd
	case *textarea.Model:
		var cmd tea.Cmd
		*c, cmd = c.Update(msg)
		return m, cmd
	case *ui.SubmitButton:
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

		return m, tea.Batch(stopwatchCmd, sendRequestCmd(m.Client, m.resolvedRequest()))
	}
	return m, nil
}

// components lists this mode's focusable fields in tab order: method, URL,
// any path params extracted from the URL, name, headers, then either the
// body params (an imported request with a known JSON schema) or the
// freeform body textarea, then submit.
func (nm *NormalMode) components(m *RequestPane) []ui.Focusable {
	components := []ui.Focusable{m.MethodSelector, m.URLInput}
	components = append(components, paramFocusables(m.PathParams)...)
	components = append(components, m.NameInput, m.Headers)
	if len(m.BodyParams) > 0 {
		components = append(components, paramFocusables(m.BodyParams)...)
	} else {
		components = append(components, m.Body)
	}
	components = append(components, m.SubmitButton)
	return components
}

// GetFocusManager returns the focus manager for normal mode
func (nm *NormalMode) GetFocusManager(m *RequestPane) *ui.FocusManager {
	return ui.NewFocusManager(nm.components(m))
}

// GetFocusManagerWithIndex returns the focus manager for normal mode starting at a specific index
func (nm *NormalMode) GetFocusManagerWithIndex(m *RequestPane, index int) *ui.FocusManager {
	return ui.NewFocusManagerWithIndex(nm.components(m), index)
}

// OnEnter is called when entering normal mode
func (nm *NormalMode) OnEnter(m *RequestPane) {}

// OnExit is called when exiting normal mode
func (nm *NormalMode) OnExit(m *RequestPane) {}
