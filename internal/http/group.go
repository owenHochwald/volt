package http

import (
	"fmt"
	"strings"
)

// Server is a named base URL that one or more request groups can point at,
// e.g. "prod" -> https://api.example.com or "local" -> http://localhost:3000.
type Server struct {
	ID      int64  `json:"id,omitempty"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
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
