package utils

import "strings"

// UnixSocketScheme is the URL prefix Volt recognizes for dialing a Unix
// domain socket instead of a TCP host - e.g.
// "unix:///var/run/docker.sock:/containers/json" sends a request for
// "/containers/json" over the socket at "/var/run/docker.sock". Kept as
// one plain string so it fits the same URL field a TCP request already
// uses (the request pane's URL input, the WebSocket pane's URL input, a
// Server's BaseURL) instead of a separate socket-path input - modeled
// loosely on curl's --unix-socket, folded into the URL the way some other
// unix-socket-aware HTTP tools already do.
//
// Lives here (rather than internal/http, its main caller) so internal/ws
// - deliberately independent of internal/http's request/response model,
// see its own package doc - can recognize the same syntax without
// depending on internal/http at all.
const UnixSocketScheme = "unix://"

// IsUnixSocketURL reports whether rawURL uses the "unix://" scheme.
func IsUnixSocketURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, UnixSocketScheme)
}

// ParseUnixSocketURL splits a "unix://<socket-path>:<request-path>" URL
// into its socket path and request path. ok is false if rawURL doesn't use
// the unix scheme at all. The request path defaults to "/" when omitted
// (e.g. "unix:///var/run/docker.sock" alone, for a bare root health
// check).
//
// The ":" delimiter is unambiguous in practice: a Unix socket path never
// contains one, and only the first "/"-preceded ":" past the scheme is
// treated as the split point - an HTTP path's own colons (e.g. inside a
// query value) always come later, past that first one.
func ParseUnixSocketURL(rawURL string) (socketPath, requestPath string, ok bool) {
	rest, found := strings.CutPrefix(rawURL, UnixSocketScheme)
	if !found {
		return "", "", false
	}
	if idx := strings.Index(rest, ":/"); idx >= 0 {
		return rest[:idx], rest[idx+1:], true
	}
	return rest, "/", true
}
