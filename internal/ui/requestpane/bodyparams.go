package requestpane

import (
	"encoding/json"
	"regexp"
	"sort"
)

// bodySpec is one field extracted from a body template: its name and,
// when the template encoded one, its declared type (see bodyTypeMarker).
type bodySpec struct {
	Name string
	Type string
}

// bodyTypeMarker matches the "<type>" placeholder `volt import` writes for
// a field whose OpenAPI schema declared a type, e.g. "<string>",
// "<integer>". A plain empty string means "field exists, type unknown".
var bodyTypeMarker = regexp.MustCompile(`^<(\w+)>$`)

// extractBodySpecs returns the top-level fields of body when it looks like
// a `volt import`-generated request-body-schema template: a flat JSON
// object whose values are all either empty strings or "<type>" markers,
// e.g. {"name":"<string>","color":""}. Sorted by name.
//
// The "empty or marker only" requirement is deliberate, not incidental: a
// body that already carries real data - a request saved earlier the normal
// way, body typed by hand as "key = value" and synced to {"key":"value"} -
// must stay visible and editable as text, not get swapped for blank
// fill-in fields that would hide it. Only an untouched template is treated
// as body params. Non-string values (numbers, nested objects, arrays) fail
// to unmarshal into map[string]string and so also fall back to the
// freeform textarea - a deliberate simplification for JSON-only support.
func extractBodySpecs(body string) []bodySpec {
	if body == "" {
		return nil
	}
	var obj map[string]string
	if err := json.Unmarshal([]byte(body), &obj); err != nil || len(obj) == 0 {
		return nil
	}
	for _, v := range obj {
		if v != "" && !bodyTypeMarker.MatchString(v) {
			return nil
		}
	}

	names := make([]string, 0, len(obj))
	for name := range obj {
		names = append(names, name)
	}
	sort.Strings(names)

	specs := make([]bodySpec, len(names))
	for i, name := range names {
		typ := ""
		if m := bodyTypeMarker.FindStringSubmatch(obj[name]); m != nil {
			typ = m[1]
		}
		specs[i] = bodySpec{Name: name, Type: typ}
	}
	return specs
}

// BuildRequestBody serializes the current body param values into a flat
// JSON object, e.g. {"name":"widget","color":"red"}.
func BuildRequestBody(params []Param) string {
	body := make(map[string]string, len(params))
	for _, p := range params {
		body[p.Name] = p.Input.Value()
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// syncBodyParams re-derives m.BodyParams (name, type, and input) from the
// request's current Body, preserving already-typed values for names that
// are still present. It's a no-op if the set of param names hasn't
// changed, and otherwise rebuilds the focus cycle so the (possibly new)
// fields are reachable via tab.
func (m *RequestPane) syncBodyParams() {
	specs := extractBodySpecs(m.Request.Body)

	names := make([]string, len(specs))
	for i, s := range specs {
		names[i] = s.Name
	}
	if paramNamesMatch(names, m.BodyParams) {
		return
	}

	params := rebuildParams(m.BodyParams, names)
	for i, s := range specs {
		params[i].Type = s.Type
		if s.Type != "" {
			params[i].Input.Placeholder = s.Type
		}
	}
	m.BodyParams = params
	m.rebuildFocusManager()
}
