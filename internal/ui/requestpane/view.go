package requestpane

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// View renders the request pane
func (m RequestPane) View() string {
	styles := m.styles

	// Render common fields
	methodRendered := m.MethodSelector.GetStyle().Render(m.MethodSelector.Current())
	primaryLine := lipgloss.JoinHorizontal(lipgloss.Left, methodRendered, " ", m.URLInput.View())

	nameLabel := styles.Text.Label.Render("Name ")
	nameLine := lipgloss.JoinHorizontal(lipgloss.Left, nameLabel, m.NameInput.View())

	headersLabel := styles.Text.Label.Render("Headers ")
	headersLine := lipgloss.JoinHorizontal(lipgloss.Left, headersLabel, m.Headers.View())

	bodyLabel := styles.Text.Label.Render("Body    ")
	bodyLine := lipgloss.JoinHorizontal(lipgloss.Left, bodyLabel, m.Body.View())

	// Render button based on state
	var button string
	var stopwatchCount string
	if m.RequestInProgress {
		if m.LoadTestMode {
			button = styles.Action.Busy.Render("ϟ Running Load Test...")
		} else {
			button = styles.Action.Busy.Render("ϟ Sending...")
			elapsed := m.Stopwatch.Elapsed()
			milliseconds := elapsed.Milliseconds()
			seconds := float64(milliseconds) / 1000.0
			stopwatchCount = styles.Text.Muted.Render(fmt.Sprintf("%.3fs", seconds))
		}
	} else if m.SubmitButton.IsFocused() {
		button = styles.Action.Focused.Render("→ SEND")
	} else {
		button = styles.Action.Primary.Render("→ SEND")
	}

	// Render mode-specific content
	var mainContent string
	if m.LoadTestMode {
		// Load test mode - add configuration fields
		ltConcurrencyLabel := styles.Text.Label.Render("Concurrency:    ")
		ltConcurrencyLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltConcurrencyLabel, m.LoadTestConcurrency.View())

		ltTotalLabel := styles.Text.Label.Render("Total Requests: ")
		ltTotalLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltTotalLabel, m.LoadTestTotalReqs.View())

		ltQPSLabel := styles.Text.Label.Render("QPS (limit):    ")
		ltQPSLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltQPSLabel, m.LoadTestQPS.View())

		ltTimeoutLabel := styles.Text.Label.Render("Timeout:        ")
		ltTimeoutLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltTimeoutLabel, m.LoadTestTimeout.View())

		mainContent = lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			primaryLine,
			nameLine,
			headersLine,
			bodyLine,
			"\n\n",
			styles.Text.Logo.Render("LOAD TEST CONFIGURATION"),
			ltConcurrencyLine,
			ltTotalLine,
			ltQPSLine,
			ltTimeoutLine,
			"",
			button,
		)
	} else {
		// Normal mode
		mainContent = lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			primaryLine,
			nameLine,
			headersLine,
			bodyLine,
			"",
			button,
		)
	}

	usedHeight := lipgloss.Height(mainContent) + lipgloss.Height(stopwatchCount)
	spacing := lipgloss.NewStyle().Height(max(m.Height-usedHeight, 0)).Render("")

	finalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		spacing,
		stopwatchCount,
	)

	return finalContent
}
