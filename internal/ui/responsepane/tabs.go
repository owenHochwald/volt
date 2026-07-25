package responsepane

import "charm.land/lipgloss/v2"

// Tab indices for normal mode
const (
	TabBody TabIndex = iota
	TabHeaders
	TabTiming
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
	tabs := labelsForKeys(m.keys.DirectTab.Keys(), []string{"Body", "Headers", "Timing"})
	return m.renderTabBar(tabs)
}

// renderLoadTestTabs renders the tab bar for load test mode
func (m ResponsePane) renderLoadTestTabs() string {
	tabs := labelsForKeys(m.keys.DirectTab.Keys(), []string{"Overview", "Latency", "Errors"})
	return m.renderTabBar(tabs)
}

func labelsForKeys(keys, labels []string) []string {
	tabs := make([]string, 0, len(labels))
	for i, label := range labels {
		keyLabel := ""
		if i < len(keys) {
			keyLabel = "[" + keys[i] + "] "
		}
		tabs = append(tabs, keyLabel+label)
	}
	return tabs
}

// renderTabBar is a helper that renders a tab bar with active/inactive styling
func (m ResponsePane) renderTabBar(tabs []string) string {
	renderedTabs := make([]string, 0, len(tabs)+1)

	for i, tab := range tabs {
		if i == m.activeTab {
			renderedTabs = append(renderedTabs, m.styles.Tabs.Active.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, m.styles.Tabs.Inactive.Render(tab))
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
	return 3 // Body, Headers, Timing
}
