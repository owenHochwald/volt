package responsepane

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/utils"
)

// View renders the response pane - including its own normal Body/etc.
// content while the style picker (see PickBodyStyle) is open, live-
// previewing whatever style the picker's list cursor sits on (see
// effectiveBodyStyle) - the list itself is rendered separately, by
// app/view.go, in the request pane's own box (see StylePickerListView).
func (m ResponsePane) View() string {
	if m.isLoadTest {
		return m.renderLoadTestView()
	}

	// The WebSocket tab (see ws_stream.go) has nothing to do with the last
	// HTTP response - it must still render, tab bar and all, even if no
	// request has ever been sent, as long as a WebSocket send has armed
	// it and data has arrived.
	if m.Response == nil && !m.hasWSTab() {
		return "Make a request to see the response here!"
	}

	if m.savingToFile {
		return m.renderSaveToFilePrompt()
	}

	// trailer (scroll indicator, save/search status, footer hint) is
	// joined on AFTER head's own MaxHeight clip below, not as part of head
	// itself - head's exact rendered height is only ever an estimate
	// (SetHeight's viewport.Height reservation, the tab bar's own internal
	// spacing, etc. all have to line up exactly for it to not drift), so
	// whenever head ran even a line or two over budget, whatever came
	// after it in one single combined block under one MaxHeight was
	// silently what got clipped off the bottom - the scroll indicator
	// ("more below"), then the footer hint (the key-binding reminder) once
	// that overflow itself grew. Both are exactly the lines you need most
	// when the pane is otherwise full of scrollable content, so clipping
	// head alone and unconditionally appending trailer after guarantees
	// they survive regardless of how head's own height estimate drifts.
	head, trailer := m.renderNormalView()
	clippedHead := lipgloss.NewStyle().MaxHeight(max(m.height-lipgloss.Height(trailer), 0)).Render(head)
	return clippedHead + "\n" + trailer
}

// renderSaveToFilePrompt renders the filename entry overlay opened by
// SaveResponseRaw ("s") - mirrors groupspane's create/rename prompt
// (label + textinput + a one-line key hint).
func (m ResponsePane) renderSaveToFilePrompt() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Text.Label.Render("Save raw response body to file:"),
		m.filenameInput.View(),
		"",
		footerHelpStyle.MaxWidth(max(m.width, 1)).Render("enter: save · esc: cancel"),
	)
}

// renderSearchInputLine renders the "/" search prompt inline, in place of
// footerHint, right where the pinned key-binding hint normally sits -
// unlike renderSaveToFilePrompt, this deliberately does NOT take over the
// whole pane, so the response content (and any highlighted matches - see
// highlightMatches) stays visible and scrollable while typing.
func (m ResponsePane) renderSearchInputLine() string {
	return lipgloss.JoinHorizontal(lipgloss.Left, m.styles.Text.Label.Render("/"), m.searchInput.View())
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
	wrapHint := "w: wrap"
	if m.wrapBody {
		wrapHint = "w: unwrap"
	}
	hint := fmt.Sprintf("%s · %s · s: save to file · y: copy · /: search (regex) · gg/G: top/bottom · t: style · c: remember auth · v: view saved · H: history", rawHint, wrapHint)
	if m.searchQuery != "" && len(m.searchMatches) > 0 {
		hint = fmt.Sprintf("/%s (%d/%d) · n/b: next/prev · N/P: next/prev from here · %s", m.searchQuery, m.searchMatchIdx+1, len(m.searchMatches), hint)
	}
	return hint
}

// renderNormalView renders the normal response view's status bar, tabs,
// and content (head) separately from everything pinned below them -
// scroll indicator, save/search status, footer hint (trailer) - kept
// apart so View() can clip head alone rather than the two together (see
// View()'s own comment for why that split matters).
func (m ResponsePane) renderNormalView() (head, trailer string) {
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
	switch {
	case m.diffing:
		statusBar = m.styles.Badge.Info.Render("DIFF") + " " + m.styles.Text.Muted.Render(m.diffLabel)
	case m.Response == nil:
		// Only reachable via the WebSocket tab (see View's own nil-guard,
		// which lets a nil Response through exactly when hasWSTab() is
		// true) - there's no HTTP response to summarize here at all.
		statusBar = m.styles.Text.Muted.Render("No response yet")
	case m.Response.Error != "":
		statusBar = m.styles.Badge.Error.Render("ERROR")
	default:
		statusBar = m.renderHeaderBar()
	}

	tabHeader := m.renderTabs()
	tabContent := m.renderActiveTabContent()
	// Deliberately no blank line between statusBar and tabHeader - an
	// earlier version joined them with a literal "\n" element meant as a
	// single blank-line separator, but lipgloss.JoinVertical splits that
	// element on its own internal newline too, rendering it as *two*
	// blank lines instead of one.
	head = lipgloss.JoinVertical(lipgloss.Left, statusBar, tabHeader, tabContent)

	var trailerLines []string
	if indicator := m.scrollIndicator(); indicator != "" {
		trailerLines = append(trailerLines, indicator)
	}
	if m.saveStatusMsg != "" {
		style := successTextStyle
		if m.saveStatusErr {
			style = errorTextStyle
		}
		trailerLines = append(trailerLines, style.MaxWidth(max(m.width, 1)).Render(m.saveStatusMsg))
	}
	if m.searchStatusMsg != "" {
		trailerLines = append(trailerLines, errorTextStyle.MaxWidth(max(m.width, 1)).Render(m.searchStatusMsg))
	}
	// While the "/" prompt is open (see openSearch), it takes the footer
	// hint's usual spot instead of appearing alongside it - both are
	// one-line, pinned-below-content fixtures, so there's no room (or
	// need) for both at once.
	if m.searching {
		trailerLines = append(trailerLines, m.renderSearchInputLine())
	} else {
		trailerLines = append(trailerLines, footerHelpStyle.MaxWidth(max(m.width, 1)).Render(m.footerHint()))
	}
	trailer = lipgloss.JoinVertical(lipgloss.Left, trailerLines...)
	return head, trailer
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
	stats := m.LoadTestStats
	progress := loadTestProgress(stats)

	if stats.EndTime.IsZero() {
		b.WriteString(m.styles.Badge.Live.Render("● LIVE"))
	} else {
		if stats.FailedRequests > 0 {
			b.WriteString(m.styles.Badge.Error.Render("× COMPLETE"))
		} else {
			b.WriteString(m.styles.Badge.Success.Render("✓ COMPLETE"))
		}
	}
	b.WriteString("  ")
	b.WriteString(m.styles.Metric.Value.Render(fmt.Sprintf(
		"%s / %s",
		formatRequestCount(stats.CompletedRequests),
		formatRequestCount(stats.TotalRequests),
	)))
	b.WriteString(m.styles.Metric.Unit.Render(" requests"))
	b.WriteString("  ")
	b.WriteString(m.styles.Metric.Value.Render(fmt.Sprintf("%.0f%%", progress*100)))
	if stats.FailedRequests > 0 {
		b.WriteString("  ")
		b.WriteString(m.styles.Notice.Error.Render(fmt.Sprintf("%d failed", stats.FailedRequests)))
	}
	b.WriteString("\n")
	b.WriteString(m.renderLoadTestProgress(progress))
	b.WriteString("\n")

	b.WriteString(m.renderLoadTestTabs())

	b.WriteString(m.viewport.View())
	if indicator := m.scrollIndicator(); indicator != "" {
		b.WriteString("\n")
		b.WriteString(indicator)
	}

	return b.String()
}

func (m ResponsePane) renderLoadTestProgress(progress float64) string {
	width := min(max(m.width-2, 12), 60)
	filled := int(progress * float64(width))
	filled = min(max(filled, 0), width)

	var b strings.Builder
	if filled > 0 {
		b.WriteString(m.styles.Chart.Primary.Render(strings.Repeat("━", filled)))
	}
	if filled < width {
		if filled > 0 {
			b.WriteString(m.styles.Chart.Primary.Render("╺"))
			filled++
		}
		b.WriteString(m.styles.Text.Muted.Render(strings.Repeat("━", width-filled)))
	}
	return b.String()
}

// renderHeaderBar renders the status bar for normal responses - shown
// above the tab bar regardless of which tab is active, unlike the
// Timing tab's own "Sent At" (see renderTiming), which only a Timing-tab
// visit would surface. SentAt leads it, before the status code, so when a
// request was sent is visible at a glance without switching tabs.
func (m ResponsePane) renderHeaderBar() string {
	statusStyle := m.styles.Status.ForCode(m.Response.StatusCode)
	status := statusStyle.Render(m.Response.Status)
	duration := fmt.Sprintf(" %d ms", m.Response.Duration.Milliseconds())
	if m.Response.RoundTrip {
		duration += " (round trip)"
	} else {
		duration += " (direct)"
	}
	size := fmt.Sprintf(" %s", utils.FormatSize(len(m.Response.Body)))
	parts := []string{" | "}
	if !m.SentAt.IsZero() {
		parts = append(parts, m.styles.Text.Muted.Render(m.SentAt.Format("15:04:05")), " | ")
	}
	parts = append(parts, status, " | ", duration, " | ", size)
	if indicator := m.jarIndicator(); indicator != "" {
		parts = append(parts, " | ", indicator)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// jarIndicator renders a short hint when something's remembered (see
// internal/authjar) for the current response's host - "" when there's
// nothing to show, so a request with no saved cookies/auth doesn't
// clutter the status bar.
func (m ResponsePane) jarIndicator() string {
	host := m.currentHost()
	if host == "" {
		return ""
	}
	if m.jar.Snapshot(host).IsEmpty() {
		return ""
	}
	return m.styles.Text.Muted.Render("🔑 saved for " + host)
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
