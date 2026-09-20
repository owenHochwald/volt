package http

import (
	"encoding/json"
	"net/url"
	"slices"
	"strings"

	"github.com/owenHochwald/Volt/internal/apperror"
	"github.com/owenHochwald/Volt/internal/utils"
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
	// Description is the OpenAPI operation's own "description" - set by
	// `volt import` when the spec declares one, shown read-only in the
	// sidebar (for the currently selected request) and the request pane
	// (under Name - see requestpane's fieldsContent). A hand-created
	// request has no spec to derive one from, so it stays empty.
	Description string `json:"description,omitempty"`
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
	// QueryParamSchema mirrors PathParamSchema for a request's query
	// string parameters (an OpenAPI parameter with "in: query") - editable
	// as individually-validated fields (see requestpane.QueryParams)
	// instead of typed directly into the URL's own "?..." suffix. Only
	// ever set by `volt import` (always, even to an empty non-nil map, for
	// an operation with no query parameters of its own - same "explicit
	// marker" reasoning as PathParamSchema); a hand-typed request has no
	// schema to derive one from, so it keeps typing "?key=value" straight
	// into the URL, same as before this existed.
	QueryParamSchema map[string]BodyFieldSchema `json:"query_param_schema,omitempty"`
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
	QueryParamValues map[string]string              `json:"query_param_values,omitempty"`
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
		return apperror.ValidationError(apperror.InvalidConfig, "The request name is too long.", "Use a name with 40 characters or fewer.")
	}
	if r.Method == "" {
		return apperror.ValidationError(apperror.InvalidConfig, "Choose an HTTP method.", "Press ? to see the available request controls.")
	}
	if r.URL == "" {
		return apperror.InvalidURL(r.URL)
	}

	if !slices.Contains(validMethods, r.Method) {
		return apperror.ValidationError(apperror.InvalidConfig, "The HTTP method is not supported.", "Choose GET, POST, PUT, PATCH, or DELETE.")
	}

	// A URL may also be a bare path (e.g. "/pets"), meant to be resolved
	// against a group's/server's base URL at send time - see EffectiveURL -
	// exempt from the scheme/host check below. A "unix://" URL (see
	// utils.ParseUnixSocketURL) is exempt too - it isn't a real http(s)
	// URL at all, just a socket path and a request path packed into one
	// string, checked on its own terms instead.
	if socketPath, _, ok := utils.ParseUnixSocketURL(r.URL); ok {
		if socketPath == "" {
			return apperror.InvalidURL(r.URL)
		}
	} else if !strings.HasPrefix(r.URL, "/") {
		parsedURL, err := url.Parse(r.URL)
		if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
			return apperror.InvalidURL(r.URL)
		}
	}

	if r.Headers != nil && len(r.Headers) > 100 {
		return apperror.ValidationError(apperror.InvalidHeaders, "There are too many request headers.", "Use 100 headers or fewer.")
	}
	if r.Body != "" && len(r.Body) > 10000 {
		return apperror.ValidationError(apperror.InvalidConfig, "The request body is too long.", "Use a body with 10,000 characters or fewer.")
	}
	if r.Body != "" && isJSONRequest(r.Headers) && !json.Valid([]byte(r.Body)) {
		return apperror.ValidationError(apperror.InvalidJSON, "The JSON request body is not valid.", "Fix the JSON syntax, then try again.")
	}

	return nil
}

// IsAbsoluteURL reports whether r.URL is directly sendable as-is, as
// opposed to a group-relative path that still needs a server to resolve
// against (see EffectiveURL).
func (r *Request) IsAbsoluteURL() bool {
	return strings.HasPrefix(r.URL, "http") || utils.IsUnixSocketURL(r.URL)
}

// EffectiveURL returns r.URL unchanged if it's already absolute. Otherwise,
// if server is non-nil, it joins r.URL onto server.BaseURL. With no server
// available, the relative URL is returned as-is (not directly sendable).
//
// A server whose BaseURL is itself a "unix://" socket URL (see
// utils.ParseUnixSocketURL) joins differently: r.URL becomes the new
// request path against that same socket, rather than being appended onto
// BaseURL as a literal string suffix - so a group of requests can share
// one Unix socket the same way a group of ordinary requests already
// shares one TCP base URL.
func (r *Request) EffectiveURL(server *Server) string {
	if r.IsAbsoluteURL() {
		return r.URL
	}
	if server == nil || server.BaseURL == "" {
		return r.URL
	}
	if socketPath, _, ok := utils.ParseUnixSocketURL(server.BaseURL); ok {
		return utils.UnixSocketScheme + socketPath + ":" + r.URL
	}
	return strings.TrimRight(server.BaseURL, "/") + r.URL
}

// HostOf returns rawURL's host (lowercased, including a non-default port
// if one's given - e.g. "api.example.com:8443") for a fully-resolved,
// absolute URL, or "" if rawURL can't be parsed as one (a still-relative
// group path with no server to resolve it against, or garbage input).
// Used to key the in-memory per-host cookie/auth-header jar (see
// internal/authjar) - independent of whether a request has a ServerID at
// all, since an ad-hoc absolute-URL request with none still has a host.
//
// A "unix://" URL (see utils.ParseUnixSocketURL) has no host in the usual
// sense - its socket path fills the same role instead, so two requests
// against the same socket still share one jar entry the same way two
// requests against the same TCP host would.
func HostOf(rawURL string) string {
	if socketPath, _, ok := utils.ParseUnixSocketURL(rawURL); ok {
		return utils.UnixSocketScheme + socketPath
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Host)
}

func isJSONRequest(headers map[string]string) bool {
	for key, value := range headers {
		if strings.EqualFold(key, "Content-Type") && strings.Contains(strings.ToLower(value), "application/json") {
			return true
		}
	}
	return false
}
