package requestpane

import (
	"regexp"
	"strings"
)

var pathParamPattern = regexp.MustCompile(`\{([^{}]+)\}`)

// extractPathParamNames returns the distinct {name} placeholders in url
// (e.g. "/cluster/{cluster_id}/repository" -> ["cluster_id"]), in the
// order they first appear.
func extractPathParamNames(url string) []string {
	matches := pathParamPattern.FindAllStringSubmatch(url, -1)
	if len(matches) == 0 {
		return nil
	}
	names := make([]string, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, match := range matches {
		name := match[1]
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

// SubstitutePathParams replaces each {name} placeholder in url with the
// corresponding param's current value. A placeholder with no matching
// param (shouldn't normally happen, since params are derived from the URL
// itself) is left as-is.
func SubstitutePathParams(url string, params []Param) string {
	for _, p := range params {
		url = strings.ReplaceAll(url, "{"+p.Name+"}", p.Input.Value())
	}
	return url
}

// syncPathParams re-derives m.PathParams from the request's current URL,
// preserving already-typed values for params that are still present. It's
// a no-op if the set of param names hasn't changed, and otherwise rebuilds
// the focus cycle so the (possibly new) fields are reachable via tab.
func (m *RequestPane) syncPathParams() {
	names := extractPathParamNames(m.Request.URL)
	if paramNamesMatch(names, m.PathParams) {
		return
	}
	m.PathParams = rebuildParams(m.PathParams, names)
	m.rebuildFocusManager()
}
