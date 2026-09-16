package requestpane

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/owenHochwald/Volt/internal/http"
)

// bodySpec is one field extracted from a request's body schema template.
type bodySpec struct {
	Name      string
	Type      string
	Default   string
	Enum      []string
	MinLength *int
	MaxLength *int
	Minimum   *float64
	Maximum   *float64
}

// extractBodySpecs returns the fields declared under the sentinel
// http.BodySchemaKey in body (as `volt import` writes for an operation
// with a JSON request body schema), sorted by name. Any other shape -
// empty, not JSON, a real (non-schema) body a user typed by hand - yields
// no body params, so the freeform Body textarea is used instead (see
// RequestPane.View). Detecting the schema by this explicit sentinel key,
// rather than inferring it from the body's shape, means a real request
// body can never be mistaken for one: nothing but `volt import` would
// plausibly produce a top-level key literally named "__volt_body_schema__".
func extractBodySpecs(body string) []bodySpec {
	if body == "" {
		return nil
	}
	var wrapper map[string]map[string]http.BodyFieldSchema
	if err := json.Unmarshal([]byte(body), &wrapper); err != nil {
		return nil
	}
	schema, ok := wrapper[http.BodySchemaKey]
	if !ok || len(schema) == 0 {
		return nil
	}

	names := make([]string, 0, len(schema))
	for name := range schema {
		names = append(names, name)
	}
	sort.Strings(names)

	specs := make([]bodySpec, len(names))
	for i, name := range names {
		f := schema[name]
		specs[i] = bodySpec{
			Name:      name,
			Type:      f.Type,
			Default:   f.Default,
			Enum:      f.Enum,
			MinLength: f.MinLength,
			MaxLength: f.MaxLength,
			Minimum:   f.Minimum,
			Maximum:   f.Maximum,
		}
	}
	return specs
}

// BuildRequestBody serializes the current body param values into a flat
// JSON object, respecting each field's declared type (e.g.
// {"active":true,"count":5,"name":"widget"}) rather than sending every
// value as a JSON string regardless of what the schema actually declared.
func BuildRequestBody(params []Param) string {
	body := make(map[string]interface{}, len(params))
	for _, p := range params {
		body[p.Name] = paramJSONValue(p)
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// paramJSONValue converts a param's current text value to the JSON type its
// schema declared. A value that doesn't actually parse as that type (still
// being typed, or invalid - see ValidateParam) falls back to a plain
// string, so it round-trips as typed instead of being silently dropped or
// zeroed.
func paramJSONValue(p Param) interface{} {
	val := p.Value()
	switch p.Type {
	case "boolean":
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	case "integer":
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			return n
		}
	case "number":
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return val
}

// syncBodyParams re-derives m.BodyParams from the request's current Body.
// It's a no-op if the set of param names hasn't changed, and otherwise
// rebuilds the focus cycle so the (possibly new) fields are reachable via
// tab. Unlike syncPathParams, this only ever runs once per loaded/created
// request (from reinitRequestPane) rather than on every keystroke - see
// RequestPane.syncRequest for why re-deriving continuously would be
// actively wrong here - so params are always built fresh (no existing
// input's value needs preserving) and can freely take their schema's
// default value or enum picker right away.
func (m *RequestPane) syncBodyParams() {
	specs := extractBodySpecs(m.Request.Body)

	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = s.Name
	}
	if paramNamesMatch(names, m.BodyParams) {
		return
	}

	params := make([]Param, len(specs))
	for i, s := range specs {
		p := Param{
			Name:      s.Name,
			Type:      s.Type,
			MinLength: s.MinLength,
			MaxLength: s.MaxLength,
			Minimum:   s.Minimum,
			Maximum:   s.Maximum,
		}
		if len(s.Enum) > 0 {
			// Enum values are always one of the fixed options, so
			// min/maxLength and minimum/maximum don't apply - an enum
			// picker can't produce an out-of-range value to begin with.
			p.Enum = NewEnumSelector(s.Enum, s.Default)
		} else if s.Type == "boolean" {
			// A boolean can only ever be true or false, so it gets the
			// same fixed-choice picker as a declared enum instead of a
			// free-text field someone could type "yes"/"1"/anything else
			// into.
			initial := "false"
			if s.Default == "true" || s.Default == "false" {
				initial = s.Default
			}
			p.Enum = NewEnumSelector([]string{"true", "false"}, initial)
		} else {
			p.Input = newParamInput()
			if s.Type != "" {
				p.Input.Placeholder = s.Type
			}
			if s.MaxLength != nil {
				// Enforced live: typing simply can't go past this, unlike
				// MinLength/Minimum/Maximum which are checked at send time
				// (see ValidateParams) since they can't be enforced
				// without rejecting perfectly normal in-progress typing.
				p.Input.CharLimit = *s.MaxLength
			}
			if s.Default != "" {
				p.Input.SetValue(s.Default)
			}
		}
		params[i] = p
	}
	m.BodyParams = params
	m.rebuildFocusManager()
}
