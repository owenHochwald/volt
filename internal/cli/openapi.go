package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/owenHochwald/Volt/internal/http"
	"gopkg.in/yaml.v3"
)

// importableMethods maps lowercase OpenAPI operation verbs to the HTTP
// methods Volt's request model actually supports.
var importableMethods = map[string]string{
	"get":    http.GET,
	"post":   http.POST,
	"put":    http.PUT,
	"patch":  http.PATCH,
	"delete": http.DELETE,
}

// allHTTPVerbs is every verb OpenAPI allows as a path-item key, used to
// distinguish operations from unrelated path-item fields (parameters, etc.).
var allHTTPVerbs = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true,
	"options": true, "head": true, "patch": true, "trace": true,
}

// ParseOpenAPIURLs extracts one Request stub per operation from an OpenAPI
// 3.x document. baseURLOverride, if non-empty, replaces whatever "servers"
// entry the spec declares (useful when the spec's server doesn't match
// where you're actually testing against).
//
// When allowRelative is true (importing into a request group), request
// URLs are stored as bare paths (e.g. "/pets") rather than being prefixed
// with a base URL - they're meant to be resolved against the group's
// configured server at send time, so the same import works even when the
// spec declares no server at all. When false, a base URL is required and
// every request URL is absolute, matching a plain ungrouped import.
func ParseOpenAPIURLs(data []byte, isYAML bool, baseURLOverride string, allowRelative bool) ([]*http.Request, []string, error) {
	doc, err := decodeSpec(data, isYAML)
	if err != nil {
		return nil, nil, err
	}

	baseURL := strings.TrimRight(baseURLOverride, "/")
	if baseURL == "" {
		baseURL = firstServerURL(doc)
	}
	if baseURL == "" && !allowRelative {
		return nil, nil, fmt.Errorf("spec has no absolute server URL; pass -base-url http://host to specify one")
	}

	rawPaths, _ := doc["paths"].(map[string]interface{})
	if len(rawPaths) == 0 {
		return nil, nil, fmt.Errorf(`no "paths" found in spec`)
	}

	paths := make([]string, 0, len(rawPaths))
	for p := range rawPaths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var requests []*http.Request
	var warnings []string

	for _, path := range paths {
		item, ok := rawPaths[path].(map[string]interface{})
		if !ok {
			continue
		}

		verbs := make([]string, 0, len(item))
		for v := range item {
			verbs = append(verbs, v)
		}
		sort.Strings(verbs)

		for _, verb := range verbs {
			lowerVerb := strings.ToLower(verb)
			method, supported := importableMethods[lowerVerb]
			if !supported {
				if allHTTPVerbs[lowerVerb] {
					warnings = append(warnings, fmt.Sprintf("skipped %s %s: method not supported by Volt", strings.ToUpper(verb), path))
				}
				continue
			}

			op, _ := item[verb].(map[string]interface{})
			url := path
			if !allowRelative {
				url = baseURL + path
			}
			req := http.NewRequestWithParams(method, url)
			req.Name = truncateName(operationName(op, method, path))
			if fields := jsonRequestBodyFields(doc, op); len(fields) > 0 {
				req.Body = requestBodyTemplate(fields)
			}

			if err := req.Validate(); err != nil {
				warnings = append(warnings, fmt.Sprintf("skipped %s %s: %v", method, req.URL, err))
				continue
			}
			requests = append(requests, req)
		}
	}

	return requests, warnings, nil
}

// SpecServerURL returns the first absolute server URL declared in the
// spec, or "" if the spec declares none.
func SpecServerURL(data []byte, isYAML bool) (string, error) {
	doc, err := decodeSpec(data, isYAML)
	if err != nil {
		return "", err
	}
	return firstServerURL(doc), nil
}

func decodeSpec(data []byte, isYAML bool) (map[string]interface{}, error) {
	var doc map[string]interface{}
	var err error
	if isYAML {
		err = yaml.Unmarshal(data, &doc)
	} else {
		err = json.Unmarshal(data, &doc)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}
	if len(doc) == 0 {
		return nil, fmt.Errorf("spec is empty")
	}
	if _, hasOpenAPI := doc["openapi"]; !hasOpenAPI {
		if _, hasSwagger := doc["swagger"]; hasSwagger {
			return nil, fmt.Errorf("Swagger 2.0 specs are not supported, only OpenAPI 3.x")
		}
	}
	return doc, nil
}

func firstServerURL(doc map[string]interface{}) string {
	servers, _ := doc["servers"].([]interface{})
	for _, s := range servers {
		entry, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		url, _ := entry["url"].(string)
		if strings.HasPrefix(url, "http") {
			return strings.TrimRight(url, "/")
		}
	}
	return ""
}

// bodyField is one top-level property of a request body's JSON schema.
type bodyField struct {
	Name string
	// Type is the schema's declared "type" (string, integer, number,
	// boolean, array, object), or "" if the schema doesn't say. Every
	// field is still edited as plain text in the TUI regardless of Type -
	// it's shown only as a hint - so an array/object type isn't resolved
	// any further.
	Type string
	// Default is the schema's "default" value, stringified regardless of
	// its JSON type (a deliberate simplification - see stringifyJSONValue),
	// or "" if the schema declares none.
	Default string
	// Enum is the schema's "enum" values, stringified, or nil if the
	// schema doesn't constrain this field to a fixed set of values.
	Enum []string
	// MinLength/MaxLength/Minimum/Maximum mirror the schema's own
	// constraints of the same name, nil where the schema declares none.
	MinLength *int
	MaxLength *int
	Minimum   *float64
	Maximum   *float64
}

// jsonRequestBodyFields returns the top-level properties of an operation's
// `application/json` request body schema, sorted by name. Local $ref
// chains (e.g. "#/components/schemas/Foo") are resolved via
// resolveSchemaRef; anything else (external refs, allOf/oneOf composition,
// other content types) is left alone for now - scope deliberately limited
// to flat JSON object bodies.
func jsonRequestBodyFields(doc, op map[string]interface{}) []bodyField {
	requestBody, _ := op["requestBody"].(map[string]interface{})
	if requestBody == nil {
		return nil
	}
	content, _ := requestBody["content"].(map[string]interface{})
	if content == nil {
		return nil
	}
	jsonContent, _ := content["application/json"].(map[string]interface{})
	if jsonContent == nil {
		return nil
	}
	schema, _ := jsonContent["schema"].(map[string]interface{})
	if schema == nil {
		return nil
	}
	schema = resolveSchemaRef(doc, schema)

	properties, _ := schema["properties"].(map[string]interface{})
	if len(properties) == 0 {
		return nil
	}
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	fields := make([]bodyField, len(names))
	for i, name := range names {
		field := bodyField{Name: name}
		if prop, ok := properties[name].(map[string]interface{}); ok {
			field.Type, _ = prop["type"].(string)
			if def, ok := prop["default"]; ok {
				field.Default = stringifyJSONValue(def)
			}
			if enumRaw, ok := prop["enum"].([]interface{}); ok {
				for _, e := range enumRaw {
					field.Enum = append(field.Enum, stringifyJSONValue(e))
				}
			}
			if v, ok := prop["minLength"].(float64); ok {
				n := int(v)
				field.MinLength = &n
			}
			if v, ok := prop["maxLength"].(float64); ok {
				n := int(v)
				field.MaxLength = &n
			}
			if v, ok := prop["minimum"].(float64); ok {
				field.Minimum = &v
			}
			if v, ok := prop["maximum"].(float64); ok {
				field.Maximum = &v
			}
		}
		fields[i] = field
	}
	return fields
}

// stringifyJSONValue renders a decoded JSON value (string/float64/bool/nil,
// per encoding/json's default map[string]interface{} unmarshal) as plain
// text, for a schema's "default"/"enum" entries - Volt's body params are
// always edited as strings regardless of the schema's declared type.
func stringifyJSONValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return ""
	default:
		return fmt.Sprint(val)
	}
}

// maxSchemaRefDepth caps how many "$ref": "#/components/schemas/Name"
// hops resolveSchemaRef will follow (A -> B -> C -> ...). Besides bounding
// pathological specs, this also protects against a circular ref chain
// (A -> B -> A) spinning forever - resolution just stops at the cap and
// works with whatever schema it landed on.
const maxSchemaRefDepth = 10

// resolveSchemaRef follows a chain of "$ref": "#/components/schemas/Name"
// (up to maxSchemaRefDepth hops) to the schema it ultimately points at.
// Any other shape of $ref (external files, non-schema targets) stops
// resolution at that point.
func resolveSchemaRef(doc, schema map[string]interface{}) map[string]interface{} {
	const localSchemaPrefix = "#/components/schemas/"

	current := schema
	for i := 0; i < maxSchemaRefDepth; i++ {
		ref, ok := current["$ref"].(string)
		if !ok || !strings.HasPrefix(ref, localSchemaPrefix) {
			return current
		}
		name := strings.TrimPrefix(ref, localSchemaPrefix)

		components, _ := doc["components"].(map[string]interface{})
		schemas, _ := components["schemas"].(map[string]interface{})
		resolved, _ := schemas[name].(map[string]interface{})
		if resolved == nil {
			return current
		}
		current = resolved
	}
	return current
}

// requestBodyTemplate builds a Request.Body value that carries this
// operation's field schema (type/default/enum) under the sentinel
// http.BodySchemaKey - the TUI recognizes that exact key (see
// requestpane.extractBodySpecs) to drive the Body Params UI: showing each
// field's type, pre-filling its default, and offering a picker for enum
// values, instead of a blank freeform textarea.
func requestBodyTemplate(fields []bodyField) string {
	schema := make(map[string]http.BodyFieldSchema, len(fields))
	for _, f := range fields {
		schema[f.Name] = http.BodyFieldSchema{
			Type:      f.Type,
			Default:   f.Default,
			Enum:      f.Enum,
			MinLength: f.MinLength,
			MaxLength: f.MaxLength,
			Minimum:   f.Minimum,
			Maximum:   f.Maximum,
		}
	}
	wrapper := map[string]interface{}{http.BodySchemaKey: schema}
	data, err := json.Marshal(wrapper)
	if err != nil {
		return ""
	}
	return string(data)
}

func operationName(op map[string]interface{}, method, path string) string {
	if id, ok := op["operationId"].(string); ok && id != "" {
		return id
	}
	if summary, ok := op["summary"].(string); ok && summary != "" {
		return summary
	}
	return method + " " + path
}

func truncateName(s string) string {
	r := []rune(s)
	if len(r) <= 40 {
		return s
	}
	return string(r[:40])
}
