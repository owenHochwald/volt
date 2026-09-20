package requestpane

import (
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/apperror"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// NormalMode implements the standard request mode
type NormalMode struct{}

// HandleInput handles keyboard input in normal mode
func (nm *NormalMode) HandleInput(m *RequestPane, msg tea.KeyPressMsg) (*RequestPane, tea.Cmd) {
	// Check for special keybindings FIRST before delegating to text components
	if keybindings.Matches(msg, m.keys.SendRequest) {
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
	// so this one case handles all of them. A body param whose schema
	// declared an enum is an *EnumSelector instead, cycled the same way as
	// the method (left/right), not typed.
	switch c := m.FocusManager.Current().(type) {
	case *ui.MethodSelector:
		if isNextControlKey(msg, m.keys) {
			c.Next()
		}
		if isPrevControlKey(msg, m.keys) {
			c.Prev()
		}
	case *EnumSelector:
		if isNextControlKey(msg, m.keys) {
			c.Next()
		}
		if isPrevControlKey(msg, m.keys) {
			c.Prev()
		}
	case *ArrayCountControl:
		// Unlike MethodSelector/EnumSelector, cycling this changes the
		// number of focusable fields (an item's worth), so the focus cycle
		// needs rebuilding right after - same as PathParams/BodyParams
		// changing shape elsewhere.
		if isNextControlKey(msg, m.keys) {
			c.Next()
			m.rebuildFocusManager()
			m.scrollToFocused()
		}
		if isPrevControlKey(msg, m.keys) {
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
		return nm.handleSubmit(m, msg)
	}
	return m, nil
}

// isNextControlKey/isPrevControlKey recognize both m.keys.ChangeMethodNext/
// ChangeMethodPrev (h/l) and the literal arrow keys, for cycling a
// MethodSelector/EnumSelector/ArrayCountControl. Arrow keys are checked
// here directly rather than added to ChangeMethodNext/ChangeMethodPrev's
// own registry entry - keybindings.DefaultRegistry deliberately reserves
// up/down/left/right for text editing everywhere (see
// TestDefaultRegistryReservesEditingKeys), since a registered action's
// keys can, in general, be matched from contexts a plain textinput/
// textarea is also live in. It's safe here specifically because this
// whole block only ever runs when the *focused* component is one of
// these three non-text control types (see the type switch above) - a
// real *textinput.Model/*textarea.Model, handled in the cases below,
// always gets first refusal on any key via its own case instead.
func isNextControlKey(msg tea.KeyPressMsg, keys keybindings.KeyMap) bool {
	return keybindings.Matches(msg, keys.ChangeMethodNext) || msg.String() == "right"
}

func isPrevControlKey(msg tea.KeyPressMsg, keys keybindings.KeyMap) bool {
	return keybindings.Matches(msg, keys.ChangeMethodPrev) || msg.String() == "left"
}

// handleSubmit handles the submit button in normal mode
func (nm *NormalMode) handleSubmit(m *RequestPane, msg tea.KeyPressMsg) (*RequestPane, tea.Cmd) {
	if keybindings.Matches(msg, m.keys.SendRequest) || keybindings.Matches(msg, m.keys.ActivateControl) {
		if m.RequestInProgress {
			return m, nil
		}

		m.syncRequest()

		if err := m.validateRequest(); err != nil {
			m.RequestInProgress = false
			return m, ui.NotifyErrorCmd(apperror.FromApplication(err))
		}

		errs := ValidateParams(m.PathParams)
		errs = append(errs, ValidateParams(m.QueryParams)...)
		errs = append(errs, ValidateParams(m.BodyParams)...)
		errs = append(errs, ValidateArrayParams(m.ArrayParams)...)
		errs = append(errs, RequiredFieldErrors(m.PathParams)...)
		errs = append(errs, RequiredFieldErrors(m.QueryParams)...)
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

		m.Stopwatch.Reset()
		stopwatchCmd := m.Stopwatch.Start()

		cmds := []tea.Cmd{stopwatchCmd, sendRequestCmd(m.Client, m.resolvedRequest())}
		// Persisted only once a send actually goes out (validation has
		// already passed above), never for a request that's merely being
		// edited - see ApplySnapshot for where this gets restored.
		if m.DB != nil {
			cmds = append(cmds, ui.SaveLastSentRequestCmd(m.DB, m.Snapshot()))
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

// components lists this mode's focusable fields in tab order. This must
// match fieldsContent()'s render order in view.go (server - only for a
// grouped request, see ShowServerSelector - name, method+URL, path
// params, body params, array params, headers, submit) - otherwise tab
// visits fields in a different order than they're drawn in, and the cursor
// appears to jump up and down the screen instead of moving straight
// through it.
func (nm *NormalMode) components(m *RequestPane) []ui.Focusable {
	var components []ui.Focusable
	if m.ShowServerSelector() {
		components = append(components, m.ServerSelector)
	}
	components = append(components, m.NameInput, m.MethodSelector, m.URLInput)
	components = append(components, paramFocusables(m.PathParams)...)
	components = append(components, paramFocusables(m.QueryParams)...)
	if len(m.BodyParams) > 0 || len(m.ArrayParams) > 0 {
		components = append(components, paramFocusables(m.BodyParams)...)
		components = append(components, arrayParamFocusables(m.ArrayParams)...)
	} else if !m.IsSchemaRequest() {
		// A schema request with no body fields doesn't render the
		// freeform Body field at all (see fieldsContent) - it must be
		// left out of the focus cycle too, or tab would land on a field
		// that isn't actually on screen.
		components = append(components, m.Body)
	}
	components = append(components, m.Headers, m.SubmitButton)
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
