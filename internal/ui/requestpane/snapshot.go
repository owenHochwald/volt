package requestpane

import "github.com/owenHochwald/Volt/internal/http"

// Snapshot captures this pane's current request template plus every
// currently typed/selected field value (see http.RequestSnapshot) - used
// to persist "the last sent request, and all its modified fields" (see
// mode_normal.go/mode_loadtest.go's handleSubmit) so it can be restored
// after the app restarts (see ApplySnapshot). Only ever called once a send
// has actually passed validation - never for a request that's merely
// being edited.
func (m *RequestPane) Snapshot() http.RequestSnapshot {
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

	return http.RequestSnapshot{
		Request:          *m.Request,
		PathParamValues:  pathValues,
		BodyParamValues:  bodyValues,
		ArrayParamValues: arrayValues,
	}
}

// ApplySnapshot restores a previously captured Snapshot: loads its base
// Request template exactly like loading any saved/imported request (see
// reinitRequestPane), then overlays each saved field value back onto the
// freshly-built params, matching by name, so editing can continue right
// where it left off instead of only showing schema defaults again.
func (m *RequestPane) ApplySnapshot(snapshot *http.RequestSnapshot) {
	req := snapshot.Request
	m.reinitRequestPane(&req)

	for i := range m.PathParams {
		if v, ok := snapshot.PathParamValues[m.PathParams[i].Name]; ok {
			m.PathParams[i].SetValue(v)
		}
	}
	for i := range m.BodyParams {
		if v, ok := snapshot.BodyParamValues[m.BodyParams[i].Name]; ok {
			m.BodyParams[i].SetValue(v)
		}
	}
	for ai := range m.ArrayParams {
		items, ok := snapshot.ArrayParamValues[m.ArrayParams[ai].Name]
		if !ok {
			continue
		}
		// Grow/shrink to the saved item count before applying per-item
		// values, mirroring what the user had via the count control (see
		// ArrayCountControl) rather than just filling in however many
		// items newArrayParam happened to start with.
		for len(m.ArrayParams[ai].Items) < len(items) {
			m.ArrayParams[ai].AddItem()
		}
		for len(m.ArrayParams[ai].Items) > len(items) {
			m.ArrayParams[ai].RemoveLastItem()
		}
		for itemIdx, values := range items {
			for fi := range m.ArrayParams[ai].Items[itemIdx].Fields {
				name := m.ArrayParams[ai].Items[itemIdx].Fields[fi].Name
				if v, ok := values[name]; ok {
					m.ArrayParams[ai].Items[itemIdx].Fields[fi].SetValue(v)
				}
			}
		}
	}
	// PathParams/BodyParams/ArrayParams' underlying Input/Enum instances
	// weren't replaced (only their values changed in place), but
	// growing/shrinking an ArrayParam above changes its item count, which
	// changes the focus cycle's shape - rebuild it so tab/shift+tab visits
	// exactly the restored fields.
	m.rebuildFocusManager()
}
