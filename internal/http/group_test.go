package http

import "testing"

func TestSubstituteVariables_ReplacesKnownPlaceholders(t *testing.T) {
	vars := map[string]string{"token": "abc123", "host": "api.example.com"}
	got := SubstituteVariables("https://{{host}}/v1?auth={{token}}", vars)
	want := "https://api.example.com/v1?auth=abc123"
	if got != want {
		t.Errorf("SubstituteVariables() = %q, want %q", got, want)
	}
}

func TestSubstituteVariables_TrimsWhitespaceInsidePlaceholder(t *testing.T) {
	vars := map[string]string{"token": "abc123"}
	got := SubstituteVariables("Bearer {{ token }}", vars)
	if got != "Bearer abc123" {
		t.Errorf("SubstituteVariables() = %q, want %q", got, "Bearer abc123")
	}
}

func TestSubstituteVariables_UnknownPlaceholderStaysLiteral(t *testing.T) {
	got := SubstituteVariables("{{missing}}", map[string]string{"token": "abc123"})
	if got != "{{missing}}" {
		t.Errorf("SubstituteVariables() = %q, want the placeholder left untouched", got)
	}
}

func TestSubstituteVariables_NoVariables_ReturnsTextUnchanged(t *testing.T) {
	text := "{{token}} stays as-is"
	if got := SubstituteVariables(text, nil); got != text {
		t.Errorf("SubstituteVariables() = %q, want %q", got, text)
	}
}

func TestSubstituteVariables_AdjacentPlaceholders(t *testing.T) {
	vars := map[string]string{"a": "1", "b": "2"}
	got := SubstituteVariables("{{a}}{{b}}", vars)
	if got != "12" {
		t.Errorf("SubstituteVariables() = %q, want %q", got, "12")
	}
}

func TestServer_MergeHeaders_NilServer(t *testing.T) {
	var s *Server
	headers := map[string]string{"X-Env": "prod"}
	if got := s.MergeHeaders(headers); got["X-Env"] != "prod" {
		t.Errorf("expected requestHeaders unchanged for a nil server, got %+v", got)
	}
}
