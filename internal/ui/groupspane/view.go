package groupspane

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	// labelStyle mirrors the removed ui.LabelStyle - main has no
	// equivalent for this package (groupspane doesn't exist there), so it
	// stays a plain local style rather than moving into design.Styles.
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	focusColor   = lipgloss.Color("205")
	unfocusColor = lipgloss.Color("240")
)

// applyFocus mirrors the removed ui.ApplyFocus, same rationale as
// labelStyle above.
func applyFocus(style lipgloss.Style, focus bool) lipgloss.Style {
	if focus {
		return style.BorderForeground(focusColor).Bold(true)
	}
	return style.BorderForeground(unfocusColor)
}

func (m GroupsPane) View() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(m.width).
		Height(m.height)

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Groups & Servers"),
		"",
		m.ViewContent(),
		hintStyle.Render(m.footerHint()),
	)

	return modalStyle.Render(modalContent)
}

// ViewContent renders the pane's browse/form body and status line only,
// without a title, footer hint, or surrounding border - for embedding
// inside another modal's own title/tabs/footer chrome (see shortcutpane's
// Groups surface). View() wraps this same content in its own independent
// modal for when GroupsPane is shown standalone.
func (m GroupsPane) ViewContent() string {
	var body string
	switch m.mode {
	case modeCreateGroup:
		body = m.renderGroupForm("New group")
	case modeRenameGroup:
		body = m.renderGroupForm("Rename group")
	case modeCreateServer:
		body = m.renderServerForm("New server", true)
	case modeRenameServer:
		body = m.renderServerForm("Rename server", false)
	case modeEditServerURL:
		body = m.renderURLForm(fmt.Sprintf("Update base URL for %q", m.renamingName))
	case modeEditServerHeaders:
		body = m.renderHeadersForm(fmt.Sprintf("Default headers for %q", m.renamingName))
	case modeAssignServer:
		body = m.renderAssignServer()
	default:
		body = m.renderBrowse()
	}

	var status string
	if m.pendingDelete != nil {
		kind := "group"
		if m.pendingDelete.kind == pendingDeleteServer {
			kind = "server"
		}
		status = errStyle.Render(fmt.Sprintf("Delete %s %q? y/enter=yes, any other key=no", kind, m.pendingDelete.name))
	} else if m.statusMsg != "" {
		style := okStyle
		if m.statusErr {
			style = errStyle
		}
		status = style.Render(m.statusMsg)
	}

	return lipgloss.JoinVertical(lipgloss.Left, body, "", status)
}

func (m GroupsPane) renderBrowse() string {
	groupsBox := applyFocus(boxStyle, m.target == focusGroups).Render(m.groups.View())
	serversBox := applyFocus(boxStyle, m.target == focusServers).Render(m.servers.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, groupsBox, "  ", serversBox)
}

func (m GroupsPane) renderGroupForm(title string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(title),
		"",
		labelStyle.Render("Name:"),
		m.nameInput.View(),
	)
}

func (m GroupsPane) renderServerForm(title string, showURL bool) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Render(title),
		"",
		labelStyle.Render("Name:"),
		m.nameInput.View(),
	}
	if showURL {
		lines = append(lines, "", labelStyle.Render("Base URL:"), m.urlInput.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m GroupsPane) renderURLForm(title string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(title),
		"",
		labelStyle.Render("Base URL:"),
		m.urlInput.View(),
	)
}

func (m GroupsPane) renderHeadersForm(title string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(title),
		"",
		labelStyle.Render("Headers (key = value, one per line):"),
		m.headersInput.View(),
	)
}

func (m GroupsPane) renderAssignServer() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Assign a server to %q", m.assigningGroup)),
		"",
		m.servers.View(),
	)
}

// FooterHint exposes footerHint for the embedding shortcutpane (see
// shortcutpane/view.go), whose own footer text needs to track whichever
// mode-specific hint applies (browse vs. a create/rename/edit form)
// instead of a single static string.
func (m GroupsPane) FooterHint() string {
	return m.footerHint()
}

func (m GroupsPane) footerHint() string {
	switch m.mode {
	case modeCreateServer:
		return "tab: switch field · enter: save · esc: cancel"
	case modeCreateGroup, modeRenameGroup, modeRenameServer, modeEditServerURL:
		return "enter: save · esc: cancel"
	case modeEditServerHeaders:
		return "ctrl+s: save · esc: cancel"
	case modeAssignServer:
		return "enter: assign · esc: cancel"
	default:
		return "tab: switch list · n: new · r: rename · u: server url · H: server headers · enter: assign server (on a group) · d: delete · q/esc: close"
	}
}
