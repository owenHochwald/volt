package requestpane

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
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
