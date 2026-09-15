package cli

import (
	"strings"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
)

const samplePetstoreYAML = `
openapi: 3.0.0
servers:
  - url: https://api.example.com/v1
paths:
  /pets:
    get:
      operationId: listPets
      summary: List all pets
    post:
      operationId: createPet
  /pets/{petId}:
    get:
      operationId: getPet
    delete:
      operationId: deletePet
    options:
      operationId: petOptions
  /pets/{petId}/photos:
    parameters:
      - name: petId
        in: path
    put:
      summary: Upload a pet photo
`

func TestParseOpenAPIURLs_Basic(t *testing.T) {
	requests, warnings, err := ParseOpenAPIURLs([]byte(samplePetstoreYAML), true, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(requests) != 5 {
		t.Fatalf("expected 5 importable requests, got %d", len(requests))
	}

	byKey := make(map[string]*http.Request, len(requests))
	for _, r := range requests {
		byKey[r.Method+" "+r.URL] = r
	}

	want := map[string]string{
		"GET https://api.example.com/v1/pets":                "listPets",
		"POST https://api.example.com/v1/pets":               "createPet",
		"GET https://api.example.com/v1/pets/{petId}":        "getPet",
		"DELETE https://api.example.com/v1/pets/{petId}":     "deletePet",
		"PUT https://api.example.com/v1/pets/{petId}/photos": "Upload a pet photo",
	}
	for key, wantName := range want {
		req, ok := byKey[key]
		if !ok {
			t.Errorf("missing expected request %q", key)
			continue
		}
		if req.Name != wantName {
			t.Errorf("%s: expected name %q, got %q", key, wantName, req.Name)
		}
	}

	// OPTIONS isn't a method Volt's own UI supports, so it should be
	// skipped with a warning rather than silently dropped or imported.
	if len(warnings) != 1 || !strings.Contains(warnings[0], "OPTIONS") {
		t.Errorf("expected one OPTIONS warning, got %v", warnings)
	}
}

func TestParseOpenAPIURLs_JSON(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"servers": [{"url": "https://api.example.com"}],
		"paths": {
			"/widgets": {"get": {"operationId": "listWidgets"}}
		}
	}`

	requests, _, err := ParseOpenAPIURLs([]byte(spec), false, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 1 || requests[0].URL != "https://api.example.com/widgets" {
		t.Fatalf("unexpected requests: %+v", requests)
	}
}

func TestParseOpenAPIURLs_NoAbsoluteServerRequiresOverride(t *testing.T) {
	spec := `{"openapi":"3.0.0","servers":[{"url":"/v1"}],"paths":{"/x":{"get":{}}}}`

	if _, _, err := ParseOpenAPIURLs([]byte(spec), false, "", false); err == nil {
		t.Fatal("expected an error when no absolute server URL is available")
	}

	requests, _, err := ParseOpenAPIURLs([]byte(spec), false, "http://localhost:4000", false)
	if err != nil {
		t.Fatalf("unexpected error with base-url override: %v", err)
	}
	if len(requests) != 1 || requests[0].URL != "http://localhost:4000/x" {
		t.Fatalf("unexpected requests: %+v", requests)
	}
}

func TestParseOpenAPIURLs_AllowRelativeStoresBarePaths(t *testing.T) {
	// No servers block at all - previously this always required -base-url;
	// with allowRelative it should succeed and store bare paths instead.
	spec := `{"openapi":"3.0.0","paths":{"/widgets":{"get":{"operationId":"listWidgets"}}}}`

	requests, _, err := ParseOpenAPIURLs([]byte(spec), false, "", true)
	if err != nil {
		t.Fatalf("unexpected error with allowRelative: %v", err)
	}
	if len(requests) != 1 || requests[0].URL != "/widgets" {
		t.Fatalf("expected a bare-path request, got %+v", requests)
	}
	if requests[0].IsAbsoluteURL() {
		t.Errorf("expected a relative URL, got absolute: %s", requests[0].URL)
	}
}

func TestSpecServerURL(t *testing.T) {
	spec := `{"openapi":"3.0.0","servers":[{"url":"https://api.example.com/v2"}],"paths":{}}`

	url, err := SpecServerURL([]byte(spec), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://api.example.com/v2" {
		t.Errorf("expected https://api.example.com/v2, got %q", url)
	}
}

func TestParseOpenAPIURLs_RejectsSwagger2(t *testing.T) {
	spec := `{"swagger":"2.0","host":"api.example.com","paths":{"/x":{"get":{}}}}`

	_, _, err := ParseOpenAPIURLs([]byte(spec), false, "", false)
	if err == nil || !strings.Contains(err.Error(), "Swagger 2.0") {
		t.Fatalf("expected a Swagger 2.0 rejection error, got %v", err)
	}
}
