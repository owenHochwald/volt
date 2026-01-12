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
	var statusBar string
	if m.Response.Error != "" {
		statusBar = errorStyle.Render("ERROR")
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
