package requestpane

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/owenHochwald/Volt/internal/ui"
)

// footerHelpStyle renders this pane's pinned help hint without
// ui.HelpStyle's Margin(1, 2) - that margin bakes in a blank line above
// *and* below the text (2 extra rows just for a hint line), which in this
// pane's already-tight footer budget left barely any height for
// FieldsViewport to show actual fields in.
var footerHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

// errorTextStyle renders a blocked-send reason (see ValidateParams) in the
// footer, distinct from the plain help hint below it.
var errorTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

// fieldsContent renders everything FieldsViewport scrolls: method+URL,
// name, path params, body params, headers, and (in load-test mode) the
// load-test config fields. Pulled out of View() so it can also be called
// from Update() - PageUp/PageDown there must refresh the viewport's
// content before scrolling it, since View() has a value receiver and so
// can only ever mutate its own local, discarded copy of FieldsViewport;
// left solely to View(), the persisted FieldsViewport a later Update()
// scrolls would still have the empty content it was constructed with, so
// maxYOffset() would always read 0 and pgdown would silently do nothing.
func (m RequestPane) fieldsContent() string {
	serverLabel := ui.LabelStyle.Render("Server ")
	serverLine := lipgloss.JoinHorizontal(lipgloss.Left, serverLabel, m.ServerSelector.View())

	methodRendered := m.MethodSelector.GetStyle().Render(m.MethodSelector.Current())
	primaryLine := lipgloss.JoinHorizontal(lipgloss.Left, methodRendered, " ", m.URLInput.View())

	pathParamsBlock := renderParamSection("Path Params", m.PathParams)

	nameLabel := ui.LabelStyle.Render("Name ")
	nameLine := lipgloss.JoinHorizontal(lipgloss.Left, nameLabel, m.NameInput.View())

	headersLabel := ui.LabelStyle.Render("Headers ")
	headersLine := lipgloss.JoinHorizontal(lipgloss.Left, headersLabel, m.Headers.View())

	// An imported request with a known JSON body schema gets structured,
	// individually-fillable fields (like path params) instead of the
	// freeform body textarea.
	var bodyBlock string
	if len(m.BodyParams) > 0 {
		bodyBlock = renderParamSection("Body Params", m.BodyParams)
	} else {
		bodyLabel := ui.LabelStyle.Render("Body    ")
		bodyBlock = lipgloss.JoinHorizontal(lipgloss.Left, bodyLabel, m.Body.View())
	}

	// Field order: server, method+URL, name, path params, body params,
	// headers, then mode-specific content (load-test config fields, or
	// nothing). Each present block is separated from the next by a blank
	// line; pathParamsBlock is skipped entirely (no double blank line) when
	// the URL has no {params}.
	elements := []string{"", serverLine, "", primaryLine, "", nameLine}
	if pathParamsBlock != "" {
		elements = append(elements, "", pathParamsBlock)
	}
	elements = append(elements, "", bodyBlock, "", headersLine)

	if m.LoadTestMode {
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

		elements = append(elements,
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true).Render("Load Test Configuration:"),
			ltConcurrencyLine,
			ltTotalLine,
			ltQPSLine,
			ltTimeoutLine,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, elements...)
}

// View renders the request pane
func (m RequestPane) View() string {
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

	// Render mode-specific help hint
	var helpText string
	if m.LoadTestMode {
		helpText = footerHelpStyle.Render("ctrl+l: exit load test mode • tab/↑/↓: navigate • pgup/pgdn: scroll • ctrl+p: start load test")
	} else {
		helpText = footerHelpStyle.Render("ctrl+l: load test mode • tab/↑/↓: navigate • pgup/pgdn: scroll • ←/→ or h/l: change method • ctrl+p: send • enter/→: accept URL • ctrl+s: save")
	}

	// A blocked send (a body param outside its schema's min/maxLength or
	// minimum/maximum - see ValidateParams) leaves its reasons in
	// ParseErrors; show them so "nothing happened" has a visible cause.
	var errorsBlock string
	if len(m.ParseErrors) > 0 {
		lines := make([]string, len(m.ParseErrors))
		for i, e := range m.ParseErrors {
			lines[i] = errorTextStyle.Render("! " + e)
		}
		errorsBlock = "\n" + lipgloss.JoinVertical(lipgloss.Left, lines...)
	}

	// Size the viewport from the footer's *actual* rendered height (it
	// varies with RequestInProgress/stopwatch/ParseErrors) rather than a
	// guessed constant, so it never over- or under-reserves space - and
	// clip defensively with MaxHeight in case content is ever taller than
	// m.Height regardless (e.g. m.Height not set yet). FieldsViewport can
	// be taller than the pane's box with enough fields (a wide imported
	// body schema, many path params) - unlike a plain lipgloss.Height()
	// (which only pads, never truncates, and would corrupt the rest of
	// the app's layout once content overflowed it), the viewport always
	// clips to exactly its Height and can be scrolled (pgup/pgdown) to
	// reach whatever doesn't fit.
	footer := lipgloss.JoinVertical(lipgloss.Left, "", button, stopwatchCount, errorsBlock, helpText)
	m.FieldsViewport.Height = max(m.Height-lipgloss.Height(footer), 1)
	m.FieldsViewport.SetContent(m.fieldsContent())

	return lipgloss.NewStyle().MaxHeight(max(m.Height, 1)).Render(
		lipgloss.JoinVertical(lipgloss.Left, m.FieldsViewport.View(), footer),
	)
}

var paramSectionTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("140")).Bold(true)

// renderParamSection renders a titled block with one "name [input]" line
// per param - used for both Path Params (from the URL) and Body Params
// (from an imported JSON schema). The title makes the two sections, and
// the plain fields around them, visually distinct from each other.
func renderParamSection(title string, params []Param) string {
	if len(params) == 0 {
		return ""
	}
	lines := make([]string, 0, len(params)+1)
	lines = append(lines, paramSectionTitleStyle.Render(title+":"))
	for _, p := range params {
		text := "  " + p.Name
		if p.Type != "" {
			text += " (" + p.Type + ")"
		}
		label := ui.LabelStyle.Render(text + " ")

		var field, comment string
		if p.Enum != nil {
			// A fixed-choice picker (including a boolean's true/false
			// switch) can't produce an invalid value, so it never needs
			// the red-highlight/comment treatment below.
			field = p.Enum.View()
		} else {
			if msg, ok := ValidateParam(p); !ok {
				p.Input.TextStyle = errorTextStyle
				comment = errorTextStyle.Render("  ! " + msg)
			}
			field = p.Input.View()
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, label, field, comment))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
