package requestpane

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/owenHochwald/Volt/internal/http"
)

// bodySpec is one field extracted from a request's body schema template.
type bodySpec struct {
	Name        string
	Type        string
	Default     string
	Enum        []string
	Description string
	MinLength   *int
	MaxLength   *int
	Minimum     *float64
	Maximum     *float64
	// Items/ItemsRefName/ItemType mirror http.BodyFieldSchema's fields of
	// the same name. When Type is "array" and Items is non-empty (a
	// "complex" array - its schema's "items" was a $ref to an object),
	// this spec becomes a "complex" ArrayParam; when Type is "array" and
	// Items is empty but ItemType isn't (a "simple" array - its items had
	// no object structure to compose), it becomes a "simple" ArrayParam
	// instead - see syncBodyParams and ArrayParam.IsSimple.
	Items        map[string]http.BodyFieldSchema
	ItemsRefName string
	ItemType     string
	Required     bool
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
			Name:         name,
			Type:         f.Type,
			Default:      f.Default,
			Enum:         f.Enum,
			Description:  f.Description,
			MinLength:    f.MinLength,
			MaxLength:    f.MaxLength,
			Minimum:      f.Minimum,
			Maximum:      f.Maximum,
			Items:        f.Items,
			ItemsRefName: f.ItemsRefName,
			ItemType:     f.ItemType,
			Required:     f.Required,
		}
	}
	return specs
}

// BuildRequestBody serializes the current body param values - both scalar
// BodyParams and repeatable ArrayParams - into a flat JSON object,
// respecting each field's declared type (e.g. {"active":true,"count":5,
// "name":"widget","tags":[{"key":"a","value":"1"}]}) rather than sending
// every value as a JSON string regardless of what the schema declared.
//
// A field with no value at all - never typed into (or, for an Enum/
// boolean picker, never cycled off its starting "no value" state - see
// EnumSelector.allowUnset) and no schema Default - is left out of the
// JSON entirely rather than sent as "" (or 0 for a number, false for a
// boolean the user never touched): only fields the user actually filled
// in, or that came pre-filled from the schema's own default, should reach
// the server.
func BuildRequestBody(params []Param, arrays []ArrayParam) string {
	body := make(map[string]interface{}, len(params)+len(arrays))
	for _, p := range params {
		if p.Value() == "" {
			continue
		}
		body[p.Name] = paramJSONValue(p)
	}
	for _, a := range arrays {
		body[a.Name] = arrayJSONValue(a)
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// arrayJSONValue builds the JSON array an ArrayParam represents - an array
// of objects for a "complex" array, e.g.
// [{"key":"a","value":"1"},{"key":"b","value":"2"}], or a flat array of
// scalars for a "simple" one (see ArrayParam.IsSimple), e.g. [10,20,30] -
// always a (possibly empty) array, never null, so removing every item
// still sends "[]" rather than dropping the field's JSON type.
func arrayJSONValue(a ArrayParam) interface{} {
	if a.IsSimple() {
		// Each item holds exactly one unnamed field (see
		// simpleArrayItemName) - an item left empty (added but never typed
		// into) is skipped, same "no value, don't send it" rule as a
		// top-level param (see BuildRequestBody), rather than sending a
		// zero-value placeholder entry.
		items := make([]interface{}, 0, len(a.Items))
		for _, item := range a.Items {
			f := item.Fields[0]
			if f.Value() == "" {
				continue
			}
			items = append(items, paramJSONValue(f))
		}
		return items
	}

	// Each item's own fields follow the same "empty and no default is left
	// out" rule as top-level params (see BuildRequestBody) - an item the
	// user added but didn't (fully) fill in only sends whichever of its
	// fields actually have a value, not every field in its schema.
	items := make([]map[string]interface{}, len(a.Items))
	for i, item := range a.Items {
		obj := make(map[string]interface{}, len(item.Fields))
		for _, f := range item.Fields {
			if f.Value() == "" {
				continue
			}
			obj[f.Name] = paramJSONValue(f)
		}
		items[i] = obj
	}
	return items
}

// paramJSONValue converts a param's current text value to the JSON type its
// schema declared. A value that doesn't actually parse as that type (still
// being typed, or invalid - see ValidateParam) falls back to a raw JSON
// string (see rawJSONStringValue), so it round-trips as typed instead of
// being silently dropped or zeroed.
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
	return rawJSONStringValue(val)
}

// rawJSONStringValue builds a JSON string from val, escaping only a
// literal `"` (to `\"`, so it doesn't prematurely end the JSON string) -
// deliberately not the full JSON string escaping json.Marshal would apply
// to a plain Go string (which would also escape a literal backslash).
// That distinction matters for someone typing a real JSON escape sequence
// directly into a string field - e.g. "\n" as the two characters
// backslash+n - who wants it to reach the server as that escape sequence
// (a genuine newline once the server parses the JSON body), not
// re-escaped into the literal four-character sequence "\\n" that would
// show up as backslash-n text once parsed. The tradeoff is explicit and
// intentional: a backslash that isn't part of a valid JSON escape (e.g. a
// Windows path "C:\Users") produces invalid JSON - typing a raw escape
// sequence is presumed deliberate here, the same way the freeform Body
// textarea (for a request with no known schema) already sends whatever's
// typed close to verbatim.
func rawJSONStringValue(val string) json.RawMessage {
	escaped := strings.ReplaceAll(val, `"`, `\"`)
	return json.RawMessage(`"` + escaped + `"`)
}

// paramFromFieldSchema builds one editable Param field from a name +
// schema - shared by top-level body params (see syncBodyParams below) and
// each item of an array-of-objects param (see ArrayParam in
// arrayparams.go), so a boolean/enum/default/constraint field behaves
// identically whether it's a top-level property or nested inside an array
// item.
func paramFromFieldSchema(name string, s http.BodyFieldSchema) Param {
	p := Param{
		Name:        name,
		Type:        s.Type,
		MinLength:   s.MinLength,
		MaxLength:   s.MaxLength,
		Minimum:     s.Minimum,
		Maximum:     s.Maximum,
		Required:    s.Required,
		Description: s.Description,
	}
	if len(s.Enum) > 0 {
		// Enum values are always one of the fixed options, so
		// min/maxLength and minimum/maximum don't apply - an enum picker
		// can't produce an out-of-range value to begin with.
		p.Enum = NewEnumSelector(s.Enum, s.Default)
	} else if s.Type == "boolean" {
		// A boolean can only ever be true or false, so it gets the same
		// fixed-choice picker as a declared enum instead of a free-text
		// field someone could type "yes"/"1"/anything else into. Without a
		// declared default, this used to silently start on "false" - not a
		// real default, just this field's zero value, but indistinguishable
		// from a deliberate choice once serialized - so a boolean the user
		// never touched was always sent as false. Passing "" through lets
		// NewEnumSelector start it unset instead (see allowUnset), so it's
		// only ever sent once actually set to true or false.
		initial := ""
		if s.Default == "true" || s.Default == "false" {
			initial = s.Default
		}
		p.Enum = NewEnumSelector([]string{"true", "false"}, initial)
	} else {
		p.Input = newParamInput()
		switch {
		case s.Description != "":
			// The description is more useful than the bare type name as
			// a hint (e.g. "the user's display name" vs. "string") - see
			// http.BodyFieldSchema.Description.
			p.Input.Placeholder = s.Description
		case s.Type != "":
			p.Input.Placeholder = s.Type
		}
		if s.MaxLength != nil {
			// Enforced live: typing simply can't go past this, unlike
			// MinLength/Minimum/Maximum which are checked at send time (see
			// ValidateParams) since they can't be enforced without
			// rejecting perfectly normal in-progress typing.
			p.Input.CharLimit = *s.MaxLength
		}
		if s.Default != "" {
			p.Input.SetValue(s.Default)
		}
	}
	return p
}

// syncBodyParams re-derives m.BodyParams/m.ArrayParams from the request's
// current Body. It's a no-op if the set of field names hasn't changed, and
// otherwise rebuilds the focus cycle so the (possibly new) fields are
// reachable via tab. Unlike syncPathParams, this only ever runs once per
// loaded/created request (from reinitRequestPane) rather than on every
// keystroke - see RequestPane.syncRequest for why re-deriving continuously
// would be actively wrong here - so params are always built fresh (no
// existing input's value needs preserving) and can freely take their
// schema's default value or enum picker right away.
func (m *RequestPane) syncBodyParams() {
	specs := extractBodySpecs(m.Request.Body)

	// reinitRequestPane always resets both to nil immediately before
	// calling this, so this only ever actually skips the rebuild below
	// when the request has no body schema at all.
	if len(specs) == 0 && len(m.BodyParams) == 0 && len(m.ArrayParams) == 0 {
		return
	}

	var bodyParams []Param
	var arrayParams []ArrayParam
	for _, s := range specs {
		// A "complex" array (its schema's "items" was a $ref to an object)
		// becomes an ArrayParam with one Param per named sub-field per
		// item; a "simple" array (Items unset but ItemType is - e.g.
		// items: {type: number}, with no per-property structure to
		// compose) becomes an ArrayParam too, but with each item holding
		// one unnamed scalar field instead (see ArrayParam.IsSimple) -
		// either way, editable as a repeatable list rather than a single
		// opaque text field.
		if s.Type == "array" && len(s.Items) > 0 {
			arrayParams = append(arrayParams, newArrayParam(s.Name, s.ItemsRefName, s.Items))
			continue
		}
		if s.Type == "array" && s.ItemType != "" {
			arrayParams = append(arrayParams, newSimpleArrayParam(s.Name, s.ItemType))
			continue
		}
		bodyParams = append(bodyParams, paramFromFieldSchema(s.Name, http.BodyFieldSchema{
			Type: s.Type, Default: s.Default, Enum: s.Enum,
			Description: s.Description,
			MinLength:   s.MinLength, MaxLength: s.MaxLength,
			Minimum: s.Minimum, Maximum: s.Maximum,
			Required: s.Required,
		}))
	}
	m.BodyParams = bodyParams
	m.ArrayParams = arrayParams
	// Wired up here rather than inside newArrayParam: a *ArrayParam taken
	// there would point at newArrayParam's own local variable, not at the
	// copy that ends up living in m.ArrayParams once this loop appends it -
	// mutating through that stale pointer (see ArrayCountControl) would
	// silently miss the slice View() and the focus cycle actually read from.
	for i := range m.ArrayParams {
		m.ArrayParams[i].CountControl = &ArrayCountControl{array: &m.ArrayParams[i]}
	}
	m.rebuildFocusManager()
}
