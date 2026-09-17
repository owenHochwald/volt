package requestpane

import "github.com/owenHochwald/Volt/internal/http"

// Snapshot returns a full copy of this pane's current request - its
// schema/template (m.Request) plus every currently typed/selected field
// value (see paramValues/http.ParamValues) - suitable for persisting as-is
// (see ui.SaveRequestCmd/ui.SaveLastSentRequestCmd): reloading it (see
// reinitRequestPane, which every load already goes through) restores
// exactly what was filled in, not just blank fields matching the schema.
func (m *RequestPane) Snapshot() http.Request {
	req := *m.Request
	req.ParamValues = m.paramValues()
	return req
}

// paramValues captures every currently filled-in path/body/array param
// value, keyed by name - the part of Snapshot that isn't already sitting
// on m.Request. A field left blank (nothing typed, no schema default) is
// omitted, same "no value, nothing to save" rule BuildRequestBody already
// follows when sending.
func (m *RequestPane) paramValues() http.ParamValues {
	pathValues := make(map[string]string, len(m.PathParams))
	for _, p := range m.PathParams {
		if v := p.Value(); v != "" {
			pathValues[p.Name] = v
		}
	}
	bodyValues := make(map[string]string, len(m.BodyParams))
	for _, p := range m.BodyParams {
		if v := p.Value(); v != "" {
			bodyValues[p.Name] = v
		}
	}
	arrayValues := make(map[string][]map[string]string, len(m.ArrayParams))
	for _, a := range m.ArrayParams {
		items := make([]map[string]string, len(a.Items))
		for i, item := range a.Items {
			obj := make(map[string]string, len(item.Fields))
			for _, f := range item.Fields {
				if v := f.Value(); v != "" {
					obj[f.Name] = v
				}
			}
			items[i] = obj
		}
		arrayValues[a.Name] = items
	}

	return http.ParamValues{
		PathParamValues:  pathValues,
		BodyParamValues:  bodyValues,
		ArrayParamValues: arrayValues,
	}
}

// applyParamValues overlays values onto the pane's already-built
// PathParams/BodyParams/ArrayParams, matching by name - called from
// reinitRequestPane right after those are (re)built from request's
// schema, so any request that carries ParamValues (a saved request
// reloaded, or the last-sent request restored on app start - see
// http.Request.ParamValues) shows exactly what was filled in before,
// not just blank/default fields. A name with no entry in values is left
// at whatever its schema default (or blank) already put it at.
func (m *RequestPane) applyParamValues(values http.ParamValues) {
	for i := range m.PathParams {
		if v, ok := values.PathParamValues[m.PathParams[i].Name]; ok {
			m.PathParams[i].SetValue(v)
		}
	}
	for i := range m.BodyParams {
		if v, ok := values.BodyParamValues[m.BodyParams[i].Name]; ok {
			m.BodyParams[i].SetValue(v)
		}
	}
	for ai := range m.ArrayParams {
		items, ok := values.ArrayParamValues[m.ArrayParams[ai].Name]
		if !ok {
			continue
		}
		// Grow/shrink to the saved item count before applying per-item
		// values, mirroring what the user had via the count control (see
		// ArrayCountControl) rather than just filling in however many
		// items newArrayParam happened to start with. This changes the
		// focus cycle's shape, which is why reinitRequestPane's own
		// rebuildFocusManager call runs *after* this, not before.
		for len(m.ArrayParams[ai].Items) < len(items) {
			m.ArrayParams[ai].AddItem()
		}
		for len(m.ArrayParams[ai].Items) > len(items) {
			m.ArrayParams[ai].RemoveLastItem()
		}
		for itemIdx, itemValues := range items {
			for fi := range m.ArrayParams[ai].Items[itemIdx].Fields {
				name := m.ArrayParams[ai].Items[itemIdx].Fields[fi].Name
				if v, ok := itemValues[name]; ok {
					m.ArrayParams[ai].Items[itemIdx].Fields[fi].SetValue(v)
				}
			}
		}
	}
}

// ApplySnapshot restores a previously captured request (see Snapshot) -
// identical to loading any saved/imported request (reinitRequestPane
// already restores ParamValues as part of every normal load), kept as its
// own exported entry point for callers outside this package (see
// app.SetupModel's restore of the last request sent).
func (m *RequestPane) ApplySnapshot(request *http.Request) {
	m.reinitRequestPane(request)
}

// clearParamValues resets every path/body/array param back to its
// schema's own defaults, discarding anything the user typed over them -
// bound to ClearParams ("ctrl+x"). Unlike ResetRequest ("ctrl+r", which
// replaces the whole request with a brand new blank one), this keeps the
// request's own identity (schema/URL/method/name) and only clears its
// *values* - e.g. to start filling a form over from scratch, or to save
// it back without whatever was typed in before (see SaveRequest).
func (m *RequestPane) clearParamValues() {
	req := *m.Request
	req.ParamValues = http.ParamValues{}
	m.reinitRequestPane(&req)
}
