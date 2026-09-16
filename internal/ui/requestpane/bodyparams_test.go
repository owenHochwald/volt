package requestpane

import (
	"encoding/json"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
)

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
		{"untyped fields, sorted", `{"color":"","name":""}`, []bodySpec{{Name: "color"}, {Name: "name"}}},
		{"typed fields", `{"age":"<integer>","name":"<string>"}`, []bodySpec{{Name: "age", Type: "integer"}, {Name: "name", Type: "string"}}},
		{"mix of typed and untyped", `{"count":"<integer>","note":""}`, []bodySpec{{Name: "count", Type: "integer"}, {Name: "note"}}},
		{"real data present falls back to freeform", `{"name":"actual value"}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBodySpecs(tt.body)
			if len(got) != len(tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

// TestRequestPane_SyncBodyParams_ExtractsFromImportedTemplate verifies that
// loading a request whose Body is a `volt import`-style JSON template
// (e.g. {"name":"","color":""}) surfaces each key as its own field.
func TestRequestPane_SyncBodyParams_ExtractsFromImportedTemplate(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: `{"name":"","color":""}`, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 2 {
		t.Fatalf("expected 2 body params, got %+v", pane.BodyParams)
	}
	if pane.BodyParams[0].Name != "color" || pane.BodyParams[1].Name != "name" {
		t.Errorf("expected sorted [color name], got %+v", pane.BodyParams)
	}
}

// TestRequestPane_SyncBodyParams_CarriesTypeThrough verifies a field's
// declared type (from `volt import`'s "<type>" marker) shows up on the
// resulting Param, and pre-fills the input's placeholder with it too.
func TestRequestPane_SyncBodyParams_CarriesTypeThrough(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: `{"age":"<integer>","name":"<string>"}`, Headers: map[string]string{}}
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

// TestRequestPane_ResolvedRequest_BuildsBodyFromParams verifies values
// typed into body param fields end up in the JSON body actually sent,
// while the stored template is left untouched for next time.
func TestRequestPane_ResolvedRequest_BuildsBodyFromParams(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: `{"name":"","color":""}`, Headers: map[string]string{}}
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
		t.Errorf("expected the stored request to keep the empty-value template, got %q", pane.Request.Body)
	}
}

// TestRequestPane_SyncRequest_DoesNotClobberBodyParamsTemplate guards
// against a real bug risk: syncRequest (which runs on nearly every Update
// tick) must not overwrite an imported request's body-param template with
// "{}" from the now-hidden, empty freeform Body textarea.
func TestRequestPane_SyncRequest_DoesNotClobberBodyParamsTemplate(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: `{"name":"","color":""}`, Headers: map[string]string{}}
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

	req := &http.Request{
		Method:  http.POST,
		URL:     "https://api.example.com/widgets/{id}",
		Body:    `{"name":""}`,
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
	var body map[string]string
	if err := json.Unmarshal([]byte(resolved.Body), &body); err != nil {
		t.Fatalf("resolved body isn't valid JSON: %v", err)
	}
	if body["name"] != "widget-7" {
		t.Errorf("expected name=widget-7, got %v", body)
	}
}
