package requestpane

import (
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
		focusables[i] = &params[i].Input
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
