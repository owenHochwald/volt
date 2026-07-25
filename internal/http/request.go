package http

import (
	"encoding/json"
	"net/url"
	"slices"
	"strings"

	"github.com/owenHochwald/Volt/internal/apperror"
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

	parsedURL, err := url.Parse(r.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return apperror.InvalidURL(r.URL)
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

func isJSONRequest(headers map[string]string) bool {
	for key, value := range headers {
		if strings.EqualFold(key, "Content-Type") && strings.Contains(strings.ToLower(value), "application/json") {
			return true
		}
	}
	return false
}
