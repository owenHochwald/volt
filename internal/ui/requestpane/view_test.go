package requestpane

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/http"
)

func TestLoadTestConfigurationFitsAndShowsEveryControl(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetWidth(100)
	pane.SetHeight(26)
	pane.toggleLoadTestMode()

	view := pane.View()
	for _, text := range []string{
		"LOAD TEST CONFIGURATION",
		"Concurrency:",
		"Total Requests:",
		"QPS (limit):",
		"Timeout:",
		"RUN LOAD TEST",
	} {
		if !strings.Contains(view, text) {
			t.Errorf("load-test view does not contain %q", text)
		}
	}
	if got := lipgloss.Height(view); got > pane.Height {
		t.Fatalf("load-test view height = %d, panel height = %d", got, pane.Height)
	}
}

// TestFieldsContent_RendersQueryParamsSection is a regression test for
// Phase 2 of "query params as structured fields": an imported request's
// QueryParams must render as their own labeled section (mirroring Path
// Params), not silently disappear or fall back to nothing.
func TestFieldsContent_RendersQueryParamsSection(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetWidth(100)
	pane.SetHeight(30)

	req := &http.Request{
		Method:           http.GET,
		URL:              "https://api.example.com/widgets",
		QueryParamSchema: map[string]http.BodyFieldSchema{"page": {Type: "integer"}},
	}
	pane.reinitRequestPane(req)

	view := pane.View()
	if !strings.Contains(view, "Query Params") {
		t.Errorf("expected the view to show a Query Params section, got:\n%s", view)
	}
	if !strings.Contains(view, "page") {
		t.Errorf("expected the view to show the \"page\" field, got:\n%s", view)
	}
}
