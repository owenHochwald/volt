// Package authjar remembers cookies and authorization headers per host, in
// memory only, for the lifetime of the process - never persisted to disk or
// the database (see the response pane's "remember"/"view saved" actions,
// and requestpane's resolvedRequest, which is what actually replays them
// into future requests to the same host).
package authjar

import (
	"net/http"
	"sort"
	"strings"
	"sync"
)

// Entry is a single remembered cookie or authorization header, as shown by
// Snapshot and deleted by DeleteCookie/DeleteAuthHeader.
type Entry struct {
	Name  string
	Value string
}

type hostEntry struct {
	cookies     map[string]string
	authHeaders map[string]string
}

// Store is an in-memory, concurrency-safe collection of cookies and
// authorization headers remembered per host. Every method is safe to call
// on a nil *Store (a no-op, or a zero value for reads) so callers that
// haven't been given a real Store - tests, mainly - never need a nil
// check of their own, the same convention this codebase already uses for
// a nil *storage.SQLiteStorage.
type Store struct {
	mu    sync.Mutex
	hosts map[string]*hostEntry
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{hosts: make(map[string]*hostEntry)}
}

// RememberCookies stores or overwrites cookies for host - name/value pairs
// as returned by (&http.Response{Header: h}).Cookies(), which is how
// callers should parse a response's raw Set-Cookie headers before calling
// this (see responsepane's capture path).
func (s *Store) RememberCookies(host string, cookies []*http.Cookie) {
	if s == nil || host == "" || len(cookies) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entryLocked(host)
	for _, c := range cookies {
		if c.Name == "" {
			continue
		}
		e.cookies[c.Name] = c.Value
	}
}

// RememberAuthHeader stores or overwrites a single header (e.g.
// "Authorization") for host.
func (s *Store) RememberAuthHeader(host, name, value string) {
	if s == nil || host == "" || name == "" || value == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entryLocked(host)
	e.authHeaders[name] = value
}

// entryLocked returns host's entry, creating it if needed. Callers must
// hold s.mu.
func (s *Store) entryLocked(host string) *hostEntry {
	e, ok := s.hosts[host]
	if !ok {
		e = &hostEntry{cookies: map[string]string{}, authHeaders: map[string]string{}}
		s.hosts[host] = e
	}
	return e
}

// HeadersFor returns the headers that should be merged - as the
// lowest-precedence layer, see requestpane.resolvedRequest - into every
// request to host: remembered auth headers verbatim, plus a single
// "Cookie" header built from every remembered cookie (name=value pairs,
// "; "-joined per RFC 6265 §5.4, sorted by name for a deterministic
// result). Returns nil if nothing's remembered for host.
func (s *Store) HeadersFor(host string) map[string]string {
	if s == nil || host == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.hosts[host]
	if !ok {
		return nil
	}
	headers := make(map[string]string, len(e.authHeaders)+1)
	for name, value := range e.authHeaders {
		headers[name] = value
	}
	if cookie := buildCookieHeader(e.cookies); cookie != "" {
		headers["Cookie"] = cookie
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

func buildCookieHeader(cookies map[string]string) string {
	if len(cookies) == 0 {
		return ""
	}
	names := make([]string, 0, len(cookies))
	for name := range cookies {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, name+"="+cookies[name])
	}
	return strings.Join(parts, "; ")
}

// Snapshot is everything remembered for a host, sorted by name - used by
// the response pane's viewer to show what's stored and let the user
// delete individual entries.
type Snapshot struct {
	Host        string
	Cookies     []Entry
	AuthHeaders []Entry
}

// IsEmpty reports whether nothing's remembered at all - used to decide
// whether to show a "saved for this server" indicator.
func (snap Snapshot) IsEmpty() bool {
	return len(snap.Cookies) == 0 && len(snap.AuthHeaders) == 0
}

// Snapshot returns everything currently remembered for host.
func (s *Store) Snapshot(host string) Snapshot {
	snap := Snapshot{Host: host}
	if s == nil || host == "" {
		return snap
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.hosts[host]
	if !ok {
		return snap
	}
	for name, value := range e.cookies {
		snap.Cookies = append(snap.Cookies, Entry{Name: name, Value: value})
	}
	for name, value := range e.authHeaders {
		snap.AuthHeaders = append(snap.AuthHeaders, Entry{Name: name, Value: value})
	}
	sort.Slice(snap.Cookies, func(i, j int) bool { return snap.Cookies[i].Name < snap.Cookies[j].Name })
	sort.Slice(snap.AuthHeaders, func(i, j int) bool { return snap.AuthHeaders[i].Name < snap.AuthHeaders[j].Name })
	return snap
}

// DeleteCookie removes a single remembered cookie for host.
func (s *Store) DeleteCookie(host, name string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.hosts[host]; ok {
		delete(e.cookies, name)
	}
}

// DeleteAuthHeader removes a single remembered authorization header for
// host.
func (s *Store) DeleteAuthHeader(host, name string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.hosts[host]; ok {
		delete(e.authHeaders, name)
	}
}

// Clear removes everything remembered for host - cookies and auth headers
// alike (the response pane's "clear all" action).
func (s *Store) Clear(host string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.hosts, host)
}
