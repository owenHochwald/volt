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
			req.Description, _ = op["description"].(string)
			if fields := jsonRequestBodyFields(doc, op); len(fields) > 0 {
				req.Body = requestBodyTemplate(fields)
			}
			// Always set, even to an empty (non-nil) map for an operation
			// with no path parameters of its own - requestpane.
			// IsSchemaRequest uses PathParamSchema != nil as its marker
			// for "this request came from an import", regardless of
			// whether that import actually found any path (or body)
			// fields to describe, so an imported bodyless/paramless
			// endpoint doesn't look indistinguishable from a hand-typed
			// request once loaded back from the database.
			req.PathParamSchema = pathParamFields(doc, item, op)
			// Same "always set, even empty" reasoning as PathParamSchema
			// above - see queryParamFields.
			req.QueryParamSchema = queryParamFields(doc, item, op)

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
	// scalar field is still edited as plain text in the TUI regardless of
	// Type - it's shown only as a hint - except "array", which is resolved
	// further (see Items) when it's an array of objects.
	Type string
	// Default is the schema's "default" value, stringified regardless of
	// its JSON type (a deliberate simplification - see stringifyJSONValue),
	// or "" if the schema declares none.
	Default string
	// Enum is the schema's "enum" values, stringified, or nil if the
	// schema doesn't constrain this field to a fixed set of values.
	Enum []string
	// Description is the property's own "description", or "" if the
	// schema declares none.
	Description string
	// MinLength/MaxLength/Minimum/Maximum mirror the schema's own
	// constraints of the same name, nil where the schema declares none.
	MinLength *int
	MaxLength *int
	Minimum   *float64
	Maximum   *float64
	// Items/ItemsRefName/ItemType mirror http.BodyFieldSchema's fields of
	// the same name - see extractFieldSchema for how they're derived.
	Items        map[string]http.BodyFieldSchema
	ItemsRefName string
	ItemType     string
	// Required mirrors whether this field's name appeared in the parent
	// object schema's "required" list - see requiredSet.
	Required bool
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

	required := requiredSet(schema)
	fields := make([]bodyField, len(names))
	for i, name := range names {
		field := bodyField{Name: name, Required: required[name]}
		if prop, ok := properties[name].(map[string]interface{}); ok {
			schema := extractFieldSchema(doc, prop, 0)
			field.Type = schema.Type
			field.Default = schema.Default
			field.Enum = schema.Enum
			field.Description = schema.Description
			field.MinLength = schema.MinLength
			field.MaxLength = schema.MaxLength
			field.Minimum = schema.Minimum
			field.Maximum = schema.Maximum
			field.Items = schema.Items
			field.ItemsRefName = schema.ItemsRefName
			field.ItemType = schema.ItemType
		}
		fields[i] = field
	}
	return fields
}

// pathParamFields extracts the "in: path" parameters declared for a path
// item/operation pair, keyed by name - path-item-level parameters (shared
// across every verb on that path, e.g. the {host_id} in
// "/hosts/{host_id}") merged with the operation's own (which, per the
// OpenAPI spec, override a path-item entry of the same name+in). Used to
// populate http.Request.PathParamSchema so the TUI can validate a path
// param's value the same way it already does for body fields (type,
// min/maxLength, minimum/maximum, enum, required) instead of treating
// every {name} URL placeholder as unconstrained free text. Local $ref
// chains in a parameter's own "schema" are resolved via extractFieldSchema
// the same way a body property's schema is; a $ref'd *parameter object*
// itself (rather than its schema) isn't resolved, matching this package's
// existing scope (see jsonRequestBodyFields's doc comment).
func pathParamFields(doc, item, op map[string]interface{}) map[string]http.BodyFieldSchema {
	fields := make(map[string]http.BodyFieldSchema)
	collect := func(raw interface{}) {
		params, _ := raw.([]interface{})
		for _, p := range params {
			param, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if in, _ := param["in"].(string); in != "path" {
				continue
			}
			name, _ := param["name"].(string)
			if name == "" {
				continue
			}
			paramSchema, _ := param["schema"].(map[string]interface{})
			field := extractFieldSchema(doc, paramSchema, 0)
			// A parameter's own "description" lives on the parameter
			// object itself, a sibling of "schema" - not inside the
			// schema extractFieldSchema just read (that's where a body
			// property's description lives instead). Override rather
			// than only filling in when empty: the parameter-level text
			// is the one an OpenAPI author actually meant for this
			// specific field.
			if desc, ok := param["description"].(string); ok && desc != "" {
				field.Description = desc
			}
			// A path parameter's own "required" - distinct from a body
			// property's parent-object "required" list (see requiredSet) -
			// is spec-mandated true for any "in: path" parameter; read an
			// explicit value if present (a spec that gets this wrong still
			// shouldn't crash), defaulting to true otherwise rather than
			// treating an unspecified one as silently optional.
			field.Required = true
			if req, ok := param["required"].(bool); ok {
				field.Required = req
			}
			fields[name] = field
		}
	}
	collect(item["parameters"])
	collect(op["parameters"])
	return fields
}

// queryParamFields mirrors pathParamFields for a spec's "in: query"
// parameters (http.Request.QueryParamSchema - see requestpane.QueryParams),
// so an imported endpoint's query string is editable as individually
// validated fields instead of typed directly into the URL. Unlike a path
// parameter, a query parameter isn't spec-mandated required by virtue of
// existing - it defaults to false (an ordinary optional query string
// value) unless the spec explicitly says otherwise, the same default
// "required" itself already has anywhere else in this package.
func queryParamFields(doc, item, op map[string]interface{}) map[string]http.BodyFieldSchema {
	fields := make(map[string]http.BodyFieldSchema)
	collect := func(raw interface{}) {
		params, _ := raw.([]interface{})
		for _, p := range params {
			param, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			if in, _ := param["in"].(string); in != "query" {
				continue
			}
			name, _ := param["name"].(string)
			if name == "" {
				continue
			}
			paramSchema, _ := param["schema"].(map[string]interface{})
			field := extractFieldSchema(doc, paramSchema, 0)
			// See pathParamFields' identical override for why this reads
			// the parameter object's own "description" rather than
			// relying on whatever extractFieldSchema found in "schema".
			if desc, ok := param["description"].(string); ok && desc != "" {
				field.Description = desc
			}
			if req, ok := param["required"].(bool); ok {
				field.Required = req
			}
			fields[name] = field
		}
	}
	collect(item["parameters"])
	collect(op["parameters"])
	return fields
}

// requiredSet turns a schema's "required" array (a sibling of its
// "properties" map, per the OpenAPI/JSON Schema convention) into a set of
// property names, for marking which of them must be filled in before a
// request built from this schema can be sent. Returns nil (not an empty,
// non-nil map) when the schema declares no required properties, so a
// simple `required[name]` lookup on a nil map correctly reads false.
func requiredSet(schema map[string]interface{}) map[string]bool {
	raw, _ := schema["required"].([]interface{})
	if len(raw) == 0 {
		return nil
	}
	set := make(map[string]bool, len(raw))
	for _, r := range raw {
		if name, ok := r.(string); ok {
			set[name] = true
		}
	}
	return set
}

// maxArrayItemDepth caps how many levels deep an array-of-objects' own
// object properties are resolved (an item schema that's itself an array of
// objects, and so on) - besides bounding pathological specs, this protects
// against a circular $ref chain between object schemas spinning forever.
const maxArrayItemDepth = 3

// extractFieldSchema converts one OpenAPI property schema into the shape
// Volt's body-params UI understands. For a "type: array" property whose
// "items" is a (possibly $ref'd) object schema, it also recurses into that
// object's own properties (up to maxArrayItemDepth levels) so the array can
// be composed as a repeatable set of sub-fields in the TUI (see
// requestpane.ArrayParam) instead of being left as opaque text - a "simple"
// array (items with no object structure) is left with Items/ItemsRefName
// unset, editable as a whole via other means.
func extractFieldSchema(doc, prop map[string]interface{}, depth int) http.BodyFieldSchema {
	// A property can be entirely a $ref (e.g. "priority: {$ref:
	// '#/components/schemas/HostPriority'}") rather than declaring type/
	// constraints inline - resolved first so every field below reads from
	// the schema those constraints actually live on, the same way the
	// request body's own top-level schema is already resolved before its
	// "properties" map is read (see jsonRequestBodyFields). Without this,
	// a $ref'd property silently lost its type, enum, and min/max/length
	// constraints - Volt had no schema to read them from.
	prop = resolveSchemaRef(doc, prop)
	var field http.BodyFieldSchema
	field.Type, _ = prop["type"].(string)
	field.Description, _ = prop["description"].(string)
	if def, ok := prop["default"]; ok {
		field.Default = stringifyJSONValue(def)
	}
	if enumRaw, ok := prop["enum"].([]interface{}); ok {
		for _, e := range enumRaw {
			field.Enum = append(field.Enum, stringifyJSONValue(e))
		}
	}
	if v, ok := asFloat64(prop["minLength"]); ok {
		n := int(v)
		field.MinLength = &n
	}
	if v, ok := asFloat64(prop["maxLength"]); ok {
		n := int(v)
		field.MaxLength = &n
	}
	if v, ok := asFloat64(prop["minimum"]); ok {
		field.Minimum = &v
	}
	if v, ok := asFloat64(prop["maximum"]); ok {
		field.Maximum = &v
	}

	if field.Type != "array" || depth >= maxArrayItemDepth {
		return field
	}
	itemsRaw, ok := prop["items"].(map[string]interface{})
	if !ok {
		return field
	}
	if ref, ok := itemsRaw["$ref"].(string); ok {
		field.ItemsRefName = schemaRefName(ref)
	}
	resolvedItems := resolveSchemaRef(doc, itemsRaw)
	itemProperties, _ := resolvedItems["properties"].(map[string]interface{})
	if len(itemProperties) == 0 {
		// A "simple" array - its items have no object structure to compose
		// (e.g. items: {type: number}) - still declares a scalar item type
		// worth keeping (see http.BodyFieldSchema.ItemType), rather than
		// discarding it and leaving the array's entries typeless.
		field.ItemType, _ = resolvedItems["type"].(string)
		return field
	}
	itemRequired := requiredSet(resolvedItems)
	field.Items = make(map[string]http.BodyFieldSchema, len(itemProperties))
	for name, raw := range itemProperties {
		if itemProp, ok := raw.(map[string]interface{}); ok {
			itemField := extractFieldSchema(doc, itemProp, depth+1)
			itemField.Required = itemRequired[name]
			field.Items[name] = itemField
		}
	}
	return field
}

// schemaRefName extracts the component name from a local schema $ref (e.g.
// "#/components/schemas/Tag" -> "Tag"), or "" for any other ref shape.
func schemaRefName(ref string) string {
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}
	return ""
}

// stringifyJSONValue renders a decoded scalar value as plain text, for a
// schema's "default"/"enum" entries - Volt's body params are always edited
// as strings regardless of the schema's declared type. Handles both
// encoding/json's decoding (numbers always float64) and gopkg.in/yaml.v3's
// (a whole-number scalar decodes as int, only float64 when it has a
// decimal point - see also asFloat64).
func stringifyJSONValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'g', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return ""
	default:
		return fmt.Sprint(val)
	}
}

// asFloat64 extracts a schema constraint's numeric value regardless of
// whether the containing document was parsed as JSON (encoding/json always
// decodes numbers as float64) or YAML (gopkg.in/yaml.v3 decodes a
// whole-number scalar as int, only using float64 when it has a decimal
// point) - without this, minLength/maxLength (always whole numbers) would
// silently fail to extract from any YAML spec, since the type assertion to
// float64 would never match.
func asFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
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
			Type:         f.Type,
			Default:      f.Default,
			Enum:         f.Enum,
			Description:  f.Description,
			MinLength:    f.MinLength,
			MaxLength:    f.MaxLength,
			Minimum:      f.Minimum,
			Maximum:      f.Maximum,
			Items:        f.Items,
			ItemsRefName: f.ItemsRefName,
			ItemType:     f.ItemType,
			Required:     f.Required,
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
