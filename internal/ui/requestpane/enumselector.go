package requestpane

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/ui"
)

// EnumSelector is a fixed-choice field for a body param whose OpenAPI
// schema declared an "enum" (or, from paramFromFieldSchema, a "boolean" -
// cycled the same way over ["true","false"]) - cycled with the same
// left/right keys as MethodSelector, rather than typed freely.
type EnumSelector struct {
	options []string
	// index selects the current option; -1 is a distinct "no value chosen"
	// state, only ever reachable when allowUnset is set.
	index int
	// allowUnset is set when the field's schema declared no default (see
	// NewEnumSelector) - without one, there's no value that's actually
	// "the field's own", so it starts on no value at all rather than
	// silently landing on the first listed option, and Next/Prev cycle
	// through that empty state as one more stop (after the last option,
	// before the first) instead of only ever moving between real options.
	// A field whose schema *did* declare a default never gets this - it
	// always has a real value, and can't be cycled back out of one.
	allowUnset bool
	focused    bool
}

// NewEnumSelector builds a selector over options, starting on initial if
// it's among them (e.g. the schema's "default"). An empty initial means the
// schema declared no default at all - not that "" is itself a legitimate
// option - so the selector starts with no value chosen (see allowUnset)
// rather than defaulting to the first option, matching how a plain text
// field with no schema default starts empty too (see paramFromFieldSchema).
func NewEnumSelector(options []string, initial string) *EnumSelector {
	e := &EnumSelector{options: options}
	if initial == "" {
		e.allowUnset = true
		e.index = -1
		return e
	}
	for i, o := range options {
		if o == initial {
			e.index = i
			return e
		}
	}
	// initial was non-empty but isn't one of options (shouldn't normally
	// happen) - fall back to the first option, not unset, since a
	// non-empty initial signals the schema *did* declare a default.
	return e
}

// Value returns the current option, or "" when nothing's been chosen yet
// (see allowUnset) - the same "no value" BuildRequestBody already omits a
// field for (see its Value() == "" check), so an untouched enum/boolean
// field is left out of the sent JSON exactly like an untouched text field.
func (e *EnumSelector) Value() string {
	if e.index < 0 || e.index >= len(e.options) {
		return ""
	}
	return e.options[e.index]
}

func (e *EnumSelector) Next() {
	if len(e.options) == 0 {
		return
	}
	if e.allowUnset {
		e.index++
		if e.index >= len(e.options) {
			e.index = -1
		}
		return
	}
	e.index = (e.index + 1) % len(e.options)
}

func (e *EnumSelector) Prev() {
	if len(e.options) == 0 {
		return
	}
	if e.allowUnset {
		e.index--
		if e.index < -1 {
			e.index = len(e.options) - 1
		}
		return
	}
	e.index = (e.index - 1 + len(e.options)) % len(e.options)
}

// Focus/Blur satisfy ui.Focusable so an EnumSelector can sit in the same
// tab cycle as the text inputs around it.
func (e *EnumSelector) Focus() tea.Cmd {
	e.focused = true
	return nil
}

func (e *EnumSelector) Blur() {
	e.focused = false
}

func (e *EnumSelector) View() string {
	style := ui.UnfocusedButton
	if e.focused {
		style = ui.FocusedButton
	}
	display := e.Value()
	if display == "" {
		display = "(not set)"
	}
	return style.Render(fmt.Sprintf("‹ %s ›", display))
}
