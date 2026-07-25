package responsepane

import (
	"fmt"
	"sort"
	"strings"
)

// renderBody renders the response body with appropriate formatting and syntax highlighting
func (m ResponsePane) renderBody() string {
	if m.Response == nil {
		return "No response data"
	}

	if m.Response.Error != "" {
		return m.Response.Error
	}

	contentType := m.Response.ParseContentType()
	return formatContentByType(m.Response.Body, contentType)
}

// renderHeaders renders the response headers in a sorted, styled format
func (m ResponsePane) renderHeaders() string {
	if m.Response == nil || m.Response.Headers == nil {
		return "No headers available"
	}

	var b strings.Builder
	b.WriteString("Response Headers:\n")
	b.WriteString(strings.Repeat("-", len("Response Headers:")+2) + "\n\n")

	// Sort keys for consistent display
	keys := make([]string, 0, len(m.Response.Headers))
	for k := range m.Response.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Render each header with styling
	for _, key := range keys {
		values := m.Response.Headers[key]
		for _, value := range values {
			b.WriteString(m.styles.Text.ResponseKey.Render(key))
			b.WriteString(": ")
			b.WriteString(m.styles.Text.Value.Render(value))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderTiming renders the response timing information
func (m ResponsePane) renderTiming() string {
	if m.Response == nil {
		return "No timing data available"
	}

	var b strings.Builder
	b.WriteString("Request Timing\n")
	b.WriteString(strings.Repeat("─", 60) + "\n\n")

	// Total duration
	b.WriteString(m.styles.Text.ResponseLabel.Render("Total Duration"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(m.Response.Duration.String()))
	b.WriteString("\n\n")

	// Format in milliseconds for readability
	ms := m.Response.Duration.Milliseconds()
	b.WriteString(m.styles.Text.ResponseLabel.Render("Milliseconds"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(fmt.Sprintf("%d ms", ms)))
	b.WriteString("\n\n")

	// Connection type
	b.WriteString(m.styles.Text.ResponseLabel.Render("Connection Type"))
	b.WriteString(": ")
	if m.Response.RoundTrip {
		b.WriteString(m.styles.Text.Value.Render("Round Trip (new connection)"))
	} else {
		b.WriteString(m.styles.Text.Value.Render("Direct (keep-alive)"))
	}
	b.WriteString("\n\n")

	b.WriteString(m.styles.Text.Faint.Render(
		"Note: Detailed timing breakdown (DNS, TLS, TTFB) coming in a future release!",
	))

	return b.String()
}
