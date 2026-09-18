package requestpane

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// LoadTestMode implements the load test mode
type LoadTestMode struct{}

// HandleInput handles keyboard input in load test mode
func (ltm *LoadTestMode) HandleInput(m *RequestPane, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	// Dispatch by the focused component's own type rather than a fixed
	// position, since the number of fields (PathParams) varies with the
	// URL. Every text input in this mode - URL, path params, name, and the
	// load-test config fields - is a *textinput.Model, so one case covers
	// them all. A body param whose schema declared an enum is an
	// *EnumSelector instead, cycled the same way as the method
	// (left/right), not typed.
	switch c := m.FocusManager.Current().(type) {
	case *ui.MethodSelector:
		if keybindings.Matches(msg, m.keys.ChangeMethodNext) {
			c.Next()
		}
		if keybindings.Matches(msg, m.keys.ChangeMethodPrev) {
			c.Prev()
		}
	case *EnumSelector:
		if keybindings.Matches(msg, m.keys.ChangeMethodNext) {
			c.Next()
		}
		if keybindings.Matches(msg, m.keys.ChangeMethodPrev) {
			c.Prev()
		}
	case *ArrayCountControl:
		if keybindings.Matches(msg, m.keys.ChangeMethodNext) {
			c.Next()
			m.rebuildFocusManager()
			m.scrollToFocused()
		}
		if keybindings.Matches(msg, m.keys.ChangeMethodPrev) {
			c.Prev()
			m.rebuildFocusManager()
			m.scrollToFocused()
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
		return ltm.handleSubmit(m, msg)
	}
	return m, nil
}

// handleSubmit handles the submit button in load test mode
func (ltm *LoadTestMode) handleSubmit(m *RequestPane, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == tea.KeyEnter.String() || keybindings.Matches(msg, m.keys.SendRequest) {
		if m.RequestInProgress {
			return m, nil
		}

		m.syncRequest()

		errs := ValidateParams(m.PathParams)
		errs = append(errs, ValidateParams(m.BodyParams)...)
		errs = append(errs, ValidateArrayParams(m.ArrayParams)...)
		errs = append(errs, RequiredFieldErrors(m.PathParams)...)
		errs = append(errs, RequiredFieldErrors(m.BodyParams)...)
		errs = append(errs, RequiredArrayFieldErrors(m.ArrayParams)...)
		if len(errs) > 0 {
			// Shown inline next to each offending field instead (see
			// renderParamSection/renderArrayParamSection) - ShowRequiredErrors
			// switches on the required-empty half of that display; the
			// type/format half (ValidateParam) already always shows live.
			m.ShowRequiredErrors = true
			if idx, ok := m.firstErrorFocusIndex(); ok {
				m.FocusManager.FocusIndex(idx)
				m.scrollToFocused()
			}
			return m, nil
		}

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

// components lists this mode's focusable fields in tab order. This must
// match fieldsContent()'s render order in view.go (server - only for a
// grouped request, see ShowServerSelector - name, method+URL, path
// params, body params, headers, load-test config, submit) - otherwise
// tab visits fields in a different order than they're drawn in, and the
// cursor appears to jump up and down the screen instead of moving straight
// through it.
func (ltm *LoadTestMode) components(m *RequestPane) []ui.Focusable {
	var components []ui.Focusable
	if m.ShowServerSelector() {
		components = append(components, m.ServerSelector)
	}
	components = append(components, m.NameInput, m.MethodSelector, m.URLInput)
	components = append(components, paramFocusables(m.PathParams)...)
	if len(m.BodyParams) > 0 || len(m.ArrayParams) > 0 {
		components = append(components, paramFocusables(m.BodyParams)...)
		components = append(components, arrayParamFocusables(m.ArrayParams)...)
	} else {
		components = append(components, m.Body)
	}
	components = append(components,
		m.Headers,
		m.LoadTestConcurrency,
		m.LoadTestTotalReqs,
		m.LoadTestQPS,
		m.LoadTestTimeout,
		m.SubmitButton,
	)
	return components
}

// GetFocusManager returns the focus manager for load test mode
func (ltm *LoadTestMode) GetFocusManager(m *RequestPane) *ui.FocusManager {
	return ui.NewFocusManager(ltm.components(m))
}

// GetFocusManagerWithIndex returns the focus manager for load test mode starting at a specific index
func (ltm *LoadTestMode) GetFocusManagerWithIndex(m *RequestPane, index int) *ui.FocusManager {
	return ui.NewFocusManagerWithIndex(ltm.components(m), index)
}

// OnEnter is called when entering load test mode
func (ltm *LoadTestMode) OnEnter(m *RequestPane) {}

// OnExit is called when exiting load test mode
func (ltm *LoadTestMode) OnExit(m *RequestPane) {}
