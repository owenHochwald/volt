package responsepane

import (
	"net/http"
	"strings"

	httppkg "github.com/owenHochwald/Volt/internal/http"
)

// authorizationHeader is the only header captureAuthHeader remembers -
// see its own doc comment for why this is intentionally narrow rather
// than guessing at arbitrary "looks like auth" header names.
const authorizationHeader = "Authorization"

// currentHost returns the host SentRequest was actually sent to, or "" if
// there's no response yet or its URL can't be parsed as absolute (see
// http.HostOf) - the key every capture/injection/viewer call uses into
// the jar, so a request with no configured server still gets its cookies
// remembered by host alone.
func (m *ResponsePane) currentHost() string {
	if m.SentRequest == nil {
		return ""
	}
	return httppkg.HostOf(m.SentRequest.URL)
}

// captureCookies remembers every Set-Cookie header on the current
// Response for its host - called automatically by SetResponse when
// autoRememberCookies is set, and unconditionally by captureNow (the
// manual "c" action).
func (m *ResponsePane) captureCookies() {
	if m.jar == nil || m.Response == nil {
		return
	}
	host := m.currentHost()
	if host == "" {
		return
	}
	// Response.Headers is a plain net/http.Header (see http.Response) -
	// wrapping it in a throwaway *http.Response to reuse its own Cookies()
	// parser (Name/Value/Path/Domain/Expires/... from the raw Set-Cookie
	// strings) instead of hand-rolling that parsing here.
	cookies := (&http.Response{Header: m.Response.Headers}).Cookies()
	m.jar.RememberCookies(host, cookies)
}

// captureAuthHeader remembers the sent request's Authorization header (if
// it set one) for its host. Deliberately only this one specific header,
// not an automatic guess at whatever "looks like auth" - a real HTTP
// response essentially never sends back a ready-made Authorization value
// of its own (that's not what servers do), so the only auth data worth
// remembering here is whatever the user already put on a request that
// then succeeded - typically a login/token request they hand-crafted -
// so it can be replayed on later requests to the same host without
// retyping it every time.
func (m *ResponsePane) captureAuthHeader() {
	if m.jar == nil || m.SentRequest == nil {
		return
	}
	host := m.currentHost()
	if host == "" {
		return
	}
	if value, ok := headerValueCI(m.SentRequest.Headers, authorizationHeader); ok && value != "" {
		m.jar.RememberAuthHeader(host, authorizationHeader, value)
	}
}

// captureNow is CaptureAuth's ("c") handler - always remembers both
// cookies and the sent request's Authorization header for the current
// response's host, regardless of the auto-remember settings (those only
// gate what SetResponse does automatically on arrival - see
// SetAutoRememberCookies/SetAutoRememberAuthHeaders).
func (m *ResponsePane) captureNow() {
	m.captureCookies()
	m.captureAuthHeader()
}

// headerValueCI looks up name in headers case-insensitively -
// http.Request.Headers is a plain map[string]string, not net/http's
// canonicalizing Header type, so a request built with e.g. "authorization"
// (lowercase, as many hand-typed header lines are) would otherwise be
// missed by an exact "Authorization" lookup.
func headerValueCI(headers map[string]string, name string) (string, bool) {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return value, true
		}
	}
	return "", false
}
