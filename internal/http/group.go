package http

import (
	"fmt"
	"strings"
)

// Server is a named base URL that one or more request groups (or a request
// directly, via ServerID) can point at, e.g. "prod" -> https://api.example.com
// or "local" -> http://localhost:3000.
type Server struct {
	ID      int64  `json:"id,omitempty"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	// Headers are applied to every request that resolves against this
	// server (directly via Request.ServerID, or via its group's server),
	// merged underneath the request's own headers - see MergeHeaders.
	Headers map[string]string `json:"headers,omitempty"`
}

func (s *Server) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("server name is required")
	}
	if len(s.Name) > 40 {
		return fmt.Errorf("server name too long: %s", s.Name)
	}
	if !strings.HasPrefix(s.BaseURL, "http") {
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
