package requestpane

import (
	"net/url"
	"sort"
	"strings"

	"github.com/owenHochwald/Volt/internal/http"
)

// sortedSchemaNames returns schema's keys in a stable (sorted) order -
// QueryParams' equivalent of extractPathParamNames, except names come
// from the schema itself (there's no {name}-style placeholder syntax in
// a query string to derive them from - see http.Request.QueryParamSchema's
// own doc comment) rather than from parsing the URL text.
func sortedSchemaNames(schema map[string]http.BodyFieldSchema) []string {
	names := make([]string, 0, len(schema))
	for name := range schema {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// syncQueryParams re-derives m.QueryParams from the request's
// QueryParamSchema, preserving already-typed/selected values for names
// that are still present - mirrors syncBodyParams (derived once per
// loaded/created request, from reinitRequestPane, not on every
// keystroke - QueryParamSchema doesn't change as the URL or anything
// else is edited, unlike PathParams' URL-derived names) rather than
// syncPathParams' continuous re-derivation.
func (m *RequestPane) syncQueryParams() {
	names := sortedSchemaNames(m.Request.QueryParamSchema)
	if paramNamesMatch(names, m.QueryParams) {
		return
	}
	m.QueryParams = rebuildParams(m.QueryParams, names, m.Request.QueryParamSchema)
	m.rebuildFocusManager()
}

// BuildQueryString serializes the current query param values into a
// "?key=value&..." string (URL-encoded, so a value containing "&"/"="/
// spaces/etc. round-trips correctly), or "" if none has a value - the
// query-string counterpart of BuildRequestBody. A param with no value at
// all (never typed into, no schema Default) is left out entirely, same
// "no value, don't send it" rule BuildRequestBody already follows.
func BuildQueryString(params []Param) string {
	values := url.Values{}
	for _, p := range params {
		if v := p.Value(); v != "" {
			values.Set(p.Name, v)
		}
	}
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}

// appendQueryString appends queryString (as BuildQueryString returns it,
// including its own leading "?") to rawURL. An imported request's own
// URL never has a query string of its own (query params are tracked
// separately via QueryParamSchema/QueryParams, not embedded in the URL
// text - see http.Request.QueryParamSchema), but a defensively-handled
// "?" already present (a hand-edited URL, or a re-import merged onto an
// existing request) is joined with "&" instead of overwritten.
func appendQueryString(rawURL, queryString string) string {
	if queryString == "" {
		return rawURL
	}
	if strings.Contains(rawURL, "?") {
		return rawURL + "&" + strings.TrimPrefix(queryString, "?")
	}
	return rawURL + queryString
}
