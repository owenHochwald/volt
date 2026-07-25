package requestpane

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
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
	m.Request.Body = m.Body.Value()
	m.ParseErrors = headerErrors
}

func (m *RequestPane) validateRequest() error {
	if len(m.ParseErrors) > 0 {
		return fmt.Errorf("invalid headers: %s", strings.Join(m.ParseErrors, "; "))
	}
	if err := m.Request.Validate(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}
	return nil
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

	if len(parseErrors) > 0 {
		return nil, fmt.Errorf("invalid load test configuration: %s", strings.Join(parseErrors, "; "))
	}
	if err := m.validateRequest(); err != nil {
		return nil, err
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
