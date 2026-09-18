package requestpane

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
)

func TestBodyInnerContent(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"empty (e.g. an OpenAPI-imported GET/DELETE request)", "", ""},
		{"single char", "{", ""},
		{"empty JSON object", "{}", ""},
		{"populated JSON object", `{"a":"b"}`, `"a":"b"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bodyInnerContent(tt.body); got != tt.want {
				t.Errorf("bodyInnerContent(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

// TestReinitRequestPane_EmptyBodyDoesNotPanic guards against a real crash:
// selecting a saved request whose Body is "" (as every `volt import`-created
// request's is, since GET/DELETE requests carry no body) used to panic with
// "slice bounds out of range [:-1]" in reinitRequestPane.
func TestReinitRequestPane_EmptyBodyDoesNotPanic(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{Method: http.GET, URL: "http://example.com/pets", Headers: map[string]string{}}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("reinitRequestPane panicked on an empty body: %v", r)
		}
	}()

	handled, cmd := pane.Update(ui.SetRequestPaneRequestMsg{Request: req})
	if cmd != nil {
		t.Errorf("expected no command, got one")
	}
	result, ok := handled.(RequestPane)
	if !ok {
		t.Fatalf("expected a RequestPane, got %T", handled)
	}
	if result.Body.Value() != "" {
		t.Errorf("expected an empty body textarea, got %q", result.Body.Value())
	}
}

// TestSetRequestPaneRequestMsg_FocusesPastMethodURL is a regression test
// for a real feature request: picking a request from the sidebar already
// tells you its method and URL (that's what you picked it *for* - it's
// right there in the list), so focus should skip past them to whatever's
// actually left to fill in - the first path param here.
func TestSetRequestPaneRequestMsg_FocusesPastMethodURL(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{
		Method:  http.GET,
		URL:     "https://api.example.com/hosts/{id}",
		Headers: map[string]string{},
	}

	model, _ := pane.Update(ui.SetRequestPaneRequestMsg{Request: req})
	result, ok := model.(RequestPane)
	if !ok {
		t.Fatalf("expected RequestPane, got %T", model)
	}
	if len(result.PathParams) != 1 {
		t.Fatalf("test setup error: expected 1 path param, got %+v", result.PathParams)
	}
	if result.FocusManager.Current() != &result.PathParams[0].Input {
		t.Errorf("expected focus on the first path param, got %T", result.FocusManager.Current())
	}
}

// TestSetRequestPaneRequestMsg_FocusesBodyWhenNoPathParams verifies the
// same skip-past-method-URL behavior lands on the freeform Body textarea
// when there are no path params and no known body schema - whatever
// comes right after method+URL, not specifically a path param.
func TestSetRequestPaneRequestMsg_FocusesBodyWhenNoPathParams(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{
		Method:  http.POST,
		URL:     "https://api.example.com/hosts",
		Headers: map[string]string{},
	}

	model, _ := pane.Update(ui.SetRequestPaneRequestMsg{Request: req})
	result, ok := model.(RequestPane)
	if !ok {
		t.Fatalf("expected RequestPane, got %T", model)
	}
	if len(result.PathParams) != 0 {
		t.Fatalf("test setup error: expected no path params, got %+v", result.PathParams)
	}
	if result.FocusManager.Current() != result.Body {
		t.Errorf("expected focus on the freeform Body textarea, got %T", result.FocusManager.Current())
	}
}

// TestResetToBlank_StillFocusesMethodURL contrasts with the sidebar
// behavior above: resetting to a blank request (see resetToBlank, ctrl+r)
// - unlike loading one picked from the sidebar - has no method/URL worth
// skipping past yet, so it must keep defaulting focus to method+URL
// itself (see defaultFocusIndex), not fall through to whatever the
// SetRequestPaneRequestMsg path would land on.
func TestResetToBlank_StillFocusesMethodURL(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.PanelFocused = true

	model, _ := pane.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	result, ok := model.(RequestPane)
	if !ok {
		t.Fatalf("expected RequestPane, got %T", model)
	}
	if _, ok := result.FocusManager.Current().(*ui.MethodSelector); !ok {
		t.Errorf("expected reset to still focus MethodSelector, got %T", result.FocusManager.Current())
	}
}
