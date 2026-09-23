package importpane

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/cli"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	// labelStyle mirrors the removed ui.LabelStyle - main has no equivalent
	// for this package (importpane doesn't exist there), so it stays a
	// plain local style rather than moving into design.Styles.
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	// focusedFieldStyle highlights the force toggle when it's the focused
	// field - the toggle has no textinput.Model of its own to lean on for
	// that (unlike pathInput/groupInput/baseURLInput, which show focus via
	// their own cursor), so this fills in the same visual cue directly.
	focusedFieldStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
)

func (m ImportPane) View() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(m.width).
		Height(m.height)

	body := m.renderForm()
	if m.mode == modeResult {
		body = m.renderResult()
	}

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Import OpenAPI Spec"),
		"",
		body,
		"",
		hintStyle.Render(m.footerHint()),
	)

	return modalStyle.Render(modalContent)
}

func (m ImportPane) renderForm() string {
	var status string
	if m.statusMsg != "" {
		style := okStyle
		if m.statusErr {
			style = errStyle
		}
		status = style.Render(m.statusMsg)
	}

	checkbox := "[ ]"
	if m.force {
		checkbox = "[x]"
	}
	forceLine := fmt.Sprintf("%s Overwrite existing requests (same group+method+url) from this spec", checkbox)
	if m.focusIndex == 3 {
		forceLine = focusedFieldStyle.Render(forceLine)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		labelStyle.Render("Spec file path or URL:"),
		m.pathInput.View(),
		"",
		labelStyle.Render("Group (optional - created if missing):"),
		m.groupInput.View(),
		"",
		labelStyle.Render("Base URL (optional - sets the group's/default server):"),
		m.baseURLInput.View(),
		"",
		labelStyle.Render("Force (space to toggle):"),
		forceLine,
		"",
		status,
	)
}

func (m ImportPane) renderResult() string {
	if m.statusErr {
		return errStyle.Render(m.statusMsg)
	}
	// A successful summary already carries its own per-line coloring
	// (warnings highlighted individually), so it's rendered unstyled here.
	return m.statusMsg
}

func (m ImportPane) footerHint() string {
	if m.mode == modeResult {
		return "press any key to import again · esc to close"
	}
	return "tab: next field · space: toggle force · enter: import · esc: close"
}

func summaryText(summary *cli.ImportSummary) string {
	if summary == nil {
		return "Import finished with no summary."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Imported %d request(s), updated %d, skipped %d unchanged.", summary.Imported, summary.Updated, summary.Skipped)
	for _, w := range summary.Warnings {
		b.WriteString("\n")
		b.WriteString(warnStyle.Render("warning: " + w))
	}
	return b.String()
}
