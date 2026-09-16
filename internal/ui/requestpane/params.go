package requestpane

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/owenHochwald/Volt/internal/ui"
)

// Param is one editable named field derived from a request - either a
// {name} placeholder in the URL (see PathParams) or a top-level property
// of a JSON request body schema (see BodyParams).
type Param struct {
	Name string
	// Type is the field's declared data type (e.g. "string", "integer"),
	// shown as a hint next to the field. Only ever set for BodyParams,
	// where it comes from the OpenAPI schema `volt import` saw; empty when
	// unknown (PathParams, or a body schema with no declared type).
	Type  string
	Input textinput.Model
	// Enum, when non-nil (only ever for a BodyParam whose schema declared
	// one), drives the field's value instead of Input: a fixed-choice
	// picker rather than free text.
	Enum *EnumSelector
	// MinLength/MaxLength/Minimum/Maximum mirror the schema's constraints
	// of the same name (only ever set for BodyParams); nil means "no such
	// constraint". MaxLength is also enforced live via Input.CharLimit -
	// see syncBodyParams - so violating it isn't actually reachable by
	// typing, but ValidateParams still checks it for a pre-filled Default
	// that happens to exceed it.
	MinLength *int
	MaxLength *int
	Minimum   *float64
	Maximum   *float64
}

// Value returns the field's current value regardless of whether it's
// driven by Input or Enum.
func (p Param) Value() string {
	if p.Enum != nil {
		return p.Enum.Value()
	}
	return p.Input.Value()
}

// ValidateParam checks a single param's current value against its declared
// type ("integer"/"number" must actually parse as such) and its
// MinLength/MaxLength/Minimum/Maximum constraints, returning a short
// problem description and false for the first thing wrong with it, or
// ("", true) when it's fine. Always valid for an Enum param (a fixed-choice
// picker, including a boolean's true/false switch, can't produce an
// invalid value) and for an empty value (required-ness isn't enforced
// here - blank isn't itself a violation). Used both to block sending
// outright (see ValidateParams) and to highlight the field live while
// typing (see renderParamSection in view.go).
func ValidateParam(p Param) (string, bool) {
	if p.Enum != nil {
		return "", true
	}
	val := p.Input.Value()
	if val == "" {
		return "", true
	}

	switch p.Type {
	case "integer":
		if _, err := strconv.ParseInt(val, 10, 64); err != nil {
			return "must be a whole number", false
		}
	case "number":
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			return "must be a number", false
		}
	}

	if p.MinLength != nil && len(val) < *p.MinLength {
		return fmt.Sprintf("must be at least %d character(s)", *p.MinLength), false
	}
	if p.MaxLength != nil && len(val) > *p.MaxLength {
		return fmt.Sprintf("must be at most %d character(s)", *p.MaxLength), false
	}

	if p.Minimum != nil || p.Maximum != nil {
		// A value that isn't (yet) a valid number - e.g. still blank, or an
		// untyped field where Minimum/Maximum happen to be set anyway - is
		// left alone here rather than reported as a range violation, which
		// would be misleading; a declared "integer"/"number" type already
		// caught a bad value above.
		if num, err := strconv.ParseFloat(val, 64); err == nil {
			if p.Minimum != nil && num < *p.Minimum {
				return fmt.Sprintf("must be >= %v", *p.Minimum), false
			}
			if p.Maximum != nil && num > *p.Maximum {
				return fmt.Sprintf("must be <= %v", *p.Maximum), false
			}
		}
	}
	return "", true
}

// ValidateParams checks every param with ValidateParam, returning one
// "name: reason" message per violation, or nil if everything's within
// bounds. Called before a request is actually sent (see mode_normal.go and
// mode_loadtest.go's handleSubmit).
func ValidateParams(params []Param) []string {
	var errs []string
	for _, p := range params {
		if msg, ok := ValidateParam(p); !ok {
			errs = append(errs, fmt.Sprintf("%s: %s", p.Name, msg))
		}
	}
	return errs
}

func newParamInput() textinput.Model {
	return NewConfiguredTextInput(TextInputConfig{
		Placeholder: "value",
		CharLimit:   200,
		Width:       30,
	})
}

// rebuildParams derives a fresh param slice from names, reusing existing
// inputs (and their typed values) for names that are still present.
func rebuildParams(existing []Param, names []string) []Param {
	byName := make(map[string]textinput.Model, len(existing))
	for _, p := range existing {
		byName[p.Name] = p.Input
	}

	params := make([]Param, len(names))
	for i, name := range names {
		input, ok := byName[name]
		if !ok {
			input = newParamInput()
		}
		params[i] = Param{Name: name, Input: input}
	}
	return params
}

func paramNamesMatch(names []string, params []Param) bool {
	if len(names) != len(params) {
		return false
	}
	for i, name := range names {
		if params[i].Name != name {
			return false
		}
	}
	return true
}

// paramFocusables exposes each param's input for the focus cycle. Pointers
// are taken into the given slice itself (not a copy), so this must only be
// called against a slice that isn't about to be reassigned - syncPathParams
// and syncBodyParams guarantee that by rebuilding the focus manager
// immediately after (re)assigning m.PathParams/m.BodyParams.
func paramFocusables(params []Param) []ui.Focusable {
	focusables := make([]ui.Focusable, len(params))
	for i := range params {
		if params[i].Enum != nil {
			focusables[i] = params[i].Enum
		} else {
			focusables[i] = &params[i].Input
		}
	}
	return focusables
}

// rebuildFocusManager recreates the focus cycle, keeping focus where it
// currently is (clamped to the new, possibly different, field count).
// Called whenever PathParams or BodyParams changes shape.
func (m *RequestPane) rebuildFocusManager() {
	currentIndex := 0
	if m.FocusManager != nil {
		currentIndex = m.FocusManager.CurrentIndex()
	}
	m.FocusManager = m.currentMode.GetFocusManagerWithIndex(m, currentIndex)
}
