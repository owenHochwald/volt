package cli

import (
	"encoding/json"
	"fmt"
	"sort"
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
