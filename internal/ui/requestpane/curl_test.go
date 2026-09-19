package requestpane

import (
	"errors"
	"strings"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
)

func TestCopyCurlCommandPropagatesClipboardError(t *testing.T) {
	expected := errors.New("clipboard unavailable")
	cmd := copyCurlCommand("curl ...", func(string) error {
		return expected
	})

	msg := cmd()
	result, ok := msg.(RequestCopiedMsg)
	if !ok {
		t.Fatalf("message type = %T, want RequestCopiedMsg", msg)
	}
	if !errors.Is(result.Err, expected) {
		t.Fatalf("error = %v, want %v", result.Err, expected)
	}
}

// TestRequestPane_CopyAsCurl_UsesResolvedRequest is a regression test for
// "export to curl": the copied command must reflect what would actually
// be sent right now - path/query params substituted, absolute URL - not
// the raw stored template with its {name} placeholders still in it.
func TestRequestPane_CopyAsCurl_UsesResolvedRequest(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.Request = &http.Request{
		Method:           http.GET,
		URL:              "https://api.example.com/widgets/{id}",
		QueryParamSchema: map[string]http.BodyFieldSchema{"verbose": {Type: "boolean"}},
	}
	pane.reinitRequestPane(pane.Request)
	for i := range pane.PathParams {
		pane.PathParams[i].Input.SetValue("42")
	}
	for i := range pane.QueryParams {
		pane.QueryParams[i].Enum.SetValue("true")
	}

	var copied string
	cmd := copyCurlCommand(http.BuildCurlCommand(pane.resolvedRequest()), func(s string) error {
		copied = s
		return nil
	})
	cmd()

	if !strings.Contains(copied, "'https://api.example.com/widgets/42?verbose=true'") {
		t.Errorf("expected the resolved URL in the copied command, got %q", copied)
	}
	if strings.Contains(copied, "{id}") {
		t.Errorf("expected no raw {id} placeholder in the copied command, got %q", copied)
	}
}
