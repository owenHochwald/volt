package requestpane

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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

// TestFieldsContent_ShowsRequestDescriptionUnderName verifies an
// operation's OpenAPI description (see http.Request.Description) renders
// as a read-only line right under Name, and that a request with none
// (the ordinary hand-typed case) doesn't show a stray blank line for it.
func TestFieldsContent_ShowsRequestDescriptionUnderName(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetWidth(100)
	pane.SetHeight(30)

	req := &http.Request{
		Method:      http.GET,
		URL:         "https://api.example.com/widgets",
		Description: "Lists every widget in the catalog.",
	}
	pane.reinitRequestPane(req)

	view := pane.View()
	if !strings.Contains(view, "Lists every widget in the catalog.") {
		t.Errorf("expected the request's description to render under Name, got:\n%s", view)
	}

	noDescReq := &http.Request{Method: http.GET, URL: "https://api.example.com/other"}
	pane.reinitRequestPane(noDescReq)
	if strings.Contains(pane.View(), "Lists every widget") {
		t.Error("expected the previous request's description not to leak into one with none")
	}
}

// TestFieldsContent_EnumFieldShowsDescriptionToTheRight verifies a body
// param backed by an EnumSelector (or a boolean picker) shows its schema
// description to the right of the field itself, since a picker has no
// placeholder-when-empty slot the way a plain text field does.
func TestFieldsContent_EnumFieldShowsDescriptionToTheRight(t *testing.T) {
	pane := newTestRequestPane(t)
	pane.SetWidth(100)
	pane.SetHeight(30)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"status": {Type: "string", Enum: []string{"active", "inactive"}, Description: "Whether the widget is active."},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	view := pane.View()
	if !strings.Contains(view, "Whether the widget is active.") {
		t.Errorf("expected the enum field's description to render, got:\n%s", view)
	}
}

// TestFieldsContent_RequiredEnumError_TakesPriorityOverDescription
// verifies a blocked send's inline "! required" error still shows for an
// enum field that also has a description - the error is more actionable
// than reference text, so it wins the shared comment slot rather than
// showing both or silently dropping the error.
func TestFieldsContent_RequiredEnumError_TakesPriorityOverDescription(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.SetWidth(100)
	pane.SetHeight(40)

	body := schemaBody(t, map[string]http.BodyFieldSchema{
		"status": {Type: "string", Enum: []string{"active", "inactive"}, Description: "Whether the widget is active.", Required: true},
	})
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/widgets", Body: body, Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	result, _ := pane.currentMode.HandleInput(&pane, tea.KeyPressMsg(tea.Key{Code: 'p', Mod: tea.ModCtrl}))

	view := result.View()
	if !strings.Contains(view, "! required") {
		t.Errorf("expected the required error to show, got:\n%s", view)
	}
	if strings.Contains(view, "Whether the widget is active.") {
		t.Errorf("expected the error to take priority over the description on the same line, got:\n%s", view)
	}
}
