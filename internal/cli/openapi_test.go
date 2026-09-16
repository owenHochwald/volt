package cli

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

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

func requestBodyKeys(t *testing.T, req *http.Request) []string {
	t.Helper()
	if req.Body == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(req.Body), &m); err != nil {
		t.Fatalf("request body %q is not a flat JSON object: %v", req.Body, err)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestParseOpenAPIURLs_InlineRequestBodySchema(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"servers": [{"url": "https://api.example.com"}],
		"paths": {
			"/widgets": {
				"post": {
					"operationId": "createWidget",
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"properties": {
										"name": {"type": "string"},
										"color": {"type": "string"}
									}
								}
							}
						}
					}
				}
			}
		}
	}`

	requests, _, err := ParseOpenAPIURLs([]byte(spec), false, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	keys := requestBodyKeys(t, requests[0])
	want := []string{"color", "name"}
	if len(keys) != len(want) || keys[0] != want[0] || keys[1] != want[1] {
		t.Errorf("expected body template keys %v, got %v (body=%q)", want, keys, requests[0].Body)
	}

	var body map[string]string
	if err := json.Unmarshal([]byte(requests[0].Body), &body); err != nil {
		t.Fatalf("body isn't valid JSON: %v", err)
	}
	if body["name"] != "<string>" || body["color"] != "<string>" {
		t.Errorf("expected each field's declared type to be encoded as a <type> marker, got %v", body)
	}
}

func TestParseOpenAPIURLs_RequestBodySchemaViaRef(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"servers": [{"url": "https://api.example.com"}],
		"components": {
			"schemas": {
				"Widget": {
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"quantity": {"type": "integer"}
					}
				}
			}
		},
		"paths": {
			"/widgets": {
				"post": {
					"operationId": "createWidget",
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {"$ref": "#/components/schemas/Widget"}
							}
						}
					}
				}
			}
		}
	}`

	requests, _, err := ParseOpenAPIURLs([]byte(spec), false, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	keys := requestBodyKeys(t, requests[0])
	want := []string{"name", "quantity"}
	if len(keys) != len(want) || keys[0] != want[0] || keys[1] != want[1] {
		t.Errorf("expected body template keys %v (resolved via $ref), got %v (body=%q)", want, keys, requests[0].Body)
	}
}

func TestParseOpenAPIURLs_NoRequestBodySchemaLeavesBodyEmpty(t *testing.T) {
	// createPet in samplePetstoreYAML has no requestBody at all.
	requests, _, err := ParseOpenAPIURLs([]byte(samplePetstoreYAML), true, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range requests {
		if r.Name == "createPet" && r.Body != "" {
			t.Errorf("expected createPet's body to stay empty with no schema, got %q", r.Body)
		}
	}
}

// schemasDoc builds a minimal doc with the given components.schemas map,
// for testing resolveSchemaRef directly without going through JSON/YAML.
func schemasDoc(schemas map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"components": map[string]interface{}{
			"schemas": schemas,
		},
	}
}

func TestResolveSchemaRef_FollowsAChainOfRefs(t *testing.T) {
	doc := schemasDoc(map[string]interface{}{
		"A": map[string]interface{}{"$ref": "#/components/schemas/B"},
		"B": map[string]interface{}{"$ref": "#/components/schemas/C"},
		"C": map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}},
		},
	})

	start := map[string]interface{}{"$ref": "#/components/schemas/A"}
	resolved := resolveSchemaRef(doc, start)

	props, _ := resolved["properties"].(map[string]interface{})
	if _, ok := props["name"]; !ok {
		t.Fatalf("expected the 2-hop chain A->B->C to resolve to C's properties, got %+v", resolved)
	}
}

func TestResolveSchemaRef_CircularChainStopsInsteadOfHanging(t *testing.T) {
	doc := schemasDoc(map[string]interface{}{
		"A": map[string]interface{}{"$ref": "#/components/schemas/B"},
		"B": map[string]interface{}{"$ref": "#/components/schemas/A"},
	})

	start := map[string]interface{}{"$ref": "#/components/schemas/A"}

	done := make(chan map[string]interface{}, 1)
	go func() { done <- resolveSchemaRef(doc, start) }()

	select {
	case resolved := <-done:
		// Never reaches real properties - that's fine, just must not hang
		// or panic. jsonRequestBodyFields on top of this yields no fields.
		if _, stillRef := resolved["$ref"]; !stillRef {
			t.Errorf("expected to still be on an unresolved $ref after hitting the depth cap, got %+v", resolved)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("resolveSchemaRef hung on a circular $ref chain")
	}
}

func TestParseOpenAPIURLs_RequestBodySchemaViaChainedRef(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"servers": [{"url": "https://api.example.com"}],
		"components": {
			"schemas": {
				"CreateWidgetRequest": {"$ref": "#/components/schemas/WidgetBase"},
				"WidgetBase": {
					"type": "object",
					"properties": {"name": {"type": "string"}}
				}
			}
		},
		"paths": {
			"/widgets": {
				"post": {
					"operationId": "createWidget",
					"requestBody": {
						"content": {
							"application/json": {
								"schema": {"$ref": "#/components/schemas/CreateWidgetRequest"}
							}
						}
					}
				}
			}
		}
	}`

	requests, _, err := ParseOpenAPIURLs([]byte(spec), false, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	keys := requestBodyKeys(t, requests[0])
	if len(keys) != 1 || keys[0] != "name" {
		t.Errorf("expected body template key [name] resolved via a 2-hop $ref chain, got %v (body=%q)", keys, requests[0].Body)
	}
}
