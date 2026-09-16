package requestpane

import (
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
)

// TestNormalMode_ComponentsMatchRenderOrder locks in that tab order follows
// the same top-to-bottom sequence fieldsContent() renders in view.go
// (server, method+URL, name, path params, body params, headers, submit). The
// two lists drifting apart previously made tab jump around the screen
// instead of moving straight down it.
func TestNormalMode_ComponentsMatchRenderOrder(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"count": {Type: "integer"},
		"name":  {Type: "string"},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/items/{item_id}", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.PathParams) != 1 {
		t.Fatalf("expected 1 path param, got %+v", pane.PathParams)
	}
	if len(pane.BodyParams) != 2 {
		t.Fatalf("expected 2 body params, got %+v", pane.BodyParams)
	}

	nm := &NormalMode{}
	components := nm.components(&pane)

	want := []ui.Focusable{
		pane.ServerSelector,
		pane.MethodSelector,
		pane.URLInput,
		pane.NameInput,
		&pane.PathParams[0].Input,
		&pane.BodyParams[0].Input,
		&pane.BodyParams[1].Input,
		pane.Headers,
		pane.SubmitButton,
	}
	assertSameFocusOrder(t, components, want)
}

func TestLoadTestMode_ComponentsMatchRenderOrder(t *testing.T) {
	pane, _ := setupTestPane(t)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"count": {Type: "integer"},
		"name":  {Type: "string"},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/items/{item_id}", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	ltm := &LoadTestMode{}
	components := ltm.components(&pane)

	want := []ui.Focusable{
		pane.ServerSelector,
		pane.MethodSelector,
		pane.URLInput,
		pane.NameInput,
		&pane.PathParams[0].Input,
		&pane.BodyParams[0].Input,
		&pane.BodyParams[1].Input,
		pane.Headers,
		pane.LoadTestConcurrency,
		pane.LoadTestTotalReqs,
		pane.LoadTestQPS,
		pane.LoadTestTimeout,
		pane.SubmitButton,
	}
	assertSameFocusOrder(t, components, want)
}

// TestNormalMode_ComponentsFallsBackToFreeformBody covers a request with no
// known body schema, which uses the plain Body textarea instead of
// BodyParams - it must still sit between path params and headers.
func TestNormalMode_ComponentsFallsBackToFreeformBody(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{Method: http.GET, URL: "https://api.example.com/items/{item_id}", Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.PathParams) != 1 {
		t.Fatalf("expected 1 path param, got %+v", pane.PathParams)
	}
	if len(pane.BodyParams) != 0 {
		t.Fatalf("expected no body params, got %+v", pane.BodyParams)
	}

	nm := &NormalMode{}
	components := nm.components(&pane)

	want := []ui.Focusable{
		pane.ServerSelector,
		pane.MethodSelector,
		pane.URLInput,
		pane.NameInput,
		&pane.PathParams[0].Input,
		pane.Body,
		pane.Headers,
		pane.SubmitButton,
	}
	assertSameFocusOrder(t, components, want)
}

func assertSameFocusOrder(t *testing.T, got, want []ui.Focusable) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("focus order length mismatch: got %d fields, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("focus index %d: got %s, want %s", i, describeFocusable(got[i]), describeFocusable(want[i]))
		}
	}
}

func describeFocusable(f ui.Focusable) string {
	switch v := f.(type) {
	case *ui.MethodSelector:
		return "MethodSelector"
	case *textinput.Model:
		return "textinput(" + v.Value() + ")"
	case *textarea.Model:
		return "textarea(" + v.Value() + ")"
	case *ui.SubmitButton:
		return "SubmitButton"
	case *EnumSelector:
		return "EnumSelector(" + v.Value() + ")"
	default:
		return "unknown"
	}
}
