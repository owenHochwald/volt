package http

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/owenHochwald/Volt/internal/utils"
)

// Server is a named base URL that one or more request groups (or a request
// directly, via ServerID) can point at, e.g. "prod" -> https://api.example.com
// or "local" -> http://localhost:3000. Since different servers are commonly
// how a user already models different deployment environments, Server
// doubles as Volt's "environment" - Variables are that environment's own
// values, substituted into {{name}} placeholders (see SubstituteVariables)
// wherever a request resolves against this server, rather than introducing
// a separate environment concept alongside it.
type Server struct {
	ID      int64  `json:"id,omitempty"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	// Headers are applied to every request that resolves against this
	// server (directly via Request.ServerID, or via its group's server),
	// merged underneath the request's own headers - see MergeHeaders.
	Headers map[string]string `json:"headers,omitempty"`
	// Variables are this server's {{name}} placeholder values (see
	// SubstituteVariables) - e.g. {"token": "abc123"} lets a header value
	// of "Bearer {{token}}" resolve per-server without retyping the
	// secret into every request that needs it.
	Variables map[string]string `json:"variables,omitempty"`
}

func (s *Server) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if len(s.Name) > 40 {
		return fmt.Errorf("server name too long: %s", s.Name)
	}
	if !strings.HasPrefix(s.BaseURL, "http") && !utils.IsUnixSocketURL(s.BaseURL) {
		return fmt.Errorf("invalid server base url: %s", s.BaseURL)
	}
	return nil
}

// MergeHeaders overlays requestHeaders on top of this server's default
// headers - the server's values apply to every request that uses it, and
// the request's own headers win on a name collision. Safe to call on a nil
// server (returns requestHeaders unchanged).
func (s *Server) MergeHeaders(requestHeaders map[string]string) map[string]string {
	if s == nil || len(s.Headers) == 0 {
		return requestHeaders
	}
	merged := make(map[string]string, len(s.Headers)+len(requestHeaders))
	for k, v := range s.Headers {
		merged[k] = v
	}
	for k, v := range requestHeaders {
		merged[k] = v
	}
	return merged
}

// variablePlaceholder matches a {{name}} placeholder - name may contain
// any character except "}" (so "{{a}}{{b}}" is two placeholders, not one
// spanning both), trimmed of surrounding whitespace so "{{ name }}" and
// "{{name}}" are equivalent (a common typo/style difference other API
// clients using this same {{...}} convention already tolerate).
var variablePlaceholder = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)

// SubstituteVariables replaces every {{name}} placeholder in text with
// vars[name] - a placeholder naming a variable vars doesn't have is left
// exactly as written (e.g. "{{missing}}" stays literal), rather than
// being blanked out, so a typo'd or not-yet-defined variable is obvious
// in the resolved request instead of silently vanishing.
func SubstituteVariables(text string, vars map[string]string) string {
	if len(vars) == 0 || !strings.Contains(text, "{{") {
		return text
	}
	return variablePlaceholder.ReplaceAllStringFunc(text, func(match string) string {
		name := variablePlaceholder.FindStringSubmatch(match)[1]
		if value, ok := vars[name]; ok {
			return value
		}
		return match
	})
}

// Group is a one-level folder for saved requests, optionally bound to a
// Server so its requests' relative URLs resolve against that server.
type Group struct {
	ID       int64  `json:"id,omitempty"`
	Name     string `json:"name"`
	ServerID *int64 `json:"server_id,omitempty"`
}

func (g *Group) Validate() error {
	if g.Name == "" {
		return fmt.Errorf("group name is required")
	}
	if len(g.Name) > 40 {
		return fmt.Errorf("group name too long: %s", g.Name)
	}
	return nil
}
