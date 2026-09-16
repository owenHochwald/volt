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

// requestBodySchema decodes a request's Body as the {"__volt_body_schema__":
// {...}} wrapper `volt import` writes for a JSON request body schema, and
// fails the test if it isn't that shape.
func requestBodySchema(t *testing.T, req *http.Request) map[string]http.BodyFieldSchema {
	t.Helper()
	if req.Body == "" {
		return nil
	}
	var wrapper map[string]map[string]http.BodyFieldSchema
	if err := json.Unmarshal([]byte(req.Body), &wrapper); err != nil {
		t.Fatalf("request body %q isn't the body-schema wrapper shape: %v", req.Body, err)
	}
	return wrapper[http.BodySchemaKey]
}

func requestBodyKeys(t *testing.T, req *http.Request) []string {
	t.Helper()
	schema := requestBodySchema(t, req)
	keys := make([]string, 0, len(schema))
	for k := range schema {
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

	schema := requestBodySchema(t, requests[0])
	if schema["name"].Type != "string" || schema["color"].Type != "string" {
		t.Errorf("expected each field's declared type to carry through to the schema, got %+v", schema)
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

func TestParseOpenAPIURLs_RequestBodySchemaDefaultAndEnum(t *testing.T) {
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
										"status": {"type": "string", "enum": ["active", "inactive"], "default": "active"},
										"count": {"type": "integer", "default": 1},
										"name": {"type": "string"}
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

	schema := requestBodySchema(t, requests[0])

	status := schema["status"]
	if status.Default != "active" {
		t.Errorf("expected status default %q, got %q", "active", status.Default)
	}
	if len(status.Enum) != 2 || status.Enum[0] != "active" || status.Enum[1] != "inactive" {
		t.Errorf("expected status enum [active inactive], got %v", status.Enum)
	}

	count := schema["count"]
	if count.Default != "1" {
		t.Errorf("expected count's numeric default to stringify to %q, got %q", "1", count.Default)
	}

	name := schema["name"]
	if name.Default != "" || len(name.Enum) != 0 {
		t.Errorf("expected name to have no default/enum, got %+v", name)
	}
}

func TestParseOpenAPIURLs_RequestBodySchemaLimits(t *testing.T) {
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
										"name": {"type": "string", "minLength": 2, "maxLength": 50},
										"quantity": {"type": "integer", "minimum": 1, "maximum": 100},
										"note": {"type": "string"}
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

	schema := requestBodySchema(t, requests[0])

	name := schema["name"]
	if name.MinLength == nil || *name.MinLength != 2 {
		t.Errorf("expected name.MinLength=2, got %v", name.MinLength)
	}
	if name.MaxLength == nil || *name.MaxLength != 50 {
		t.Errorf("expected name.MaxLength=50, got %v", name.MaxLength)
	}

	quantity := schema["quantity"]
	if quantity.Minimum == nil || *quantity.Minimum != 1 {
		t.Errorf("expected quantity.Minimum=1, got %v", quantity.Minimum)
	}
	if quantity.Maximum == nil || *quantity.Maximum != 100 {
		t.Errorf("expected quantity.Maximum=100, got %v", quantity.Maximum)
	}

	note := schema["note"]
	if note.MinLength != nil || note.MaxLength != nil || note.Minimum != nil || note.Maximum != nil {
		t.Errorf("expected note to have no constraints, got %+v", note)
	}
}

// TestParseOpenAPIURLs_ArrayOfRefObject verifies a "type: array" property
// whose "items" is a $ref to an object schema resolves that object's own
// properties into Items, with ItemsRefName naming the referenced schema -
// so the array can be composed as a repeatable set of sub-fields in the
// TUI instead of being left as opaque text.
func TestParseOpenAPIURLs_ArrayOfRefObject(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"servers": [{"url": "https://api.example.com"}],
		"components": {
			"schemas": {
				"Tag": {
					"type": "object",
					"properties": {
						"key": {"type": "string"},
						"value": {"type": "string"}
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
								"schema": {
									"type": "object",
									"properties": {
										"name": {"type": "string"},
										"tags": {
											"type": "array",
											"items": {"$ref": "#/components/schemas/Tag"}
										}
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

	schema := requestBodySchema(t, requests[0])

	tags, ok := schema["tags"]
	if !ok {
		t.Fatalf("expected a 'tags' field, got %+v", schema)
	}
	if tags.Type != "array" {
		t.Errorf("expected tags.Type=array, got %q", tags.Type)
	}
	if tags.ItemsRefName != "Tag" {
		t.Errorf("expected tags.ItemsRefName=Tag, got %q", tags.ItemsRefName)
	}
	if len(tags.Items) != 2 {
		t.Fatalf("expected 2 item fields (key, value), got %+v", tags.Items)
	}
	if tags.Items["key"].Type != "string" {
		t.Errorf("expected tags.Items[key].Type=string, got %+v", tags.Items["key"])
	}
	if tags.Items["value"].Type != "string" {
		t.Errorf("expected tags.Items[value].Type=string, got %+v", tags.Items["value"])
	}

	// A plain scalar field alongside the array must be unaffected.
	if schema["name"].Type != "string" || schema["name"].Items != nil {
		t.Errorf("expected name to be a plain string with no Items, got %+v", schema["name"])
	}
}

// TestParseOpenAPIURLs_ArrayOfSimpleItems verifies a "type: array" property
// whose items are NOT an object (e.g. items: {type: string}) leaves
// Items/ItemsRefName unset - there's no per-property structure to compose,
// so it stays a "simple" array.
func TestParseOpenAPIURLs_ArrayOfSimpleItems(t *testing.T) {
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
										"labels": {
											"type": "array",
											"items": {"type": "string"}
										}
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
	schema := requestBodySchema(t, requests[0])

	labels := schema["labels"]
	if labels.Type != "array" {
		t.Errorf("expected labels.Type=array, got %q", labels.Type)
	}
	if labels.Items != nil {
		t.Errorf("expected a simple array's Items to stay unset, got %+v", labels.Items)
	}
	if labels.ItemsRefName != "" {
		t.Errorf("expected a simple array's ItemsRefName to stay unset, got %q", labels.ItemsRefName)
	}
	if labels.ItemType != "string" {
		t.Errorf("expected a simple array's own item type to be kept as ItemType=%q, got %q", "string", labels.ItemType)
	}
}

// TestParseOpenAPIURLs_RequiredFields verifies a schema's top-level
// "required" list marks the matching fields Required, and that an
// array-of-objects field's *item* schema's own "required" list is read
// separately for its sub-fields.
func TestParseOpenAPIURLs_RequiredFields(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"servers": [{"url": "https://api.example.com"}],
		"components": {
			"schemas": {
				"Tag": {
					"type": "object",
					"required": ["key"],
					"properties": {
						"key": {"type": "string"},
						"value": {"type": "string"}
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
								"schema": {
									"type": "object",
									"required": ["name"],
									"properties": {
										"name": {"type": "string"},
										"note": {"type": "string"},
										"tags": {
											"type": "array",
											"items": {"$ref": "#/components/schemas/Tag"}
										}
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
	schema := requestBodySchema(t, requests[0])

	if !schema["name"].Required {
		t.Error("expected 'name' to be required")
	}
	if schema["note"].Required {
		t.Error("expected 'note' to not be required")
	}
	tags := schema["tags"]
	if tags.Required {
		t.Error("expected 'tags' itself to not be required (not in the top-level required list)")
	}
	if !tags.Items["key"].Required {
		t.Error("expected the array item's 'key' field to be required (per Tag's own required list)")
	}
	if tags.Items["value"].Required {
		t.Error("expected the array item's 'value' field to not be required")
	}
}

// TestParseOpenAPIURLs_PathParamSchema verifies "in: path" parameters are
// captured into http.Request.PathParamSchema with their declared
// type/constraints/enum, and that an operation-level parameter overrides a
// path-item-level one of the same name (per the OpenAPI spec) rather than
// both existing side by side.
func TestParseOpenAPIURLs_PathParamSchema(t *testing.T) {
	spec := `{
		"openapi": "3.0.0",
		"servers": [{"url": "https://api.example.com"}],
		"paths": {
			"/hosts/{host_id}": {
				"parameters": [
					{"name": "host_id", "in": "path", "required": true, "schema": {"type": "integer", "minimum": 1}}
				],
				"get": {
					"operationId": "getHost"
				},
				"delete": {
					"operationId": "deleteHost",
					"parameters": [
						{"name": "host_id", "in": "path", "required": true, "schema": {"type": "integer", "minimum": 100}}
					]
				}
			},
			"/regions/{region}": {
				"get": {
					"operationId": "getRegion",
					"parameters": [
						{"name": "region", "in": "path", "required": true, "schema": {"type": "string", "enum": ["us", "eu"]}},
						{"name": "verbose", "in": "query", "schema": {"type": "boolean"}}
					]
				}
			}
		}
	}`

	requests, _, err := ParseOpenAPIURLs([]byte(spec), false, "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	byName := make(map[string]*http.Request, len(requests))
	for _, r := range requests {
		byName[r.Name] = r
	}

	getHost := byName["getHost"]
	if getHost == nil {
		t.Fatal("expected a getHost request")
	}
	hostSchema := getHost.PathParamSchema["host_id"]
	if hostSchema.Type != "integer" {
		t.Errorf("expected host_id.Type=integer (inherited from path-item level), got %q", hostSchema.Type)
	}
	if !hostSchema.Required {
		t.Error("expected host_id to be Required")
	}
	if hostSchema.Minimum == nil || *hostSchema.Minimum != 1 {
		t.Errorf("expected host_id.Minimum=1 (path-item level), got %v", hostSchema.Minimum)
	}

	deleteHost := byName["deleteHost"]
	if deleteHost == nil {
		t.Fatal("expected a deleteHost request")
	}
	deleteHostSchema := deleteHost.PathParamSchema["host_id"]
	if deleteHostSchema.Minimum == nil || *deleteHostSchema.Minimum != 100 {
		t.Errorf("expected the operation-level host_id (minimum=100) to override the path-item-level one (minimum=1), got %v", deleteHostSchema.Minimum)
	}

	getRegion := byName["getRegion"]
	if getRegion == nil {
		t.Fatal("expected a getRegion request")
	}
	regionSchema := getRegion.PathParamSchema["region"]
	if len(regionSchema.Enum) != 2 || regionSchema.Enum[0] != "us" || regionSchema.Enum[1] != "eu" {
		t.Errorf("expected region.Enum=[us eu], got %v", regionSchema.Enum)
	}
	if _, ok := getRegion.PathParamSchema["verbose"]; ok {
		t.Error("expected the 'in: query' parameter to be excluded from PathParamSchema")
	}
}
