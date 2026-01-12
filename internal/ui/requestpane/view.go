package requestpane

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/owenHochwald/Volt/internal/ui"
)

// View renders the request pane
func (m RequestPane) View() string {
	// Render common fields
	methodRendered := m.MethodSelector.GetStyle().Render(m.MethodSelector.Current())
	primaryLine := lipgloss.JoinHorizontal(lipgloss.Left, methodRendered, " ", m.URLInput.View())

	nameLabel := ui.LabelStyle.Render("Name ")
	nameLine := lipgloss.JoinHorizontal(lipgloss.Left, nameLabel, m.NameInput.View())

	headersLabel := ui.LabelStyle.Render("Headers ")
	headersLine := lipgloss.JoinHorizontal(lipgloss.Left, headersLabel, m.Headers.View())

	bodyLabel := ui.LabelStyle.Render("Body    ")
	bodyLine := lipgloss.JoinHorizontal(lipgloss.Left, bodyLabel, m.Body.View())

	// Render button based on state
	var button string
	var stopwatchCount string
	if m.RequestInProgress {
		if m.LoadTestMode {
			button = ui.FocusedButton.Render("Running Load Test...")
		} else {
			button = ui.FocusedButton.Render("Sending...")
			elapsed := m.Stopwatch.Elapsed()
			milliseconds := elapsed.Milliseconds()
			seconds := float64(milliseconds) / 1000.0
			stopwatchCount = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Render(fmt.Sprintf("%.3fs", seconds))
		}
	} else if m.SubmitButton.IsFocused() {
		button = ui.FocusedButton.Render("→ Send")
	} else {
		button = ui.UnfocusedButton.Render("→ Send")
	}

	// Render mode-specific content
	var mainContent string
	var helpText string

	if m.LoadTestMode {
		// Load test mode - add configuration fields
		ltConcurrencyLabel := ui.LabelStyle.Render("Concurrency:    ")
		ltConcurrencyLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltConcurrencyLabel, m.LoadTestConcurrency.View())

		ltTotalLabel := ui.LabelStyle.Render("Total Requests: ")
		ltTotalLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltTotalLabel, m.LoadTestTotalReqs.View())

		ltQPSLabel := ui.LabelStyle.Render("QPS (limit):    ")
		ltQPSLine := lipgloss.JoinHorizontal(lipgloss.Left,
			ltQPSLabel, m.LoadTestQPS.View())

		ltTimeoutLabel := ui.LabelStyle.Render("Timeout:        ")
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
			lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Render("Load Test Configuration:"),
			ltConcurrencyLine,
			ltTotalLine,
			ltQPSLine,
			ltTimeoutLine,
			"",
			button,
		)

		helpText = ui.HelpStyle.Render("ctrl+l: exit load test mode • tab/↑/↓: navigate • ctrl+p: start load test")
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

		helpText = ui.HelpStyle.Render("ctrl+l: load test mode • tab/↑/↓: navigate • ←/→ or h/l: change method • ctrl+p: send • enter/→: accept URL • ctrl+s: save")
	}

	var spacing string
	if m.LoadTestMode {
		spacing = lipgloss.NewStyle().Height(m.Height - 18).Render("")

	} else {
		spacing = lipgloss.NewStyle().Height(m.Height - 10).Render("")

	}

	finalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		spacing,
		stopwatchCount,
		helpText,
	)

	return finalContent
}
