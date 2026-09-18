package groupspane

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// boxStyle is the bare (unfocused-color-agnostic) frame around each of the
// browse view's two lists - applyFocus below fills in the actual border
// color from m.styles, matching whichever list currently has focus.
var boxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)

// applyFocus colors box's border using the same focused/unfocused panel
// colors every other pane's own boxes use (see design.Styles.Panel), so
// the Groups surface reads as part of the same app instead of a visually
// distinct modal bolted on the side.
func applyFocus(box, panelFocused, panelBase lipgloss.Style, focus bool) lipgloss.Style {
	if focus {
		return box.BorderForeground(panelFocused.GetBorderLeftForeground()).Bold(true)
	}
	return box.BorderForeground(panelBase.GetBorderLeftForeground())
}

func (m GroupsPane) View() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Panel.Focused.GetBorderLeftForeground()).
		Padding(1, 2).
		Width(m.width).
		Height(m.height)

	modalContent := lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Text.Logo.Render("Groups & Servers"),
		"",
		m.ViewContent(),
		m.styles.Text.Muted.Render(m.footerHint()),
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
	default:
		body = m.renderBrowse()
	}

	var status string
	if m.pendingDelete != nil {
		kind := "group"
		if m.pendingDelete.kind == pendingDeleteServer {
			kind = "server"
		}
		status = m.styles.Notice.Error.Render(fmt.Sprintf("Delete %s %q? y/enter=yes, any other key=no", kind, m.pendingDelete.name))
	} else if m.statusMsg != "" {
		style := m.styles.Notice.Success
		if m.statusErr {
			style = m.styles.Notice.Error
		}
		status = style.Render(m.statusMsg)
	}

	return lipgloss.JoinVertical(lipgloss.Left, body, "", status)
}

func (m GroupsPane) renderBrowse() string {
	groupsBox := applyFocus(boxStyle, m.styles.Panel.Focused, m.styles.Panel.Base, m.target == focusGroups).Render(m.groups.View())
	serversBox := applyFocus(boxStyle, m.styles.Panel.Focused, m.styles.Panel.Base, m.target == focusServers).Render(m.servers.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, groupsBox, "  ", serversBox)
}

func (m GroupsPane) renderGroupForm(title string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Text.Logo.Render(title),
		"",
		m.styles.Text.Label.Render("Name:"),
		m.nameInput.View(),
	)
}

func (m GroupsPane) renderServerForm(title string, showURL bool) string {
	lines := []string{
		m.styles.Text.Logo.Render(title),
		"",
		m.styles.Text.Label.Render("Name:"),
		m.nameInput.View(),
	}
	if showURL {
		lines = append(lines, "", m.styles.Text.Label.Render("Base URL:"), m.urlInput.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m GroupsPane) renderURLForm(title string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Text.Logo.Render(title),
		"",
		m.styles.Text.Label.Render("Base URL:"),
		m.urlInput.View(),
	)
}

func (m GroupsPane) renderHeadersForm(title string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Text.Logo.Render(title),
		"",
		m.styles.Text.Label.Render("Headers (key = value, one per line):"),
		m.headersInput.View(),
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
	default:
		return "tab: switch list · n: new · r: rename · u: server url · H: server headers · d: delete · q/esc: close"
	}
}
