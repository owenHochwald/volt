package http

import (
	"strings"
	"testing"
)

func TestBuildCurlCommand_GetNoHeadersNoBody(t *testing.T) {
	req := &Request{Method: GET, URL: "https://api.example.com/widgets"}
	got := BuildCurlCommand(req)
	want := "curl -X GET \\\n  'https://api.example.com/widgets'"
	if got != want {
		t.Errorf("BuildCurlCommand() = %q, want %q", got, want)
	}
}

func TestBuildCurlCommand_HeadersSortedByName(t *testing.T) {
	req := &Request{
		Method: GET,
		URL:    "https://api.example.com/widgets",
		Headers: map[string]string{
			"X-Env":         "prod",
			"Authorization": "Bearer abc123",
		},
	}
	got := BuildCurlCommand(req)
	want := "curl -X GET \\\n  'https://api.example.com/widgets' \\\n  -H 'Authorization: Bearer abc123' \\\n  -H 'X-Env: prod'"
	if got != want {
		t.Errorf("BuildCurlCommand() = %q, want %q", got, want)
	}
}

func TestBuildCurlCommand_IncludesBody(t *testing.T) {
	req := &Request{
		Method:  POST,
		URL:     "https://api.example.com/widgets",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    `{"name":"widget"}`,
	}
	got := BuildCurlCommand(req)
	want := "curl -X POST \\\n  'https://api.example.com/widgets' \\\n  -H 'Content-Type: application/json' \\\n  --data-raw '{\"name\":\"widget\"}'"
	if got != want {
		t.Errorf("BuildCurlCommand() = %q, want %q", got, want)
	}
}

func TestBuildCurlCommand_EmptyBodyOmitsDataRaw(t *testing.T) {
	req := &Request{Method: GET, URL: "https://api.example.com/widgets"}
	got := BuildCurlCommand(req)
	if strings.Contains(got, "--data-raw") {
		t.Errorf("expected no --data-raw for an empty body, got %q", got)
	}
}

func TestShellQuote_EscapesSingleQuotes(t *testing.T) {
	got := shellQuote(`it's a "test"`)
	want := `'it'\''s a "test"'`
	if got != want {
		t.Errorf("shellQuote() = %q, want %q", got, want)
	}
}

func TestBuildCurlCommand_HeaderValueWithSingleQuoteIsSafelyEscaped(t *testing.T) {
	req := &Request{
		Method:  GET,
		URL:     "https://api.example.com/widgets",
		Headers: map[string]string{"X-Note": "it's fine"},
	}
	got := BuildCurlCommand(req)
	if !strings.Contains(got, `'X-Note: it'\''s fine'`) {
		t.Errorf("expected the escaped header value, got %q", got)
	}
}
