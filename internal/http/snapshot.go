package http

// RequestSnapshot captures a request pane's exact editing state at the
// moment a request was actually sent - the base Request template
// (method/URL with {name} placeholders, schema-driven Body, headers,
// group/server association, path param schema) plus every currently
// typed or selected field value, keyed by name - so restoring it (see
// requestpane.RequestPane.ApplySnapshot) reproduces not just which
// request was open, but exactly what was filled into each of its fields,
// not just their schema defaults. Persisted as a single row (see
// storage.SaveLastSentRequest/LoadLastSentRequest) so it survives an app
// restart; only ever written once a send actually goes out - a request
// that's merely being edited, never sent, has nothing worth restoring.
type RequestSnapshot struct {
	Request Request `json:"request"`
	// PathParamValues/BodyParamValues hold each scalar path/body param's
	// current value (whichever of Input/Enum drives it - see
	// requestpane.Param.Value), keyed by name. A name with no entry here
	// (never typed into, or matching its schema default already) is left
	// to fall back to whatever Request/its schema would produce on its
	// own.
	PathParamValues map[string]string `json:"path_param_values,omitempty"`
	BodyParamValues map[string]string `json:"body_param_values,omitempty"`
	// ArrayParamValues holds each array-typed body param's current items,
	// keyed by array name - one map per item (field name -> value for a
	// "complex" array's named sub-fields; a single entry keyed by "" for a
	// "simple" array's lone unnamed scalar field). The number of items
	// itself is part of the snapshot too (via len of the slice) - applying
	// it grows/shrinks the array to match, the same as the user pressing
	// the count control would.
	ArrayParamValues map[string][]map[string]string `json:"array_param_values,omitempty"`
}
