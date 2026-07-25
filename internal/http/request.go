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

	if !strings.HasPrefix(r.URL, "http://") && !strings.HasPrefix(r.URL, "https://") {
		return fmt.Errorf("invalid url: %s", r.URL)
	}

	if r.Headers != nil && len(r.Headers) > 100 {
		return fmt.Errorf("too many headers: %d", len(r.Headers))
	}
	if r.Body != "" && len(r.Body) > 10000 {
		return fmt.Errorf("body too long: %d", len(r.Body))
	}

	return nil
}
