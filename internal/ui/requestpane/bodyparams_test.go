package requestpane

import (
	"encoding/json"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
)

// schemaBody builds a Request.Body value in the same wrapper shape `volt
// import` writes, for testing without going through the cli package.
func schemaBody(t *testing.T, fields map[string]http.BodyFieldSchema) string {
	t.Helper()
	data, err := json.Marshal(map[string]interface{}{http.BodySchemaKey: fields})
	if err != nil {
		t.Fatalf("failed to build schema body: %v", err)
	}
	return string(data)
}

func TestExtractBodySpecs(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []bodySpec
	}{
		{"empty body", "", nil},
		{"empty object", "{}", nil},
		{"not an object", `["a","b"]`, nil},
		{"not valid JSON", "not json", nil},
		{"real data present falls back to freeform", `{"name":"actual value"}`, nil},
		{
			"untyped fields, sorted",
			schemaBody(t, map[string]http.BodyFieldSchema{"color": {}, "name": {}}),
			[]bodySpec{{Name: "color"}, {Name: "name"}},
		},
		{
			"typed fields",
			schemaBody(t, map[string]http.BodyFieldSchema{"age": {Type: "integer"}, "name": {Type: "string"}}),
			[]bodySpec{{Name: "age", Type: "integer"}, {Name: "name", Type: "string"}},
		},
		{
			"field with a default",
			schemaBody(t, map[string]http.BodyFieldSchema{"count": {Type: "integer", Default: "1"}}),
			[]bodySpec{{Name: "count", Type: "integer", Default: "1"}},
		},
		{
			"field with an enum",
			schemaBody(t, map[string]http.BodyFieldSchema{"status": {Type: "string", Enum: []string{"a", "b"}}}),
			[]bodySpec{{Name: "status", Type: "string", Enum: []string{"a", "b"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBodySpecs(tt.body)
			if len(got) != len(tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
			for i := range got {
				if got[i].Name != tt.want[i].Name || got[i].Type != tt.want[i].Type || got[i].Default != tt.want[i].Default {
					t.Errorf("got %+v, want %+v", got, tt.want)
				}
				if len(got[i].Enum) != len(tt.want[i].Enum) {
					t.Errorf("got enum %v, want %v", got[i].Enum, tt.want[i].Enum)
					continue
				}
				for j := range got[i].Enum {
					if got[i].Enum[j] != tt.want[i].Enum[j] {
						t.Errorf("got enum %v, want %v", got[i].Enum, tt.want[i].Enum)
					}
				}
			}
		})
	}
}

// TestRequestPane_SyncBodyParams_ExtractsFromImportedTemplate verifies that
// loading a request whose Body is a `volt import`-style schema template
// surfaces each field as its own field, sorted by name.
func TestRequestPane_SyncBodyParams_ExtractsFromImportedTemplate(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {}, "color": {}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 2 {
		t.Fatalf("expected 2 body params, got %+v", pane.BodyParams)
	}
	if pane.BodyParams[0].Name != "color" || pane.BodyParams[1].Name != "name" {
		t.Errorf("expected sorted [color name], got %+v", pane.BodyParams)
	}
}

// TestRequestPane_SyncBodyParams_CarriesTypeThrough verifies a field's
// declared type shows up on the resulting Param, and pre-fills the input's
// placeholder with it too.
func TestRequestPane_SyncBodyParams_CarriesTypeThrough(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"age": {Type: "integer"}, "name": {Type: "string"}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 2 {
		t.Fatalf("expected 2 body params, got %+v", pane.BodyParams)
	}
	byName := map[string]Param{pane.BodyParams[0].Name: pane.BodyParams[0], pane.BodyParams[1].Name: pane.BodyParams[1]}

	if byName["age"].Type != "integer" {
		t.Errorf("expected age's type to be integer, got %q", byName["age"].Type)
	}
	if byName["age"].Input.Placeholder != "integer" {
		t.Errorf("expected age's input placeholder to be integer, got %q", byName["age"].Input.Placeholder)
	}
	if byName["name"].Type != "string" {
		t.Errorf("expected name's type to be string, got %q", byName["name"].Type)
	}
}

// TestRequestPane_SyncBodyParams_DescriptionReplacesTypeAsPlaceholder
// verifies a field with a schema description uses it as the input's
// placeholder instead of the bare type name - falling back to the type
// only when the schema has no description at all (see
// paramFromFieldSchema).
func TestRequestPane_SyncBodyParams_DescriptionReplacesTypeAsPlaceholder(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"age":  {Type: "integer", Description: "The user's age in years."},
		"name": {Type: "string"},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	byName := map[string]Param{pane.BodyParams[0].Name: pane.BodyParams[0], pane.BodyParams[1].Name: pane.BodyParams[1]}

	if got := byName["age"].Input.Placeholder; got != "The user's age in years." {
		t.Errorf("expected age's placeholder to be its description, got %q", got)
	}
	if got := byName["name"].Input.Placeholder; got != "string" {
		t.Errorf("expected name (no description) to fall back to its type as placeholder, got %q", got)
	}
}

// TestRequestPane_SyncBodyParams_PrefillsDefault verifies a field with a
// schema "default" starts pre-filled with that value, ready to send as-is.
func TestRequestPane_SyncBodyParams_PrefillsDefault(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"count": {Type: "integer", Default: "5"}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 1 {
		t.Fatalf("expected 1 body param, got %+v", pane.BodyParams)
	}
	if got := pane.BodyParams[0].Value(); got != "5" {
		t.Errorf("expected count to default to %q, got %q", "5", got)
	}
}

// TestRequestPane_SyncBodyParams_EnumBecomesPicker verifies a field with a
// schema "enum" gets an EnumSelector (not a free-text Input), starting on
// the schema's default when the default is one of the enum values.
func TestRequestPane_SyncBodyParams_EnumBecomesPicker(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"status": {Type: "string", Enum: []string{"active", "inactive"}, Default: "inactive"},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 1 {
		t.Fatalf("expected 1 body param, got %+v", pane.BodyParams)
	}
	status := pane.BodyParams[0]
	if status.Enum == nil {
		t.Fatal("expected status to get an EnumSelector")
	}
	if got := status.Value(); got != "inactive" {
		t.Errorf("expected the picker to start on the schema default %q, got %q", "inactive", got)
	}

	status.Enum.Next()
	if got := status.Value(); got != "active" {
		t.Errorf("expected Next() to cycle to %q, got %q", "active", got)
	}
	status.Enum.Next()
	if got := status.Value(); got != "inactive" {
		t.Errorf("expected Next() to wrap back around to %q, got %q", "inactive", got)
	}
}

// TestRequestPane_ResolvedRequest_BuildsBodyFromParams verifies values
// typed into body param fields end up in the JSON body actually sent,
// while the stored template is left untouched for next time.
func TestRequestPane_ResolvedRequest_BuildsBodyFromParams(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {}, "color": {}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	for i := range pane.BodyParams {
		switch pane.BodyParams[i].Name {
		case "name":
			pane.BodyParams[i].Input.SetValue("widget-1")
		case "color":
			pane.BodyParams[i].Input.SetValue("red")
		}
	}

	resolved := pane.resolvedRequest()
	var got map[string]string
	if err := json.Unmarshal([]byte(resolved.Body), &got); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v (body=%q)", err, resolved.Body)
	}
	if got["name"] != "widget-1" || got["color"] != "red" {
		t.Errorf("expected {name:widget-1, color:red}, got %v", got)
	}

	if pane.Request.Body != req.Body {
		t.Errorf("expected the stored request to keep the schema template, got %q", pane.Request.Body)
	}
}

// TestRequestPane_ResolvedRequest_BuildsBodyFromEnum verifies an enum
// picker's current selection ends up in the JSON body actually sent. This
// schema declares no default, so the picker starts unset (see
// EnumSelector.allowUnset) - Next() once reaches the first real option
// ("active"), Next() again reaches "inactive".
func TestRequestPane_ResolvedRequest_BuildsBodyFromEnum(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"status": {Enum: []string{"active", "inactive"}}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.BodyParams[0].Enum.Next()
	pane.BodyParams[0].Enum.Next()

	resolved := pane.resolvedRequest()
	var got map[string]string
	if err := json.Unmarshal([]byte(resolved.Body), &got); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v (body=%q)", err, resolved.Body)
	}
	if got["status"] != "inactive" {
		t.Errorf("expected status=inactive after Next(), got %v", got)
	}
}

// TestRequestPane_SyncRequest_DoesNotClobberBodyParamsTemplate guards
// against a real bug risk: syncRequest (which runs on nearly every Update
// tick) must not overwrite an imported request's body-schema template with
// "{}" from the now-hidden, empty freeform Body textarea.
func TestRequestPane_SyncRequest_DoesNotClobberBodyParamsTemplate(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {}, "color": {}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	pane.syncRequest()

	if len(pane.BodyParams) != 2 {
		t.Fatalf("expected body params to survive a sync tick, got %+v", pane.BodyParams)
	}
	if pane.Request.Body != req.Body {
		t.Errorf("expected the body template to survive a sync tick byte-for-byte unchanged, got %q", pane.Request.Body)
	}
}

// TestRequestPane_SyncRequest_StillUsesFreeformBodyWhenNoSchema is the
// regression guard for ordinary (non-imported) requests: with no body
// params, the freeform Body textarea must keep driving the request body
// exactly as before this feature existed.
func TestRequestPane_SyncRequest_StillUsesFreeformBodyWhenNoSchema(t *testing.T) {
	pane, _ := setupTestPane(t)

	pane.Body.SetValue("key = value")
	pane.syncRequest()

	if len(pane.BodyParams) != 0 {
		t.Fatalf("expected no body params without a schema, got %+v", pane.BodyParams)
	}
	if pane.Request.Body != `{"key":"value"}` {
		t.Errorf("expected the freeform textarea to drive the body, got %q", pane.Request.Body)
	}
}

// TestRequestPane_ReopenSavedRequestWithRealBodyStaysFreeform guards
// against a real bug caught while building this: a request saved earlier
// (the normal way, not via import) has real, non-empty body values by the
// time it's reopened. It must stay editable as plain text - not get
// swapped for blank fill-in fields that would hide the data that's
// already there.
func TestRequestPane_ReopenSavedRequestWithRealBodyStaysFreeform(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: `{"key":"value"}`, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 0 {
		t.Fatalf("expected no body params for a body with real data, got %+v", pane.BodyParams)
	}
}

// TestRequestPane_SyncRequest_TypingEmptyValuesDoesNotFlipToBodyParams
// guards the other half of the same bug: since body params are derived
// once at load/create time and never re-derived while typing, a user
// hand-typing "a = ,b = " into the freeform Body textarea (empty values,
// e.g. mid-edit before filling them in) must not suddenly have the
// textarea replaced by structured fields.
func TestRequestPane_SyncRequest_TypingEmptyValuesDoesNotFlipToBodyParams(t *testing.T) {
	pane, _ := setupTestPane(t)

	pane.Body.SetValue("a = ,b = ")
	pane.syncRequest()

	if len(pane.BodyParams) != 0 {
		t.Fatalf("expected typing in the freeform textarea to never produce body params, got %+v", pane.BodyParams)
	}
}

// TestRequestPane_PathAndBodyParams_Coexist verifies a POST with both a
// {id} path placeholder and a JSON body schema shows and resolves both
// independently.
func TestRequestPane_PathAndBodyParams_Coexist(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"name": {}})
	req := &http.Request{
		Method:  http.POST,
		URL:     "https://api.example.com/widgets/{id}",
		Body:    body,
		Headers: map[string]string{},
	}
	pane.reinitRequestPane(req)

	if len(pane.PathParams) != 1 || len(pane.BodyParams) != 1 {
		t.Fatalf("expected 1 path param and 1 body param, got path=%+v body=%+v", pane.PathParams, pane.BodyParams)
	}
	pane.PathParams[0].Input.SetValue("7")
	pane.BodyParams[0].Input.SetValue("widget-7")

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/widgets/7" {
		t.Errorf("expected substituted URL, got %q", resolved.URL)
	}
	var respBody map[string]string
	if err := json.Unmarshal([]byte(resolved.Body), &respBody); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v", err)
	}
	if respBody["name"] != "widget-7" {
		t.Errorf("expected name=widget-7, got %v", respBody)
	}
}

// TestRequestPane_BooleanBodyParam_GetsATrueFalseSwitch verifies a
// "boolean"-typed field gets a fixed-choice true/false picker instead of a
// free-text field - so it can't hold anything but a real boolean.
func TestRequestPane_BooleanBodyParam_GetsATrueFalseSwitch(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{"active": {Type: "boolean", Default: "true"}})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 1 {
		t.Fatalf("expected 1 body param, got %+v", pane.BodyParams)
	}
	p := pane.BodyParams[0]
	if p.Enum == nil {
		t.Fatal("expected a boolean field to get an Enum (true/false) picker, not a free-text Input")
	}
	if p.Enum.Value() != "true" {
		t.Errorf("expected the schema's default to be reflected, got %q", p.Enum.Value())
	}

	p.Enum.Next()
	if p.Enum.Value() != "false" {
		t.Errorf("expected cycling to reach the other boolean value, got %q", p.Enum.Value())
	}
}

// TestBuildRequestBody_RespectsDeclaredTypes verifies boolean/integer/number
// fields are serialized as real JSON types rather than every value being
// sent as a JSON string, which would misrepresent them to a server that
// actually validates the request body's shape.
func TestBuildRequestBody_RespectsDeclaredTypes(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"active": {Type: "boolean", Default: "true"},
		"count":  {Type: "integer"},
		"price":  {Type: "number"},
		"name":   {Type: "string"},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	for i := range pane.BodyParams {
		switch pane.BodyParams[i].Name {
		case "count":
			pane.BodyParams[i].Input.SetValue("5")
		case "price":
			pane.BodyParams[i].Input.SetValue("4.5")
		case "name":
			pane.BodyParams[i].Input.SetValue("widget")
		}
	}
	// "active" is left on its schema default (Enum, starts at "true" - a
	// boolean with no declared default starts unset instead and is omitted
	// from the body entirely, see TestBuildRequestBody_OmitsUntouchedOptionalFields).

	resolved := pane.resolvedRequest()

	var got map[string]interface{}
	if err := json.Unmarshal([]byte(resolved.Body), &got); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v (body=%q)", err, resolved.Body)
	}
	if _, ok := got["active"].(bool); !ok {
		t.Errorf("expected active to be a JSON boolean, got %T (%v)", got["active"], got["active"])
	}
	if n, ok := got["count"].(float64); !ok || n != 5 {
		t.Errorf("expected count to be the JSON number 5, got %T (%v)", got["count"], got["count"])
	}
	if n, ok := got["price"].(float64); !ok || n != 4.5 {
		t.Errorf("expected price to be the JSON number 4.5, got %T (%v)", got["price"], got["price"])
	}
	if s, ok := got["name"].(string); !ok || s != "widget" {
		t.Errorf("expected name to stay a JSON string, got %T (%v)", got["name"], got["name"])
	}
}

// TestBuildRequestBody_OmitsUntouchedOptionalFields is a regression test
// for a real bug report: filling in only a form's required fields still
// sent every other declared field too - empty strings/zero values for
// anything left blank - because BuildRequestBody included every BodyParam
// unconditionally. Only a field the user actually filled in, or that
// started pre-filled from the schema's own default, should reach the
// server; a field with neither should be left out of the JSON entirely.
// This now also covers Enum/boolean pickers: one with a schema default
// behaves like before (always has a real value, always included), but one
// with *no* declared default starts on no value at all (see
// EnumSelector.allowUnset) and is omitted too, unless actually cycled to a
// real choice.
func TestBuildRequestBody_OmitsUntouchedOptionalFields(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"name":       {Type: "string", Required: true},                                   // required, filled in below
		"email":      {Type: "string"},                                                   // optional text, left untouched - must be omitted
		"nickname":   {Type: "string", Default: "anon"},                                  // optional text with a schema default - must be included
		"role":       {Type: "string", Enum: []string{"user", "admin"}, Default: "user"}, // enum with a default - must be included
		"permission": {Type: "string", Enum: []string{"read", "write"}},                  // enum with NO default - untouched, must be omitted
		"active":     {Type: "boolean", Default: "true"},                                 // boolean with a default - must be included
		"subscribed": {Type: "boolean"},                                                  // boolean with NO default - untouched, must be omitted
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/accounts", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 7 {
		t.Fatalf("expected 7 body params, got %+v", pane.BodyParams)
	}
	var subscribed *Param
	for i := range pane.BodyParams {
		switch pane.BodyParams[i].Name {
		case "name":
			pane.BodyParams[i].Input.SetValue("acme")
		case "subscribed":
			subscribed = &pane.BodyParams[i]
		}
	}
	if subscribed == nil {
		t.Fatal("test setup error: no \"subscribed\" body param found")
	}
	if got := subscribed.Value(); got != "" {
		t.Fatalf("expected an untouched boolean with no schema default to start unset, got %q", got)
	}
	// "email"/"permission" left blank on purpose too - the fields with
	// neither a value nor a default.

	resolved := pane.resolvedRequest()
	var got map[string]interface{}
	if err := json.Unmarshal([]byte(resolved.Body), &got); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v (body=%q)", err, resolved.Body)
	}

	for _, wantPresent := range []string{"name", "nickname", "role", "active"} {
		if _, ok := got[wantPresent]; !ok {
			t.Errorf("expected %q to be present in the sent body, got %v", wantPresent, got)
		}
	}
	for _, wantAbsent := range []string{"email", "permission", "subscribed"} {
		if _, ok := got[wantAbsent]; ok {
			t.Errorf("expected untouched field %q (no value, no default) to be omitted, got %v", wantAbsent, got)
		}
	}
	if got["name"] != "acme" {
		t.Errorf("expected name=%q, got %v", "acme", got["name"])
	}
	if got["nickname"] != "anon" {
		t.Errorf("expected nickname to keep its schema default %q, got %v", "anon", got["nickname"])
	}
	if got["role"] != "user" {
		t.Errorf("expected role to keep its enum default %q, got %v", "user", got["role"])
	}
	if b, ok := got["active"].(bool); !ok || !b {
		t.Errorf("expected active to keep its boolean default true, got %T (%v)", got["active"], got["active"])
	}

	// Actually cycling the untouched picker to a real value must then
	// include it.
	subscribed.Enum.Next()
	if got := subscribed.Value(); got == "" {
		t.Fatal("expected Next() to move off the unset state")
	}
	resolved = pane.resolvedRequest()
	var gotAfter map[string]interface{}
	if err := json.Unmarshal([]byte(resolved.Body), &gotAfter); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v (body=%q)", err, resolved.Body)
	}
	if _, ok := gotAfter["subscribed"].(bool); !ok {
		t.Errorf("expected subscribed to be included as a JSON boolean once explicitly set, got %T (%v)", gotAfter["subscribed"], gotAfter["subscribed"])
	}
}
