package requestpane

import (
	"testing"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
)

// TestNormalMode_ComponentsMatchRenderOrder locks in that tab order follows
// the same top-to-bottom sequence fieldsContent() renders in view.go
// (name, method+URL, path params, body params, headers, submit - no
// server field here, since this request has no group - see
// ShowServerSelector/TestNormalMode_ComponentsMatchRenderOrder_Grouped).
// The two lists drifting apart previously made tab jump around the screen
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
		pane.NameInput,
		pane.MethodSelector,
		pane.URLInput,
		&pane.PathParams[0].Input,
		&pane.BodyParams[0].Input,
		&pane.BodyParams[1].Input,
		pane.Headers,
		pane.SubmitButton,
	}
	assertSameFocusOrder(t, components, want)
}

// TestNormalMode_ComponentsMatchRenderOrder_Grouped is the grouped
// counterpart: a request that belongs to a group does get a server field,
// as the very first one (see ShowServerSelector).
func TestNormalMode_ComponentsMatchRenderOrder_Grouped(t *testing.T) {
	pane, db := setupTestPane(t)

	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	req := &http.Request{Method: http.GET, URL: "/items", Headers: map[string]string{}, GroupID: &group.ID}
	pane.reinitRequestPane(req)

	nm := &NormalMode{}
	components := nm.components(&pane)

	want := []ui.Focusable{
		pane.ServerSelector,
		pane.NameInput,
		pane.MethodSelector,
		pane.URLInput,
		pane.Body,
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
		pane.NameInput,
		pane.MethodSelector,
		pane.URLInput,
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
		pane.NameInput,
		pane.MethodSelector,
		pane.URLInput,
		&pane.PathParams[0].Input,
		pane.Body,
		pane.Headers,
		pane.SubmitButton,
	}
	assertSameFocusOrder(t, components, want)
}

// TestReinitRequestPane_BlursStaleFocusFromParamShapeRebuild is a regression
// test for a real bug: loading a different request while Name was focused
// left Name visibly showing a live (non-blurred) cursor even after focus
// had moved on to Method/a path param. The cause: reinitRequestPane blurs
// whatever's focused *once*, up front, but syncPathParams/syncQueryParams/
// syncBodyParams/applyParamValues below that each call rebuildFocusManager
// whenever the new request's param shape differs from the old one (the
// normal case when switching to a genuinely different request) -
// rebuildFocusManager "preserves" focus by *index*, which reads back
// whatever index Name happened to be at (that index isn't itself cleared
// by a Blur() call) and re-focuses it, stomping the earlier blur. Reported
// as: "видим курсор и выделение похожее на курсор в поле name" - a
// leftover cursor-like highlight sitting in Name while typing actually
// landed in a completely different field (confirmed live: arrow keys
// changed the Method selector, not Name, while Name still looked focused).
func TestReinitRequestPane_BlursStaleFocusFromParamShapeRebuild(t *testing.T) {
	pane, _ := setupTestPane(t)

	// SetupRequestPane starts focus on Method (see defaultFocusIndex) -
	// move it back to Name, exactly as a user would with shift+tab.
	pane.FocusManager.Prev()
	if !pane.NameInput.Focused() {
		t.Fatalf("expected Name to be focused before reinit")
	}

	// A request with a path param the previous (blank) one didn't have -
	// this is what makes syncPathParams' internal rebuildFocusManager
	// actually fire (a no-op, param-shape-unchanged reinit wouldn't
	// reproduce the bug).
	req := &http.Request{Method: http.GET, URL: "https://api.example.com/widgets/{id}", Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if pane.NameInput.Focused() {
		t.Error("expected Name to be blurred after loading a request with a different param shape, but it's still focused")
	}
	if _, ok := pane.FocusManager.Current().(*ui.MethodSelector); !ok {
		t.Errorf("expected MethodSelector (defaultFocusIndex) to be the focused field, got %T", pane.FocusManager.Current())
	}
}

// TestReinitRequestPane_ResetsCursorToEndOfNewValue is a regression test for
// the other half of the same report: even once a field is correctly
// blurred (see TestReinitRequestPane_BlursStaleFocusFromParamShapeRebuild),
// SetValue only moves the cursor if the old position would otherwise land
// past the *new* value's end - a shorter previous value's cursor position
// easily stays a valid, but wrong, mid-string index into a longer new one.
// A blurred cursor still renders at that leftover position (textinput's
// View() always draws it, just in a plainer style than the focused
// reverse-video one) - one character subtly off-color from the rest of
// the field. Reported as: "в url... сам url имеет один символ другого
// цвета".
func TestReinitRequestPane_ResetsCursorToEndOfNewValue(t *testing.T) {
	pane, _ := setupTestPane(t)

	pane.URLInput.SetValue("https://short.example")
	pane.URLInput.SetCursor(5) // mid-string, as if the user had typed and left it there

	pane.reinitRequestPane(&http.Request{
		Method:  http.GET,
		URL:     "https://api.example.com/widgets/long-path-here",
		Headers: map[string]string{},
	})

	if got, want := pane.URLInput.Position(), len("https://api.example.com/widgets/long-path-here"); got != want {
		t.Errorf("expected the cursor to land at the end of the new URL (%d), got %d", want, got)
	}
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
