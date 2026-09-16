package responsepane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/owenHochwald/Volt/internal/utils"
)

// View renders the response pane
func (m ResponsePane) View() string {
	if m.isLoadTest {
		return m.renderLoadTestView()
	}

	if m.Response == nil {
		return "Make a request to see the response here!"
	}

	return m.renderNormalView()
}

// renderNormalView renders the normal response view with status bar, tabs, and content
func (m ResponsePane) renderNormalView() string {
	// The viewport's content is populated per-tab by
	// updateViewportForActiveTab (called from SetResponse and on every tab
	// switch) - it already puts the raw error message in the Body tab and
	// leaves Headers/Timing/Request showing their own content. Setting it
	// again here unconditionally used to stomp that on *every* render
	// frame regardless of which tab was active, so a connection error (or
	// any other Response.Error) made every tab - including Request - show
	// nothing but the raw error text, with no way to see what was actually
	// sent.
	var statusBar string
	if m.Response.Error != "" {
		statusBar = errorStyle.Render("ERROR")
	} else {
		statusBar = m.renderHeaderBar()
	}

	tabHeader := m.renderTabs()
	tabContent := m.renderActiveTabContent()

	elements := []string{statusBar, "\n", tabHeader, tabContent}
	if indicator := m.scrollIndicator(); indicator != "" {
		elements = append(elements, indicator)
	}
	return lipgloss.JoinVertical(lipgloss.Left, elements...)
}

// scrollIndicator renders a hint below the tab content when there's more to
// see than fits in the viewport - without this, a large response body,
// header set, or "sent request" dump that overflowed the box gave no visual
// cue that pgup/pgdn would reveal anything more, only a viewport clipped to
// its Height with nothing below it. Returns "" when everything already
// fits, so it doesn't clutter short responses.
func (m ResponsePane) scrollIndicator() string {
	if m.viewport.TotalLineCount() <= m.viewport.VisibleLineCount() {
		return ""
	}
	var parts []string
	if !m.viewport.AtTop() {
		parts = append(parts, "↑ more above")
	}
	if !m.viewport.AtBottom() {
		parts = append(parts, "↓ more below")
	}
	if len(parts) == 0 {
		return ""
	}
	return scrollIndicatorStyle.Render(strings.Join(parts, "   ") + fmt.Sprintf("  (%.0f%%)", m.viewport.ScrollPercent()*100))
}

// renderLoadTestView renders the load test view with status, tabs, and stats
func (m ResponsePane) renderLoadTestView() string {
	if m.LoadTestStats == nil {
		return "No load test data"
	}

	var b strings.Builder

	// Status line
	status := "Load Test "
	if m.LoadTestStats.EndTime.IsZero() {
		status += "In Progress..."
	} else {
		status += "Complete"
	}
	statusBar := loadTestStatusStyle.Render(status)
	b.WriteString(statusBar)
	b.WriteString("\n")

	// Tab header
	tabHeader := m.renderLoadTestTabs()
	b.WriteString(tabHeader)

	// Tab content
	b.WriteString(m.viewport.View())
	if indicator := m.scrollIndicator(); indicator != "" {
		b.WriteString("\n")
		b.WriteString(indicator)
	}

	return b.String()
}

// renderHeaderBar renders the status bar for normal responses
func (m ResponsePane) renderHeaderBar() string {
	statusStyle := utils.MapStatusCodeToColor(m.Response.StatusCode)
	status := statusStyle.Render(m.Response.Status)
	duration := fmt.Sprintf(" %d ms", m.Response.Duration.Milliseconds())
	if m.Response.RoundTrip {
		duration += " (round trip)"
	} else {
		duration += " (direct)"
	}
	size := fmt.Sprintf(" %s", utils.FormatSize(len(m.Response.Body)))
	return lipgloss.JoinHorizontal(lipgloss.Left, " | ", status, " | ", duration, " | ", size)
}

// renderActiveTabContent renders the content for the currently active tab.
// Every tab (Body/Headers/Timing/Request) goes through the same scrollable
// viewport rather than returning its own render function's output directly
// - updateViewportForActiveTab (called on every tab switch and whenever a
// new Response arrives) has already populated it with the right content for
// whichever tab is current. Previously only Body did this; Headers/Timing/
// Request just rendered themselves inline with no clipping or scrolling at
// all, so a response with enough headers (or a large "sent request" body)
// overflowed the box instead of being scrollable - the box would grow (or
// content would be cut off with no indication more existed) instead of
// staying a fixed size with pgup/pgdn to reach the rest.
func (m ResponsePane) renderActiveTabContent() string {
	return m.viewport.View()
}
