package groupspane

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/utils"
)

// CloseGroupsModalMsg signals the parent app to close this modal.
type CloseGroupsModalMsg struct{}

func closeCmd() tea.Msg { return CloseGroupsModalMsg{} }

func (m GroupsPane) Update(msg tea.Msg) (GroupsPane, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch m.mode {
	case modeCreateGroup, modeRenameGroup:
		return m.handleGroupFormKey(keyMsg)
	case modeCreateServer, modeRenameServer, modeEditServerURL:
		return m.handleServerFormKey(keyMsg)
	case modeEditServerHeaders:
		return m.handleServerHeadersFormKey(keyMsg)
	case modeEditServerVariables:
		return m.handleServerVariablesFormKey(keyMsg)
	default:
		return m.handleBrowseKey(keyMsg)
	}
}

func (m GroupsPane) handleBrowseKey(msg tea.KeyPressMsg) (GroupsPane, tea.Cmd) {
	// A pending delete swallows the very next key as its y/n answer, checked
	// before anything else - while confirming, that's the only thing any key
	// should mean.
	if m.pendingDelete != nil {
		pending := m.pendingDelete
		m.pendingDelete = nil
		if msg.String() != "y" && msg.String() != "Y" && msg.String() != "enter" {
			return m, nil
		}
		switch pending.kind {
		case pendingDeleteGroup:
			deletedRequests, err := m.db.DeleteGroup(pending.name)
			successMsg := fmt.Sprintf("Deleted group %q", pending.name)
			if deletedRequests > 0 {
				successMsg = fmt.Sprintf("Deleted group %q and %d request(s) in it", pending.name, deletedRequests)
			}
			m.setStatus(err, successMsg)
		case pendingDeleteServer:
			m.setStatus(m.db.DeleteServer(pending.name), fmt.Sprintf("Deleted server %q", pending.name))
		}
		m.reload()
		return m, nil
	}

	if keybindings.Matches(msg, m.keys.CloseGroups) {
		return m, closeCmd
	}
	// Plain "tab" (not h/l - see footerHint below) switches between the
	// Groups and Servers lists: h/l are reserved by the embedding
	// shortcutpane for switching between its Help/Settings/Groups surfaces
	// (see shortcutpane/update.go), so this pane can't also claim them for
	// its own internal list switch without colliding.
	if msg.String() == "tab" {
		if m.target == focusGroups {
			m.target = focusServers
		} else {
			m.target = focusGroups
		}
		return m, nil
	}
	if keybindings.Matches(msg, m.keys.CreateItem) {
		if m.target == focusGroups {
			m.statusMsg = ""
			m.nameInput.SetValue("")
			m.nameInput.Focus()
			m.mode = modeCreateGroup
			return m, nil
		}
		// A new server is always connected to whichever group is
		// currently selected (see handleServerFormKey's modeCreateServer
		// case) - there's no server that exists outside of some group's
		// scope from this UI, so creating one needs a group to attach it
		// to first.
		if _, ok := m.selectedGroup(); !ok {
			m.statusMsg, m.statusErr = "create a group first", true
			return m, nil
		}
		m.statusMsg = ""
		m.nameInput.SetValue("")
		m.nameInput.Focus()
		m.urlInput.SetValue("")
		m.formFocusURL = false
		m.mode = modeCreateServer
		return m, nil
	}
	if keybindings.Matches(msg, m.keys.RenameItem) {
		if m.target == focusGroups {
			item, ok := m.selectedGroup()
			if !ok {
				return m, nil
			}
			m.renamingName = item.group.Name
			m.nameInput.SetValue(item.group.Name)
			m.nameInput.Focus()
			m.mode = modeRenameGroup
		} else {
			item, ok := m.selectedServer()
			if !ok {
				return m, nil
			}
			m.renamingName = item.server.Name
			m.nameInput.SetValue(item.server.Name)
			m.nameInput.Focus()
			m.mode = modeRenameServer
		}
		return m, nil
	}
	if msg.String() == "u" && m.target == focusServers {
		item, ok := m.selectedServer()
		if !ok {
			return m, nil
		}
		m.renamingName = item.server.Name
		m.urlInput.SetValue(item.server.BaseURL)
		m.urlInput.Focus()
		m.mode = modeEditServerURL
		return m, nil
	}
	if msg.String() == "H" && m.target == focusServers {
		item, ok := m.selectedServer()
		if !ok {
			return m, nil
		}
		m.renamingName = item.server.Name
		m.headersInput.SetValue(utils.ParseMapToString(item.server.Headers))
		m.headersInput.Focus()
		m.mode = modeEditServerHeaders
		return m, nil
	}
	if msg.String() == "V" && m.target == focusServers {
		item, ok := m.selectedServer()
		if !ok {
			return m, nil
		}
		m.renamingName = item.server.Name
		m.variablesInput.SetValue(utils.ParseMapToString(item.server.Variables))
		m.variablesInput.Focus()
		m.mode = modeEditServerVariables
		return m, nil
	}
	if keybindings.Matches(msg, m.keys.DeleteItem) {
		// Deleting is irreversible (deleting a group cascades to its
		// requests) - arm a confirmation instead of deleting on the spot
		// (see pendingDelete above).
		if m.target == focusGroups {
			item, ok := m.selectedGroup()
			if !ok {
				return m, nil
			}
			m.pendingDelete = &pendingDeleteItem{kind: pendingDeleteGroup, name: item.group.Name}
		} else {
			item, ok := m.selectedServer()
			if !ok {
				return m, nil
			}
			m.pendingDelete = &pendingDeleteItem{kind: pendingDeleteServer, name: item.server.Name}
		}
		return m, nil
	}
	var cmd tea.Cmd
	if m.target == focusGroups {
		m.groups, cmd = m.groups.Update(msg)
		// The Servers list only ever shows the selected group's own
		// servers (see refreshGroupServers) - a keypress that moved the
		// Groups cursor (up/down, filtering, ...) needs it re-derived for
		// whichever group is selected now.
		m.refreshGroupServers()
	} else {
		m.servers, cmd = m.servers.Update(msg)
	}
	return m, cmd
}

func (m GroupsPane) handleGroupFormKey(msg tea.KeyPressMsg) (GroupsPane, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = modeBrowse
		m.nameInput.Blur()
		return m, nil
	}
	if msg.String() == "enter" {
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			m.statusMsg, m.statusErr = "name cannot be empty", true
			return m, nil
		}

		var err error
		if m.mode == modeCreateGroup {
			_, err = m.db.GetOrCreateGroup(name)
		} else {
			err = m.db.RenameGroup(m.renamingName, name)
		}
		m.setStatus(err, fmt.Sprintf("Saved group %q", name))
		if err == nil {
			m.mode = modeBrowse
			m.nameInput.Blur()
			m.reload()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m GroupsPane) handleServerFormKey(msg tea.KeyPressMsg) (GroupsPane, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = modeBrowse
		m.nameInput.Blur()
		m.urlInput.Blur()
		return m, nil
	}
	if m.mode == modeCreateServer && msg.String() == "tab" {
		m.formFocusURL = !m.formFocusURL
		if m.formFocusURL {
			m.nameInput.Blur()
			m.urlInput.Focus()
		} else {
			m.urlInput.Blur()
			m.nameInput.Focus()
		}
		return m, nil
	}
	if msg.String() == "enter" {
		var err error
		var successMsg string

		switch m.mode {
		case modeCreateServer:
			name := strings.TrimSpace(m.nameInput.Value())
			url := strings.TrimSpace(m.urlInput.Value())
			if name == "" || url == "" {
				m.statusMsg, m.statusErr = "name and base URL are both required", true
				return m, nil
			}
			group, ok := m.selectedGroup()
			if !ok {
				m.statusMsg, m.statusErr = "create a group first", true
				return m, nil
			}
			if _, err = m.db.UpsertServer(name, url); err == nil {
				err = m.db.AddGroupServer(group.group.Name, name)
			}
			successMsg = fmt.Sprintf("Saved server %q for group %q", name, group.group.Name)
		case modeRenameServer:
			name := strings.TrimSpace(m.nameInput.Value())
			if name == "" {
				m.statusMsg, m.statusErr = "name cannot be empty", true
				return m, nil
			}
			err = m.db.RenameServer(m.renamingName, name)
			successMsg = fmt.Sprintf("Renamed server to %q", name)
		case modeEditServerURL:
			url := strings.TrimSpace(m.urlInput.Value())
			if url == "" {
				m.statusMsg, m.statusErr = "base URL cannot be empty", true
				return m, nil
			}
			err = m.db.UpdateServerURL(m.renamingName, url)
			successMsg = fmt.Sprintf("Updated %q's base URL", m.renamingName)
		}

		m.setStatus(err, successMsg)
		if err == nil {
			m.mode = modeBrowse
			m.nameInput.Blur()
			m.urlInput.Blur()
			m.reload()
		}
		return m, nil
	}

	var cmd tea.Cmd
	if m.mode == modeEditServerURL || m.formFocusURL {
		m.urlInput, cmd = m.urlInput.Update(msg)
	} else {
		m.nameInput, cmd = m.nameInput.Update(msg)
	}
	return m, cmd
}

// handleServerHeadersFormKey handles editing a server's default headers -
// the ones merged into every request that resolves against it (directly, or
// via a group). Unlike the single-line name/URL forms, this is a textarea:
// enter inserts a newline instead of submitting, so ctrl+s (the same save
// key the request pane's own Headers field lives next to) submits instead.
func (m GroupsPane) handleServerHeadersFormKey(msg tea.KeyPressMsg) (GroupsPane, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = modeBrowse
		m.headersInput.Blur()
		return m, nil
	}
	if keybindings.Matches(msg, m.keys.SaveRequest) {
		headers, parseErrors := utils.ParseKeyValuePairs(m.headersInput.Value())
		if len(parseErrors) > 0 {
			m.statusMsg, m.statusErr = strings.Join(parseErrors, "; "), true
			return m, nil
		}
		err := m.db.UpdateServerHeaders(m.renamingName, headers)
		m.setStatus(err, fmt.Sprintf("Updated %q's default headers", m.renamingName))
		if err == nil {
			m.mode = modeBrowse
			m.headersInput.Blur()
			m.reload()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.headersInput, cmd = m.headersInput.Update(msg)
	return m, cmd
}

// handleServerVariablesFormKey mirrors handleServerHeadersFormKey exactly,
// for a server's {{name}} placeholder values (see
// http.SubstituteVariables) instead of its default headers.
func (m GroupsPane) handleServerVariablesFormKey(msg tea.KeyPressMsg) (GroupsPane, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = modeBrowse
		m.variablesInput.Blur()
		return m, nil
	}
	if keybindings.Matches(msg, m.keys.SaveRequest) {
		variables, parseErrors := utils.ParseKeyValuePairs(m.variablesInput.Value())
		if len(parseErrors) > 0 {
			m.statusMsg, m.statusErr = strings.Join(parseErrors, "; "), true
			return m, nil
		}
		err := m.db.UpdateServerVariables(m.renamingName, variables)
		m.setStatus(err, fmt.Sprintf("Updated %q's variables", m.renamingName))
		if err == nil {
			m.mode = modeBrowse
			m.variablesInput.Blur()
			m.reload()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.variablesInput, cmd = m.variablesInput.Update(msg)
	return m, cmd
}

func (m *GroupsPane) setStatus(err error, successMsg string) {
	if err != nil {
		m.statusMsg, m.statusErr = err.Error(), true
		return
	}
	m.statusMsg, m.statusErr = successMsg, false
}
