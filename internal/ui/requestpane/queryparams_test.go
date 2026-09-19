package requestpane

import (
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
)

func TestRequestPane_SyncQueryParams_ExtractsFromSchema(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{
		Method:           http.GET,
		URL:              "https://api.example.com/widgets",
		QueryParamSchema: map[string]http.BodyFieldSchema{"page": {Type: "integer"}, "limit": {Type: "integer"}},
	}
	pane.reinitRequestPane(req)

	if len(pane.QueryParams) != 2 {
		t.Fatalf("expected 2 query params, got %+v", pane.QueryParams)
	}
	if pane.QueryParams[0].Name != "limit" || pane.QueryParams[1].Name != "page" {
		t.Errorf("expected sorted [limit page], got %+v", pane.QueryParams)
	}
}

func TestRequestPane_SyncQueryParams_CarriesTypeAndDefaultThrough(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{
		Method: http.GET,
		URL:    "https://api.example.com/widgets",
		QueryParamSchema: map[string]http.BodyFieldSchema{
			"sort": {Type: "string", Enum: []string{"asc", "desc"}, Default: "asc"},
			"page": {Type: "integer", Default: "1"},
		},
	}
	pane.reinitRequestPane(req)

	byName := make(map[string]Param, len(pane.QueryParams))
	for _, p := range pane.QueryParams {
		byName[p.Name] = p
	}

	if byName["sort"].Enum == nil {
		t.Fatal("expected sort to get an EnumSelector")
	}
	if got := byName["sort"].Value(); got != "asc" {
		t.Errorf("expected sort to default to %q, got %q", "asc", got)
	}
	if got := byName["page"].Value(); got != "1" {
		t.Errorf("expected page to default to %q, got %q", "1", got)
	}
}

// TestRequestPane_SyncQueryParams_HandTypedRequestGetsNone verifies a
// request with no QueryParamSchema at all (a hand-typed request, or one
// imported before this existed) gets no QueryParams - it keeps typing
// "?key=value" straight into the URL, same as before this existed.
func TestRequestPane_SyncQueryParams_HandTypedRequestGetsNone(t *testing.T) {
	pane, _ := setupTestPane(t)
	req := &http.Request{Method: http.GET, URL: "https://api.example.com/widgets?q=1"}
	pane.reinitRequestPane(req)

	if len(pane.QueryParams) != 0 {
		t.Errorf("expected no query params for a hand-typed request, got %+v", pane.QueryParams)
	}
}

// TestRequestPane_ResolvedRequest_BuildsQueryStringFromParams verifies
// values typed into query param fields end up appended to the URL
// actually sent, while the stored template's URL is left untouched.
func TestRequestPane_ResolvedRequest_BuildsQueryStringFromParams(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{
		Method:           http.GET,
		URL:              "https://api.example.com/widgets",
		QueryParamSchema: map[string]http.BodyFieldSchema{"page": {Type: "integer"}, "limit": {Type: "integer"}},
	}
	pane.reinitRequestPane(req)
	for i := range pane.QueryParams {
		switch pane.QueryParams[i].Name {
		case "page":
			pane.QueryParams[i].Input.SetValue("2")
		case "limit":
			pane.QueryParams[i].Input.SetValue("10")
		}
	}

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/widgets?limit=10&page=2" {
		t.Errorf("expected the resolved URL to include the query string, got %q", resolved.URL)
	}
	if pane.Request.URL != "https://api.example.com/widgets" {
		t.Errorf("expected the stored request's URL to stay bare, got %q", pane.Request.URL)
	}
}

// TestRequestPane_ResolvedRequest_OmitsEmptyQueryParams verifies a query
// param never typed into is left out of the query string entirely,
// rather than sent as an empty "key=".
func TestRequestPane_ResolvedRequest_OmitsEmptyQueryParams(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{
		Method:           http.GET,
		URL:              "https://api.example.com/widgets",
		QueryParamSchema: map[string]http.BodyFieldSchema{"page": {Type: "integer"}, "q": {Type: "string"}},
	}
	pane.reinitRequestPane(req)
	for i := range pane.QueryParams {
		if pane.QueryParams[i].Name == "q" {
			pane.QueryParams[i].Input.SetValue("widgets")
		}
	}

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/widgets?q=widgets" {
		t.Errorf("expected only the filled-in param in the query string, got %q", resolved.URL)
	}
}

// TestRequestPane_ResolvedRequest_NoQueryParamsFilled_URLUnchanged
// verifies a schema-imported request with no query params actually typed
// in yet resolves to the same URL, same as before this existed.
func TestRequestPane_ResolvedRequest_NoQueryParamsFilled_URLUnchanged(t *testing.T) {
	pane, _ := setupTestPane(t)

	req := &http.Request{
		Method:           http.GET,
		URL:              "https://api.example.com/widgets",
		QueryParamSchema: map[string]http.BodyFieldSchema{"page": {Type: "integer"}},
	}
	pane.reinitRequestPane(req)

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/widgets" {
		t.Errorf("expected the URL to stay bare with nothing typed in, got %q", resolved.URL)
	}
}

func TestBuildQueryString_URLEncodesValues(t *testing.T) {
	params := []Param{paramWithValue(t, "hello world", Param{Name: "q"})}
	got := BuildQueryString(params)
	if got != "?q=hello+world" {
		t.Errorf("BuildQueryString() = %q, want %q", got, "?q=hello+world")
	}
}

func TestBuildQueryString_NoValues_ReturnsEmpty(t *testing.T) {
	params := []Param{paramWithValue(t, "", Param{Name: "page"})}
	if got := BuildQueryString(params); got != "" {
		t.Errorf("BuildQueryString() = %q, want empty string", got)
	}
}

func TestAppendQueryString_HandlesExistingQueryString(t *testing.T) {
	got := appendQueryString("https://api.example.com/x?already=1", "?page=2")
	if got != "https://api.example.com/x?already=1&page=2" {
		t.Errorf("appendQueryString() = %q", got)
	}
}
