package responsepane

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/owenHochwald/Volt/internal/difftext"
	"github.com/owenHochwald/Volt/internal/history"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

// IsDiffing reports whether diff mode (see startDiff, "x" from within the
// history viewer) is currently showing a diff instead of the active tab's
// normal content - checked by app/update.go for the same reason as
// IsViewingJar/IsViewingHistory: so esc reaches this pane's own Update (see
// update.go's diffing check) and exits diff mode instead of the
// panel-switch/quit-arm swallowing it first.
func (m *ResponsePane) IsDiffing() bool {
	return m.diffing
}

// startDiff enters diff mode, comparing other (a past entry, selected from
// the history viewer) against whatever's currently displayed. Unlike
// viewHistoryEntry, this never replaces m.Response/m.SentRequest - the
// current response stays what's actually displayed once diff mode is
// exited (see exitDiff), it's only used here to build the comparison.
func (m *ResponsePane) startDiff(other history.Entry) {
	if m.Response == nil {
		return
	}
	current := history.Entry{
		Response:    m.Response,
		SentRequest: m.SentRequest,
		SentAt:      time.Now(),
	}
	m.diffing = true
	m.viewingHistory = false
	m.diffLabel = fmt.Sprintf("%s vs current", other.SentAt.Format("15:04:05"))
	m.diffContent = renderResponseDiff(other.Response, current.Response, m.styles)
	m.updateViewportForActiveTab()
}

// exitDiff leaves diff mode, restoring whatever the active tab would
// normally show.
func (m *ResponsePane) exitDiff() {
	m.diffing = false
	m.updateViewportForActiveTab()
}

// responseDocument renders a response as flat lines suitable for a
// line-based diff (see difftext.Lines) - status, duration, headers (sorted
// by name, one line per value), then the body. The body is pretty-printed
// with formatJSON, deliberately the plain (non-chroma) formatter - ANSI
// escape codes from syntax highlighting would make otherwise-identical
// lines compare unequal.
func responseDocument(r *http.Response) []string {
	if r == nil {
		return []string{"(no response)"}
	}
	if r.Error != "" {
		return []string{"Error: " + r.Error}
	}

	lines := []string{
		"Status: " + r.Status,
		"Duration: " + r.Duration.String(),
		"",
		"Headers:",
	}
	names := make([]string, 0, len(r.Headers))
	for name := range r.Headers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		for _, value := range r.Headers[name] {
			lines = append(lines, fmt.Sprintf("  %s: %s", name, value))
		}
	}

	lines = append(lines, "", "Body:")
	lines = append(lines, strings.Split(formatJSON(r.Body), "\n")...)
	return lines
}

// renderResponseDiff renders a's document against b's as colored +/-/
// unchanged lines - the same visual language as a "diff --git" hunk, minus
// line numbers, which don't mean much for two independently fetched
// responses.
func renderResponseDiff(a, b *http.Response, styles design.Styles) string {
	diffLines := difftext.Lines(responseDocument(a), responseDocument(b))
	rendered := make([]string, 0, len(diffLines))
	for _, l := range diffLines {
		switch l.Op {
		case difftext.Insert:
			rendered = append(rendered, styles.Notice.Success.Render("+ "+l.Text))
		case difftext.Delete:
			rendered = append(rendered, styles.Notice.Error.Render("- "+l.Text))
		default:
			rendered = append(rendered, styles.Text.Muted.Render("  "+l.Text))
		}
	}
	return strings.Join(rendered, "\n")
}
