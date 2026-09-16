package requestpane

import (
	"testing"

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
