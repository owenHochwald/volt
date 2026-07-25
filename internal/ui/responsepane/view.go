package responsepane

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
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
	var statusBar string
	if m.Response.Error != "" {
		statusBar = m.styles.Badge.Error.Render("ERROR")
		m.viewport.SetContent(m.Response.Error)
	} else {
		statusBar = m.renderHeaderBar()
	}

	tabHeader := m.renderTabs()
	tabContent := m.renderActiveTabContent()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		statusBar,
		"\n",
		tabHeader,
		tabContent,
	)
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

// renderHeaderBar renders the status bar for normal responses
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
	return lipgloss.JoinHorizontal(lipgloss.Left, " | ", status, " | ", duration, " | ", size)
}

// renderActiveTabContent renders the content for the currently active tab
func (m ResponsePane) renderActiveTabContent() string {
	if m.isLoadTest {
		// Load test mode shouldn't call this, but handle gracefully
		return m.viewport.View()
	}

	switch TabIndex(m.activeTab) {
	case TabBody:
		return m.viewport.View()
	case TabHeaders:
		return m.renderHeaders()
	case TabTiming:
		return m.renderTiming()
	default:
		return "Something went wrong."
	}
}
