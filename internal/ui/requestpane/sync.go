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
	bodyMap, bodyErrors := utils.ParseKeyValuePairs(m.Body.Value())

	jsonData, err := json.Marshal(bodyMap)
	if err != nil {
		m.ParseErrors = append(m.ParseErrors, "JSON marshal error: "+err.Error())
		m.Request.Headers = headerMap
		m.Request.Body = "{}" // Set to valid empty JSON
		m.ParseErrors = append(m.ParseErrors, headerErrors...)
		return
	}

	m.Request.Headers = headerMap
	m.Request.Body = string(jsonData)
	m.ParseErrors = append(headerErrors, bodyErrors...)
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

	return &http.JobConfig{
		Request:       m.Request,
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
