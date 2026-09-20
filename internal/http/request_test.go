package http

import (
	"testing"

	"github.com/owenHochwald/Volt/internal/apperror"
)

func TestRequest_Validate(t *testing.T) {
	type fields struct {
		ID      int64
		Name    string
		Method  string
		URL     string
		Headers map[string]string
		Body    string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"empty method", fields{Method: ""}, true},
		{"empty url", fields{URL: ""}, true},
		{"name too long", fields{Name: "name too longname too longname too longname too long"}, true},
		{"valid", fields{Method: GET, URL: "http://localhost"}, false},
		{"valid with id", fields{ID: 1234, Method: GET, URL: "http://localhost"}, false},
		{"valid with name", fields{Name: "test", Method: GET, URL: "http://localhost"}, false},
		{"valid with headers", fields{Method: GET, URL: "http://localhost", Headers: map[string]string{"Content-Type": "application/json"}}, false},
		{"valid with body", fields{Method: GET, URL: "http://localhost", Body: "test"}, false},
		{"invalid method", fields{Method: "GETT", URL: "http://localhost"}, true},
		{"invalid url", fields{Method: GET, URL: "htt://localhost:8080"}, true},
		{"url shorter than prefix", fields{Method: GET, URL: "ab"}, true},
		{"one-character url", fields{Method: GET, URL: "h"}, true},
		{"three-character url", fields{Method: GET, URL: "htt"}, true},
		// A bare path is valid too - it's meant to be resolved against a
		// group's/server's base URL at send time (see EffectiveURL), not
		// sent as-is, so it's exempt from the scheme-prefix check.
		{"group-relative path", fields{Method: GET, URL: "/pets"}, false},
		// A "unix://" URL (see utils.ParseUnixSocketURL) is checked on its
		// own terms - a socket path, not an http(s) scheme/host.
		{"valid unix socket url", fields{Method: GET, URL: "unix:///var/run/docker.sock:/containers/json"}, false},
		{"valid unix socket url with no request path", fields{Method: GET, URL: "unix:///var/run/docker.sock"}, false},
		{"unix socket url with empty socket path", fields{Method: GET, URL: "unix://:/health"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Request{
				ID:      tt.fields.ID,
				Name:    tt.fields.Name,
				Method:  tt.fields.Method,
				URL:     tt.fields.URL,
				Headers: tt.fields.Headers,
				Body:    tt.fields.Body,
			}
			if err := r.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHostOf(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"absolute https", "https://api.example.com/v1/pets", "api.example.com"},
		{"absolute http with port", "http://localhost:8080/health", "localhost:8080"},
		{"mixed case host is lowercased", "https://API.Example.COM/v1", "api.example.com"},
		{"relative group path has no host", "/v1/pets", ""},
		{"empty URL has no host", "", ""},
		{"unix socket url uses the socket path as its host", "unix:///var/run/docker.sock:/containers/json", "unix:///var/run/docker.sock"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HostOf(tt.url); got != tt.want {
				t.Errorf("HostOf(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestRequest_IsAbsoluteURL_UnixSocket(t *testing.T) {
	r := &Request{URL: "unix:///var/run/docker.sock:/containers/json"}
	if !r.IsAbsoluteURL() {
		t.Error("expected a unix:// URL to be reported as absolute")
	}
}

// TestRequest_EffectiveURL_UnixSocketServer verifies a group-relative
// request (a bare "/path") resolves against a server whose BaseURL is
// itself a "unix://" socket URL by becoming a new request path against
// that same socket - not by naively string-concatenating onto BaseURL,
// which would produce a malformed "unix://...sock:/base-path/path" that's
// no longer a socket path followed by exactly one request path.
func TestRequest_EffectiveURL_UnixSocketServer(t *testing.T) {
	r := &Request{URL: "/containers/json"}
	server := &Server{BaseURL: "unix:///var/run/docker.sock"}

	want := "unix:///var/run/docker.sock:/containers/json"
	if got := r.EffectiveURL(server); got != want {
		t.Errorf("EffectiveURL() = %q, want %q", got, want)
	}
}

// TestRequest_EffectiveURL_UnixSocketRequestIgnoresServer verifies a
// request whose own URL is already a "unix://" one is sent as-is,
// ignoring any server - the same "already absolute" precedence an
// ordinary http(s) request URL already gets.
func TestRequest_EffectiveURL_UnixSocketRequestIgnoresServer(t *testing.T) {
	r := &Request{URL: "unix:///tmp/app.sock:/status"}
	server := &Server{BaseURL: "https://api.example.com"}

	want := "unix:///tmp/app.sock:/status"
	if got := r.EffectiveURL(server); got != want {
		t.Errorf("EffectiveURL() = %q, want %q", got, want)
	}
}

func TestRequestValidateRejectsMalformedJSON(t *testing.T) {
	request := &Request{
		Method:  POST,
		URL:     "https://example.com",
		Headers: map[string]string{"content-type": "application/json; charset=utf-8"},
		Body:    `{"unfinished":`,
	}

	err := request.Validate()
	failure, ok := err.(*apperror.Error)
	if !ok {
		t.Fatalf("error = %T, want *apperror.Error", err)
	}
	if failure.Category != apperror.Validation || failure.Code != apperror.InvalidJSON {
		t.Fatalf("error = (%s, %s), want (%s, %s)", failure.Category, failure.Code, apperror.Validation, apperror.InvalidJSON)
	}
	if failure.Hint != "Fix the JSON syntax, then try again." {
		t.Fatalf("hint = %q", failure.Hint)
	}
}
