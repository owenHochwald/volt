package requestpane

import (
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/utils"
)

// syncRequest synchronizes the UI state with the request model
func (m *RequestPane) syncRequest() {
	if m.Request.Headers == nil {
		m.Request.Headers = make(map[string]string)
	}

	m.Request.Method = m.MethodSelector.Current()
	m.Request.URL = m.URLInput.Value()
	m.Request.Name = m.NameInput.Value()

	headerMap, headerErrors := utils.ParseKeyValuePairs(m.Headers.Value())
	m.Request.Headers = headerMap
	m.ParseErrors = headerErrors

	// When BodyParams is populated (an imported request with a known JSON
	// schema), the structured fields drive the body instead - see
	// resolvedRequest - so the freeform Body textarea is hidden and must
	// not overwrite the schema's field set here.
	if len(m.BodyParams) == 0 {
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
// placeholders filled in from PathParams, its body rebuilt from BodyParams
// (when present), and, if the request belongs to a group and stores a
// relative path (e.g. "/pets"), its URL resolved against that group's
// configured server. The stored request itself is left untouched, so
// editing and saving keep operating on the raw URL/body template as saved.
func (m *RequestPane) resolvedRequest() *http.Request {
	resolvedReq := *m.Request
	resolvedReq.URL = SubstitutePathParams(m.Request.URL, m.PathParams)
	if len(m.BodyParams) > 0 {
		resolvedReq.Body = BuildRequestBody(m.BodyParams)
	}

	if m.Request.GroupID != nil && m.DB != nil {
		if resolved, err := m.DB.ResolveEffectiveURL(&resolvedReq); err == nil {
			resolvedReq.URL = resolved
		}
	}

	if resolvedReq.URL == m.Request.URL && resolvedReq.Body == m.Request.Body {
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
		}
	}
}
