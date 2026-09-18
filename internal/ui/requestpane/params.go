package requestpane

import (
	"encoding/json"
	"fmt"
	"strconv"

	"charm.land/bubbles/v2/textinput"
	"github.com/owenHochwald/Volt/internal/http"
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
	// Required mirrors whether the schema listed this field's name under
	// its parent object's "required" (only ever set for BodyParams/array
	// item fields). Its label is always shown in requiredFieldStyle as a
	// hint, regardless of whether it's currently empty; an empty value only
	// blocks sending - see RequiredFieldErrors - it isn't live-flagged the
	// way ValidateParam's checks are, since being empty is a normal,
	// unremarkable state while a form is still being filled in.
	Required bool
}

// Value returns the field's current value regardless of whether it's
// driven by Input or Enum.
func (p Param) Value() string {
	if p.Enum != nil {
		return p.Enum.Value()
	}
	return p.Input.Value()
}

// SetValue sets the field's current value, whichever of Input/Enum drives
// it - the write counterpart of Value(), used to restore a previously
// saved value (see RequestPane.ApplySnapshot). p must be addressed
// through a slice element (e.g. &params[i]), not a loop copy - same
// caveat as everywhere else a Param's Input/Enum is mutated in place.
func (p *Param) SetValue(value string) {
	if p.Enum != nil {
		p.Enum.SetValue(value)
		return
	}
	p.Input.SetValue(value)
}

// DisplayType returns what to show next to the field's name (see
// paramLabel in view.go): "enum" for a genuine schema-declared enum picker
// - its Type is whatever the enum's values are declared as (usually
// "string"), which read as a free-text field even though it's actually a
// fixed choice - rather than that underlying type. A boolean is left as
// "boolean": it's also rendered as a picker (see paramFromFieldSchema),
// but unlike an arbitrary enum, "boolean" is already the accurate,
// unambiguous type to show.
func (p Param) DisplayType() string {
	if p.Enum != nil && p.Type != "boolean" {
		return "enum"
	}
	return p.Type
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
	case "", "string":
		// A string (or untyped) field is sent close to verbatim rather
		// than through the usual full JSON string escaping (see
		// rawJSONStringValue in bodyparams.go) - only a literal `"` is
		// escaped, so a real JSON escape sequence typed directly (e.g.
		// "\n", the two characters backslash+n) reaches the server as
		// that escape sequence - a genuine newline once parsed there -
		// rather than being re-escaped into literal backslash-n text.
		// The tradeoff is a backslash that isn't part of a valid JSON
		// escape (e.g. a Windows path "C:\Users") would produce invalid
		// JSON - caught here so it blocks sending with a clear reason,
		// instead of silently corrupting the whole request body (see
		// BuildRequestBody's marshal-error fallback).
		if !json.Valid([]byte(rawJSONStringValue(val))) {
			return `contains an invalid escape sequence - \ must start a valid one: \", \\, \/, \n, \r, \t, \b, \f, or \uXXXX`, false
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

// fieldError combines ValidateParam's type/format/range check with the
// required-empty check (see RequiredFieldErrors), for the inline per-field
// display in renderParamSection/renderArrayParamSection. The required half
// is gated on showRequired (true once a send has actually been blocked on
// it - see RequestPane.ShowRequiredErrors) so an empty required field isn't
// flagged as wrong while a form is still being filled in for the first
// time; the type/format half stays live regardless, same as ValidateParam
// always was.
func fieldError(p Param, showRequired bool) (string, bool) {
	if msg, ok := ValidateParam(p); !ok {
		return msg, false
	}
	if showRequired && p.Required && p.Value() == "" {
		return "required", false
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

// RequiredFieldErrors checks every required param for an empty value,
// returning one "name: required" message per one still missing, or nil if
// all are filled in. Deliberately separate from ValidateParam/ValidateParams
// (which drive the live inline red-highlight on every render): required-ness
// is only ever checked here, at send time, so an empty required field isn't
// flagged as wrong while a form is still being filled in - only blocking the
// send itself once actually attempted.
func RequiredFieldErrors(params []Param) []string {
	var errs []string
	for _, p := range params {
		if p.Required && p.Value() == "" {
			errs = append(errs, fmt.Sprintf("%s: required", p.Name))
		}
	}
	return errs
}

// firstErrorFocusIndex returns the focus index of the first PathParams,
// BodyParams, or ArrayParams field that's currently invalid (ValidateParam)
// or a required field left empty (RequiredFieldErrors' semantics) - the
// same checks handleSubmit blocks a send on (see mode_normal.go/
// mode_loadtest.go) - so a blocked send can jump focus straight to the
// field that caused it instead of leaving the cursor wherever it happened
// to be with only the ParseErrors list under the button to notice. The
// index matches NormalMode/LoadTestMode's shared components() prefix:
// ServerSelector (only for a grouped request - see ShowServerSelector),
// NameInput, MethodSelector, URLInput, then PathParams start - both
// modes' components() must keep that same prefix (length 3, or 4 with a
// server - order among them doesn't matter here, only the count) for
// this to stay correct. ok is false when nothing in PathParams/
// BodyParams/ArrayParams has a problem - e.g. the block came from a
// malformed Headers line instead, which has no single field to jump to.
func (m *RequestPane) firstErrorFocusIndex() (int, bool) {
	focusPrefixLen := 3 // NameInput, MethodSelector, URLInput
	if m.ShowServerSelector() {
		focusPrefixLen++
	}
	fieldInvalid := func(p Param) bool {
		if _, ok := ValidateParam(p); !ok {
			return true
		}
		return p.Required && p.Value() == ""
	}

	idx := focusPrefixLen
	for _, p := range m.PathParams {
		if fieldInvalid(p) {
			return idx, true
		}
		idx++
	}
	for _, p := range m.BodyParams {
		if fieldInvalid(p) {
			return idx, true
		}
		idx++
	}
	for i := range m.ArrayParams {
		idx++ // this array's CountControl
		for _, item := range m.ArrayParams[i].Items {
			for _, f := range item.Fields {
				if fieldInvalid(f) {
					return idx, true
				}
				idx++
			}
		}
	}
	return 0, false
}

func newParamInput() textinput.Model {
	return NewConfiguredTextInput(TextInputConfig{
		Placeholder: "value",
		CharLimit:   200,
		Width:       30,
	})
}

// rebuildParams derives a fresh param slice from names, reusing each
// existing param's current value (whether typed into Input or picked via
// Enum) for names that are still present, so re-deriving this on every URL
// keystroke (see syncPathParams) doesn't reset what's already been filled
// in. schema, keyed by name, gives each param its type/enum/min-max/
// required constraints (see http.Request.PathParamSchema) via
// paramFromFieldSchema - the same constructor BodyParams use, so a path
// param behaves identically to a body field of the same schema shape
// (free text, a fixed-choice picker, a boolean switch, live validation).
// A name absent from schema (nil for a request saved before this existed,
// or a placeholder the spec declared no schema for) gets an unconstrained
// text field, same as before this existed at all.
func rebuildParams(existing []Param, names []string, schema map[string]http.BodyFieldSchema) []Param {
	byName := make(map[string]string, len(existing))
	for _, p := range existing {
		byName[p.Name] = p.Value()
	}

	params := make([]Param, len(names))
	for i, name := range names {
		s := schema[name]
		// Overriding Default with the currently-held value (rather than
		// setting it directly on the resulting Param after the fact) lets
		// paramFromFieldSchema's own Enum/boolean/Input construction pick
		// it up exactly the way it already handles a schema default - one
		// code path for both "start on the schema's default" and "resume
		// what was already typed/selected".
		if val, ok := byName[name]; ok && val != "" {
			s.Default = val
		}
		params[i] = paramFromFieldSchema(name, s)
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

// defaultFocusIndex is components()'s index for MethodSelector - where a
// freshly loaded request (see reinitRequestPane, and SetupRequestPane's
// very first one) starts focus, rather than at index 0 (NameInput, or
// ServerSelector for a grouped request). A request built from an
// imported schema rarely needs its Name touched, so there's no reason to
// land there by default every time - one shift+tab reaches it if it ever
// is needed (see components' order: server, name, method, url, ...).
func (m *RequestPane) defaultFocusIndex() int {
	idx := 0
	if m.ShowServerSelector() {
		idx++
	}
	idx++ // NameInput
	return idx
}

// focusIndexAfterURL is defaultFocusIndex()+2 - past MethodSelector and
// URLInput, to whatever comes right after them (see components' order):
// the first PathParams field, the first BodyParams/ArrayParams field, or
// the freeform Body textarea - components() always has at least one of
// those there (its own "else" fallback to Body when there are no
// PathParams/BodyParams/ArrayParams guarantees it), so this is always a
// valid index. Used when loading a request picked from the sidebar (see
// SetRequestPaneRequestMsg) - unlike a blank/reset request, its method
// and URL are already exactly what you picked it *for*, so there's
// nothing to check there before moving on to actually filling it in.
func (m *RequestPane) focusIndexAfterURL() int {
	return m.defaultFocusIndex() + 2
}
