package requestpane

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/utils"
)

// contentTypeHeader is the canonical header name BuildRequestBody's JSON
// output is advertised under (see resolvedRequest) - matches the casing
// net/http.Header.Set would itself canonicalize any spelling to once the
// request actually goes out (see client.go's makeCustomRequest).
const contentTypeHeader = "Content-Type"

// hasHeader reports whether headers already has an entry for name,
// compared case-insensitively - HTTP header names aren't case-sensitive
// (a user-typed "content-type" is the same header as "Content-Type" once
// sent - see makeCustomRequest's http.Header.Set, which canonicalizes
// whatever casing it's given), so a plain exact-key map lookup could miss
// a header that's actually already there and add a second, differently
// cased entry alongside it.
func hasHeader(headers map[string]string, name string) bool {
	for k := range headers {
		if strings.EqualFold(k, name) {
			return true
		}
	}
	return false
}

// syncRequest synchronizes the UI state with the request model
func (m *RequestPane) syncRequest() {
	if m.Request.Headers == nil {
		m.Request.Headers = make(map[string]string)
	}

	m.Request.Method = m.MethodSelector.Current()
	m.Request.URL = m.URLInput.Value()
	m.Request.Name = m.NameInput.Value()

	if m.ServerSelector != nil {
		if name := m.ServerSelector.Value(); name != noServerOption {
			m.Request.ServerID = m.serverIDByName(name)
		} else {
			m.Request.ServerID = nil
		}
	}

	headerMap, headerErrors := utils.ParseKeyValuePairs(m.Headers.Value())
	m.Request.Headers = headerMap
	m.ParseErrors = headerErrors

	// When BodyParams or ArrayParams is populated (an imported request with
	// a known JSON schema), the structured fields drive the body instead -
	// see resolvedRequest - so the freeform Body textarea is hidden and
	// must not overwrite the schema's field set here. A body schema with
	// only an array-of-objects field (no scalar properties) has an empty
	// BodyParams but a non-empty ArrayParams - checking BodyParams alone
	// missed that case, silently reparsing the hidden Body textarea's raw
	// schema-template JSON as bogus "key=value" pairs (and overwriting
	// Request.Body with the garbage result) on every send attempt,
	// regardless of what was actually typed into the array's fields.
	if len(m.BodyParams) == 0 && len(m.ArrayParams) == 0 {
		bodyMap, bodyErrors := utils.ParseKeyValuePairs(m.Body.Value())
		jsonData, err := json.Marshal(bodyMap)
		if err != nil {
			m.ParseErrors = append(m.ParseErrors, "JSON marshal error: "+err.Error())
			m.Request.Body = "{}" // Set to valid empty JSON
		} else {
			m.Request.Body = string(jsonData)
			m.ParseErrors = append(m.ParseErrors, bodyErrors...)
		}
	}

	// Unlike path params (which must track live edits to the URL text),
	// body params are derived once when a request is loaded/created (see
	// reinitRequestPane) and then left alone: the body's field set is
	// fixed metadata from the API schema, not something to keep re-guessing
	// from arbitrary text typed elsewhere - re-deriving it here on every
	// tick would risk a hand-typed freeform body flipping into structured
	// fields mid-edit purely because its values happened to go through an
	// empty state while being typed.
	m.syncPathParams()
}

// resolvedRequest returns a copy of m.Request with any {name} path
// placeholders filled in from PathParams, its body rebuilt from
// BodyParams/ArrayParams (when present), and, if the request resolves
// against a server (directly via ServerID, or via its group's server) and
// stores a relative path (e.g. "/pets"), its URL resolved against that
// server's base URL and that server's default headers merged in underneath
// its own. The stored request itself is left untouched, so editing and
// saving keep operating on the raw URL/body/header template as saved.
func (m *RequestPane) resolvedRequest() *http.Request {
	resolvedReq := *m.Request
	resolvedReq.URL = SubstitutePathParams(m.Request.URL, m.PathParams)
	headersChanged := false
	if len(m.BodyParams) > 0 || len(m.ArrayParams) > 0 {
		resolvedReq.Body = BuildRequestBody(m.BodyParams, m.ArrayParams)

		// A schema-driven body (see BuildRequestBody/paramJSONValue) is
		// always JSON, but nothing was actually telling the server that -
		// only if the user hadn't already set their own Content-Type
		// (case-insensitively - see hasHeader), which is left alone as a
		// deliberate override. resolvedReq.Headers still aliases
		// m.Request.Headers at this point (resolvedReq is a shallow copy
		// of *m.Request, and map fields copy by reference) - mutating it
		// in place would corrupt the stored template requests are saved
		// from, so a fresh map is built instead, same as
		// Server.MergeHeaders already does below for the same reason.
		if !hasHeader(resolvedReq.Headers, contentTypeHeader) {
			headers := make(map[string]string, len(resolvedReq.Headers)+1)
			for k, v := range resolvedReq.Headers {
				headers[k] = v
			}
			headers[contentTypeHeader] = "application/json"
			resolvedReq.Headers = headers
			headersChanged = true
		}
	}

	changed := resolvedReq.URL != m.Request.URL || resolvedReq.Body != m.Request.Body || headersChanged

	if m.DB != nil && (m.Request.ServerID != nil || m.Request.GroupID != nil) {
		if server, err := m.DB.ResolveEffectiveServer(&resolvedReq); err == nil && server != nil {
			resolvedReq.URL = resolvedReq.EffectiveURL(server)
			resolvedReq.Headers = server.MergeHeaders(resolvedReq.Headers)
			changed = true
		}
	}

	if !changed {
		return m.Request
	}
	return &resolvedReq
}

// buildJobConfig builds a load test job configuration from current input
func (m *RequestPane) buildJobConfig() (*http.JobConfig, error) {
	var parseErrors []string

	concurrency := 100
	if m.LoadTestConcurrency.Value() != "" {
		n, err := fmt.Sscanf(m.LoadTestConcurrency.Value(), "%d", &concurrency)
		if err != nil || n != 1 || concurrency <= 0 {
			parseErrors = append(parseErrors, "Invalid concurrency (must be positive integer)")
			concurrency = 100
		}
	}

	totalRequests := 10000
	if m.LoadTestTotalReqs.Value() != "" {
		n, err := fmt.Sscanf(m.LoadTestTotalReqs.Value(), "%d", &totalRequests)
		if err != nil || n != 1 || totalRequests <= 0 {
			parseErrors = append(parseErrors, "Invalid total requests (must be positive integer)")
			totalRequests = 10000
		}
	}

	qps := 0.0
	if m.LoadTestQPS.Value() != "" {
		n, err := fmt.Sscanf(m.LoadTestQPS.Value(), "%f", &qps)
		if err != nil || n != 1 || qps < 0 {
			parseErrors = append(parseErrors, "Invalid QPS (must be non-negative number)")
			qps = 0.0
		}
	}

	timeout := 30 * time.Second
	if m.LoadTestTimeout.Value() != "" {
		parsedTimeout, err := time.ParseDuration(m.LoadTestTimeout.Value())
		if err != nil {
			parseErrors = append(parseErrors, "Invalid timeout format (use 30s, 1m, etc.)")
			timeout = 30 * time.Second
		} else if parsedTimeout <= 0 {
			parseErrors = append(parseErrors, "Timeout must be positive")
			timeout = 30 * time.Second
		} else {
			timeout = parsedTimeout
		}
	}

	m.ParseErrors = append(m.ParseErrors, parseErrors...)

	if err := m.Request.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	resolvedReq := m.resolvedRequest()
	if !resolvedReq.IsAbsoluteURL() {
		return nil, fmt.Errorf("url %q belongs to a group with no server configured; set one with `volt group set-server`", resolvedReq.URL)
	}

	return &http.JobConfig{
		Request:       resolvedReq,
		Concurrency:   concurrency,
		TotalRequests: totalRequests,
		QPS:           qps,
		Timeout:       timeout,
		StreamUpdates: true,
	}, nil
}

// sendRequestCmd creates a command to send an HTTP request
func sendRequestCmd(client *http.Client, request *http.Request) tea.Cmd {
	return func() tea.Msg {
		res := make(chan *http.Response)
		go client.Send(request, res)

		responseObject := <-res

		return http.ResultMsg{
			Response: responseObject,
			Request:  request,
		}
	}
}
