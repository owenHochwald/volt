package http

import (
	"fmt"
	"slices"
	"strings"
)

const (
	GET = "GET"
	//HEAD   = "HEAD"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
	//CONNECT = "CONNECT"
	PATCH = "PATCH"
	//OPTIONS = "OPTIONS"
	//TRACE   = "TRACE"
)

var validMethods = []string{
	GET,
	//HEAD,
	POST,
	PUT,
	PATCH,
	DELETE,
	//CONNECT,
	//OPTIONS,
	//TRACE,
}

type Request struct {
	ID      int64             `json:"id,omitempty"`
	Name    string            `json:"name,omitempty"`
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	// GroupID, when set, ties this request to a request group. If URL is a
	// path (e.g. "/pets") rather than an absolute URL, it is meant to be
	// resolved against that group's configured Server at send time - see
	// EffectiveURL.
	GroupID *int64 `json:"group_id,omitempty"`
	// ServerID, when set, picks a server directly for this request -
	// independent of GroupID - to resolve a relative URL against and to
	// merge that server's default headers in. It takes priority over
	// GroupID's server when both are set (see storage.ResolveEffectiveServer).
	ServerID *int64 `json:"server_id,omitempty"`
	// PathParamSchema holds each {name} URL placeholder's OpenAPI
	// parameter schema (type/enum/min-max/required), keyed by name - the
	// path-param counterpart of a body's BodySchemaKey-wrapped template
	// (see BodyFieldSchema), but stored directly on the request rather
	// than smuggled into Body (path params aren't body content). Set by
	// `volt import` when the spec declares a path parameter's schema; nil
	// for a request created/imported before this existed, or whose spec
	// declared no schema for a given placeholder - either way, that
	// placeholder is just edited as unconstrained free text (see
	// requestpane.rebuildParams).
	PathParamSchema map[string]BodyFieldSchema `json:"path_param_schema,omitempty"`
	// ParamValues holds every currently typed/selected path/body/array
	// param value for this request (see ParamValues), independent of the
	// schema/template itself (Body/PathParamSchema) - so reloading a saved
	// request (see requestpane.RequestPane's reinitRequestPane) restores
	// exactly what was filled in, not just blank fields matching its
	// schema. Refreshed on every explicit save (see requestpane.
	// RequestPane.Snapshot) and by app restore of the last request
	// actually sent (see storage.SaveLastSentRequest/LoadLastSentRequest);
	// cleared by ClearParams ("ctrl+x" - see requestpane.
	// RequestPane.clearParamValues) back to the schema's own defaults.
	ParamValues ParamValues `json:"param_values,omitempty"`
}

// ParamValues holds every currently typed/selected field value for a
// request - scalar path/body param values, and array param items - keyed
// by field name (or, for a "simple" array's lone unnamed field, "" - see
// requestpane.simpleArrayItemName).
type ParamValues struct {
	PathParamValues  map[string]string              `json:"path_param_values,omitempty"`
	BodyParamValues  map[string]string              `json:"body_param_values,omitempty"`
	ArrayParamValues map[string][]map[string]string `json:"array_param_values,omitempty"`
}

func NewBlankRequest() *Request {
	return &Request{
		Method: "",
	}
}

func NewDefaultRequest() *Request {
	return &Request{
		Name:    "None",
		Method:  GET,
		URL:     "https://:",
		Headers: make(map[string]string),
		Body:    "",
	}
}

func NewRequestWithParams(method, url string) *Request {
	return &Request{
		Method:  method,
		URL:     url,
		Headers: make(map[string]string),
	}
}

func (r *Request) Validate() error {
	if r.Name != "" && len(r.Name) > 40 {
		return fmt.Errorf("name too long: %s", r.Name)
	}
	if r.Method == "" {
		return fmt.Errorf("method is required")
	}
	if r.URL == "" {
		return fmt.Errorf("url is required")
	}

	if !slices.Contains(validMethods, r.Method) {
		return fmt.Errorf("invalid method: %s", r.Method)
	}

	if r.URL != "" {
		// A URL may also be a bare path (e.g. "/pets"), meant to be resolved
		// against a group's server at send time - see EffectiveURL.
		if !strings.HasPrefix(r.URL, "http") && !strings.HasPrefix(r.URL, "/") {
			return fmt.Errorf("invalid url: %s", r.URL)
		}
	}

	if r.Headers != nil && len(r.Headers) > 100 {
		return fmt.Errorf("too many headers: %d", len(r.Headers))
	}
	if r.Body != "" && len(r.Body) > 10000 {
		return fmt.Errorf("body too long: %d", len(r.Body))
	}

	return nil
}

// IsAbsoluteURL reports whether r.URL is directly sendable as-is, as
// opposed to a group-relative path that still needs a server to resolve
// against (see EffectiveURL).
func (r *Request) IsAbsoluteURL() bool {
	return strings.HasPrefix(r.URL, "http")
}

// EffectiveURL returns r.URL unchanged if it's already absolute. Otherwise,
// if server is non-nil, it joins r.URL onto server.BaseURL. With no server
// available, the relative URL is returned as-is (not directly sendable).
func (r *Request) EffectiveURL(server *Server) string {
	if r.IsAbsoluteURL() {
		return r.URL
	}
	if server == nil || server.BaseURL == "" {
		return r.URL
	}
	return strings.TrimRight(server.BaseURL, "/") + r.URL
}
