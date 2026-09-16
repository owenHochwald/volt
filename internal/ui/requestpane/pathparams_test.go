package requestpane

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
)

func TestExtractPathParamNames(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want []string
	}{
		{"no params", "https://api.example.com/pets", nil},
		{"one param", "https://api.example.com/cluster/{cluster_id}/repository", []string{"cluster_id"}},
		{"multiple params in order", "/orgs/{org_id}/repos/{repo_id}", []string{"org_id", "repo_id"}},
		{"duplicate param counted once", "/a/{id}/b/{id}", []string{"id"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathParamNames(tt.url)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSubstitutePathParams(t *testing.T) {
	input := textinput.New()
	input.SetValue("42")
	params := []Param{
		{Name: "cluster_id", Input: input},
	}
	got := SubstitutePathParams("/cluster/{cluster_id}/repository", params)
	want := "/cluster/42/repository"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestRequestPane_SyncPathParams_ExtractsFromURL verifies that loading a
// request whose URL has a {cluster_id} placeholder (as an OpenAPI-imported
// request's does) surfaces it as its own editable field.
func TestRequestPane_SyncPathParams_ExtractsFromURL(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{Method: http.GET, URL: "https://api.example.com/cluster/{cluster_id}/repository", Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if len(pane.PathParams) != 1 || pane.PathParams[0].Name != "cluster_id" {
		t.Fatalf("expected one path param %q, got %+v", "cluster_id", pane.PathParams)
	}
}

// TestRequestPane_ResolvedRequest_SubstitutesPathParams verifies the value
// typed into a path param field actually ends up in the URL used to send
// the request, while the stored template (m.Request.URL) is left alone so
// the field keeps working if the request is sent again with a new value.
func TestRequestPane_ResolvedRequest_SubstitutesPathParams(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{Method: http.GET, URL: "https://api.example.com/cluster/{cluster_id}/repository", Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	if len(pane.PathParams) != 1 {
		t.Fatalf("expected one path param, got %+v", pane.PathParams)
	}
	pane.PathParams[0].Input.SetValue("42")

	resolved := pane.resolvedRequest()
	want := "https://api.example.com/cluster/42/repository"
	if resolved.URL != want {
		t.Errorf("expected resolved URL %q, got %q", want, resolved.URL)
	}
	if pane.Request.URL != req.URL {
		t.Errorf("expected the stored request to keep the {cluster_id} template, got %q", pane.Request.URL)
	}
}

// TestRequestPane_SyncPathParams_ReloadDoesNotLeakValues guards against a
// value typed for one request's {id} bleeding into a different request
// that happens to have a same-named param.
func TestRequestPane_SyncPathParams_ReloadDoesNotLeakValues(t *testing.T) {
	pane, _ := setupTestPane(t)

	first := &http.Request{Method: http.GET, URL: "https://api.example.com/widgets/{id}", Headers: map[string]string{}}
	pane.reinitRequestPane(first)
	pane.PathParams[0].Input.SetValue("first-value")

	second := &http.Request{Method: http.GET, URL: "https://api.example.com/gadgets/{id}", Headers: map[string]string{}}
	pane.reinitRequestPane(second)

	if len(pane.PathParams) != 1 {
		t.Fatalf("expected one path param, got %+v", pane.PathParams)
	}
	if got := pane.PathParams[0].Input.Value(); got != "" {
		t.Errorf("expected a blank value after loading a different request, got %q", got)
	}
}

// TestRequestPane_PathParamSchema_CarriesConstraintsThrough is a
// regression test for a real bug report: a path param's OpenAPI
// constraints (type, minimum/maximum, required, enum) were entirely
// discarded on import - every {name} placeholder was edited as
// unconstrained free text, unlike a body field with the same schema
// shape. It should behave exactly like a body field: typed hint, live
// invalid-value highlighting, blocked send on a required-but-missing
// value.
func TestRequestPane_PathParamSchema_CarriesConstraintsThrough(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{
		Method:  http.GET,
		URL:     "https://api.example.com/hosts/{host_id}/region/{region}",
		Headers: map[string]string{},
		PathParamSchema: map[string]http.BodyFieldSchema{
			"host_id": {Type: "integer", Required: true, Minimum: floatPtr(1)},
			"region":  {Type: "string", Enum: []string{"us", "eu"}, Default: "us"},
		},
	}
	pane.reinitRequestPane(req)

	if len(pane.PathParams) != 2 {
		t.Fatalf("expected 2 path params, got %+v", pane.PathParams)
	}
	byName := make(map[string]Param, len(pane.PathParams))
	for _, p := range pane.PathParams {
		byName[p.Name] = p
	}

	hostID := byName["host_id"]
	if hostID.Type != "integer" {
		t.Errorf("expected host_id.Type=integer, got %q", hostID.Type)
	}
	if !hostID.Required {
		t.Error("expected host_id to be Required")
	}
	if hostID.Minimum == nil || *hostID.Minimum != 1 {
		t.Errorf("expected host_id.Minimum=1, got %v", hostID.Minimum)
	}
	if hostID.Enum != nil {
		t.Error("expected host_id to be a plain Input, not an Enum picker")
	}

	region := byName["region"]
	if region.Enum == nil {
		t.Fatal("expected region to get an Enum picker")
	}
	if got := region.Value(); got != "us" {
		t.Errorf("expected region to start on its schema default %q, got %q", "us", got)
	}

	// Live validation: an out-of-range integer must be flagged, same as a
	// body field with the same constraint.
	for i := range pane.PathParams {
		if pane.PathParams[i].Name == "host_id" {
			pane.PathParams[i].Input.SetValue("0")
		}
	}
	hostIDField, _ := findPathParam(pane.PathParams, "host_id")
	if _, ok := ValidateParam(hostIDField); ok {
		t.Error("expected host_id=0 to violate minimum=1")
	}
}

// TestRequestPane_HandleSubmit_BlocksSendOnMissingRequiredPathParam is an
// end-to-end check mirroring the equivalent BodyParams test: a required
// path param left empty prevents sending and jumps focus to it.
func TestRequestPane_HandleSubmit_BlocksSendOnMissingRequiredPathParam(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{
		Method:  http.GET,
		URL:     "https://api.example.com/hosts/{host_id}",
		Headers: map[string]string{},
		PathParamSchema: map[string]http.BodyFieldSchema{
			"host_id": {Type: "integer", Required: true},
		},
	}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true
	// host_id left empty on purpose.

	model, _ := pane.currentMode.HandleInput(&pane, tea.KeyMsg{Type: tea.KeyCtrlP})
	result, ok := model.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model)
	}
	if result.RequestInProgress {
		t.Error("expected the missing required path param to block sending")
	}
	if len(result.ParseErrors) == 0 {
		t.Error("expected a ParseErrors entry explaining why the send was blocked")
	}

	current, ok := result.FocusManager.Current().(*textinput.Model)
	if !ok {
		t.Fatalf("expected focus to land on a text input, got %T", result.FocusManager.Current())
	}
	if current != &result.PathParams[0].Input {
		t.Error("expected focus to jump to the missing required path param")
	}
}

// findPathParam is a small test helper for looking up a Param by name.
func findPathParam(params []Param, name string) (Param, bool) {
	for _, p := range params {
		if p.Name == name {
			return p, true
		}
	}
	return Param{}, false
}
