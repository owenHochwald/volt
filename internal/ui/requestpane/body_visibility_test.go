package requestpane

import (
	"strings"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
)

// TestRequestPane_IsSchemaRequest verifies the PathParamSchema != nil
// marker - set by import (see internal/cli/openapi.go's pathParamFields)
// even to an empty map, cleared back to nil by clearParamValues - is what
// IsSchemaRequest actually reports.
func TestRequestPane_IsSchemaRequest(t *testing.T) {
	pane, _ := setupTestPane(t)

	pane.Request = &http.Request{Method: http.GET, URL: "http://x"}
	if pane.IsSchemaRequest() {
		t.Error("expected a hand-typed request (nil PathParamSchema) to not be a schema request")
	}

	pane.Request = &http.Request{Method: http.GET, URL: "http://x", PathParamSchema: map[string]http.BodyFieldSchema{}}
	if !pane.IsSchemaRequest() {
		t.Error("expected a request with a non-nil (even empty) PathParamSchema to be a schema request")
	}
}

// TestRequestPane_HidesBodyField_ForSchemaRequestWithNoBodyData is a
// regression test for a real feature request: an imported request whose
// schema described no body fields at all (e.g. a bodyless POST endpoint)
// must not show an empty freeform Body field inviting text syncRequest's
// own schema-aware path won't even look at - the field, and its focus
// stop, should simply not be there.
func TestRequestPane_HidesBodyField_ForSchemaRequestWithNoBodyData(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{
		Method:          http.POST,
		URL:             "http://x/no-body",
		Headers:         map[string]string{},
		PathParamSchema: map[string]http.BodyFieldSchema{},
	}
	pane.reinitRequestPane(req)

	if len(pane.BodyParams) != 0 || len(pane.ArrayParams) != 0 {
		t.Fatalf("test setup error: expected no structured body fields, got body=%+v array=%+v", pane.BodyParams, pane.ArrayParams)
	}

	content, _ := pane.fieldsContent()
	if strings.Contains(content, "Body") {
		t.Errorf("expected no Body section for a schema request with no body data, got:\n%s", content)
	}

	for _, c := range (&NormalMode{}).components(&pane) {
		if c == pane.Body {
			t.Error("expected the Body field to be excluded from the focus cycle")
		}
	}
}

// TestRequestPane_ShowsBodyField_ForHandTypedRequest verifies the freeform
// Body field still shows (and stays in the focus cycle) for a plain,
// never-imported request - the only counterpart a schema-less request has
// for setting a body at all.
func TestRequestPane_ShowsBodyField_ForHandTypedRequest(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{Method: http.POST, URL: "http://x", Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	content, _ := pane.fieldsContent()
	if !strings.Contains(content, "Body") {
		t.Errorf("expected a Body section for a hand-typed request, got:\n%s", content)
	}

	found := false
	for _, c := range (&NormalMode{}).components(&pane) {
		if c == pane.Body {
			found = true
		}
	}
	if !found {
		t.Error("expected the Body field to be in the focus cycle for a hand-typed request")
	}
}
