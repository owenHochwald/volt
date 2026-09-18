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
	return formatContentByType(m.Response.Body, contentType, m.effectiveBodyStyle())
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

// renderRequest renders the actual request that was sent - method, URL,
// headers, and body - so what went over the wire (path/body params
// substituted, server headers merged in, etc. - see
// RequestPane.resolvedRequest) stays inspectable independent of whatever's
// since been typed into the request pane above it.
func (m ResponsePane) renderRequest() string {
	if m.SentRequest == nil {
		return "No request sent yet"
	}
	req := m.SentRequest

	var b strings.Builder
	b.WriteString("Sent Request\n")
	b.WriteString(strings.Repeat("─", 60) + "\n\n")

	b.WriteString(m.styles.Text.ResponseLabel.Render("Method"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(req.Method))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Text.ResponseLabel.Render("URL"))
	b.WriteString(": ")
	b.WriteString(m.styles.Text.Value.Render(req.URL))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Text.ResponseLabel.Render("Headers"))
	b.WriteString(":\n")
	if len(req.Headers) == 0 {
		b.WriteString(m.styles.Text.Faint.Render("  (none)"))
		b.WriteString("\n")
	} else {
		keys := make([]string, 0, len(req.Headers))
		for k := range req.Headers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString("  ")
			b.WriteString(m.styles.Text.ResponseKey.Render(k))
			b.WriteString(": ")
			b.WriteString(m.styles.Text.Value.Render(req.Headers[k]))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	b.WriteString(m.styles.Text.ResponseLabel.Render("Body"))
	b.WriteString(":\n")
	if req.Body == "" || req.Body == "{}" {
		b.WriteString(m.styles.Text.Faint.Render("  (empty)"))
	} else {
		b.WriteString(formatContentByType(req.Body, "application/json", m.effectiveBodyStyle()))
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

	// A request that failed before a response came back (e.g. a dial
	// error - "connection refused", a DNS failure, a timeout) never
	// completes a round trip, so Duration stays at its zero value - not a
	// real "0s" measurement. Showing that as if it were a normal
	// (suspiciously instant) timing was misleading; say plainly that
	// there's nothing to measure instead.
	if m.Response.Error != "" {
		b.WriteString(errorTextStyle.Render("Request failed before a response was received - no timing data available."))
		return b.String()
	}

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
