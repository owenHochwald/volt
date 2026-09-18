package groupspane

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ui"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

func (m GroupsPane) View() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(m.width).
		Height(m.height)

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

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Groups & Servers"),
		"",
		body,
		"",
		status,
		hintStyle.Render(m.footerHint()),
	)

	return modalStyle.Render(modalContent)
}

func (m GroupsPane) renderBrowse() string {
	groupsBox := ui.ApplyFocus(boxStyle, m.target == focusGroups).Render(m.groups.View())
	serversBox := ui.ApplyFocus(boxStyle, m.target == focusServers).Render(m.servers.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, groupsBox, "  ", serversBox)
}

func (m GroupsPane) renderGroupForm(title string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(title),
		"",
		ui.LabelStyle.Render("Name:"),
		m.nameInput.View(),
	)
}

func (m GroupsPane) renderServerForm(title string, showURL bool) string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Render(title),
		"",
		ui.LabelStyle.Render("Name:"),
		m.nameInput.View(),
	}
	if showURL {
		lines = append(lines, "", ui.LabelStyle.Render("Base URL:"), m.urlInput.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m GroupsPane) renderURLForm(title string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(title),
		"",
		ui.LabelStyle.Render("Base URL:"),
		m.urlInput.View(),
	)
}

func (m GroupsPane) renderHeadersForm(title string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render(title),
		"",
		ui.LabelStyle.Render("Headers (key = value, one per line):"),
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
