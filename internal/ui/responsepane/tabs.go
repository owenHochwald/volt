package responsepane

import "github.com/charmbracelet/lipgloss"

// Tab indices for normal mode
const (
	TabBody TabIndex = iota
	TabHeaders
	TabTiming
	TabRequest
)

// Tab indices for load test mode
const (
	TabLoadTestOverview TabIndex = iota
	TabLoadTestLatency
	TabLoadTestErrors
)

// TabIndex represents a tab position
type TabIndex int

// renderTabs renders the tab bar for normal response mode
func (m ResponsePane) renderTabs() string {
	tabs := []string{"[1] Body", "[2] Headers", "[3] Timing", "[4] Request"}
	return m.renderTabBar(tabs)
}

// renderLoadTestTabs renders the tab bar for load test mode
func (m ResponsePane) renderLoadTestTabs() string {
	tabs := []string{"[1] Overview", "[2] Latency", "[3] Errors"}
	return m.renderTabBar(tabs)
}

// renderTabBar is a helper that renders a tab bar with active/inactive styling
func (m ResponsePane) renderTabBar(tabs []string) string {
	renderedTabs := make([]string, 0, len(tabs)+1)

	for i, tab := range tabs {
		if i == m.activeTab {
			renderedTabs = append(renderedTabs, activeTab.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, inactiveTab.Render(tab))
		}
	}
	renderedTabs = append(renderedTabs, "\n")

	return lipgloss.JoinHorizontal(lipgloss.Left, renderedTabs...)
}

// getMaxTabs returns the maximum number of tabs based on current mode
func (m ResponsePane) getMaxTabs() int {
	if m.isLoadTest {
		return 3 // Overview, Latency, Errors
	}
	return 4 // Body, Headers, Timing, Request
}
