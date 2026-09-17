package responsepane

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/owenHochwald/Volt/internal/ui"
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

	if m.savingToFile {
		return m.renderSaveToFilePrompt()
	}

	if m.searching {
		return m.renderSearchPrompt()
	}

	// footer is joined on AFTER body's own MaxHeight clip below, not as
	// one of body's own elements - body's exact rendered height is only
	// ever an estimate (SetHeight's viewport.Height reservation, the tab
	// bar's own internal spacing, etc. all have to line up exactly for it
	// to not drift), so when body ran even one line over budget, this
	// footer - the key-binding hint, i.e. the single most useful line when
	// the pane is otherwise full of scrollable content - was silently the
	// one that got clipped off, being the last line of one single combined
	// block under one MaxHeight. Clipping body alone and unconditionally
	// appending footer after guarantees it survives regardless of how
	// body's own height estimate drifts.
	body, footer := m.renderNormalView()
	clippedBody := lipgloss.NewStyle().MaxHeight(max(m.height-1, 0)).Render(body)
	return clippedBody + "\n" + footer
}

// renderSaveToFilePrompt renders the filename entry overlay opened by
// SaveResponseRaw ("s") - mirrors groupspane's create/rename prompt
// (label + textinput + a one-line key hint).
func (m ResponsePane) renderSaveToFilePrompt() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		ui.LabelStyle.Render("Save raw response body to file:"),
		m.filenameInput.View(),
		"",
		footerHelpStyle.MaxWidth(max(m.width, 1)).Render("enter: save · esc: cancel"),
	)
}

// renderSearchPrompt renders the search entry overlay opened by
// SearchResponse ("/") - mirrors renderSaveToFilePrompt above.
func (m ResponsePane) renderSearchPrompt() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		ui.LabelStyle.Render("Search:"),
		m.searchInput.View(),
		"",
		footerHelpStyle.MaxWidth(max(m.width, 1)).Render("enter: search · esc: cancel"),
	)
}

// footerHint renders this pane's pinned one-line key hint below the
// response content - reflects ToggleRawBody's current state (rather than
// a fixed string) so it always says what pressing "r" would actually do
// next, and, whenever there's an active search (see search.go), leads
// with its query and position among its matches.
func (m ResponsePane) footerHint() string {
	rawHint := "r: raw view"
	if m.rawBody {
		rawHint = "r: pretty view"
	}
	hint := fmt.Sprintf("%s · s: save to file · y: copy · /: search · gg/G: top/bottom", rawHint)
	if m.searchQuery != "" && len(m.searchMatches) > 0 {
		hint = fmt.Sprintf("/%s (%d/%d) · n: next · b: prev · %s", m.searchQuery, m.searchMatchIdx+1, len(m.searchMatches), hint)
	}
	return hint
}

// renderNormalView renders the normal response view's status bar, tabs,
// and content (body) separately from its pinned footer hint (footer) -
// kept apart so View() can clip body alone rather than the two together
// (see View()'s own comment for why that split matters).
func (m ResponsePane) renderNormalView() (body, footer string) {
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
	if m.saveStatusMsg != "" {
		style := successTextStyle
		if m.saveStatusErr {
			style = errorTextStyle
		}
		elements = append(elements, style.MaxWidth(max(m.width, 1)).Render(m.saveStatusMsg))
	}
	if m.searchStatusMsg != "" {
		elements = append(elements, errorTextStyle.MaxWidth(max(m.width, 1)).Render(m.searchStatusMsg))
	}
	body = lipgloss.JoinVertical(lipgloss.Left, elements...)
	footer = footerHelpStyle.MaxWidth(max(m.width, 1)).Render(m.footerHint())
	return body, footer
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
