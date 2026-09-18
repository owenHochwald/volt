package importpane

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// CloseImportModalMsg signals the parent app to close this modal.
type CloseImportModalMsg struct{}

func closeCmd() tea.Msg { return CloseImportModalMsg{} }

func (m ImportPane) Update(msg tea.Msg) (ImportPane, tea.Cmd) {
	switch msg := msg.(type) {
	case ImportResultMsg:
		m.importing = false
		m.mode = modeResult
		if msg.Err != nil {
			m.statusMsg, m.statusErr = msg.Err.Error(), true
			return m, nil
		}
		m.statusErr = false
		m.statusMsg = summaryText(msg.Summary)
		return m, nil

	case tea.KeyPressMsg:
		if m.importing {
			return m, nil
		}
		if m.mode == modeResult {
			return m.handleResultKey(msg)
		}
		return m.handleFormKey(msg)
	}
	return m, nil
}

func (m ImportPane) handleFormKey(msg tea.KeyPressMsg) (ImportPane, tea.Cmd) {
	if msg.String() == "esc" {
		return m, closeCmd
	}
	const fieldCount = 4 // path, group, base URL, force toggle
	if msg.String() == "tab" || msg.String() == "shift+tab" {
		m.blurAll()
		if msg.String() == "tab" {
			m.focusIndex = (m.focusIndex + 1) % fieldCount
		} else {
			m.focusIndex = (m.focusIndex + fieldCount - 1) % fieldCount
		}
		m.focusCurrent()
		return m, nil
	}
	if msg.String() == "enter" {
		return m.submit()
	}
	if m.focusIndex == 3 && msg.String() == "space" {
		m.force = !m.force
		return m, nil
	}

	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.pathInput, cmd = m.pathInput.Update(msg)
	case 1:
		m.groupInput, cmd = m.groupInput.Update(msg)
	case 2:
		m.baseURLInput, cmd = m.baseURLInput.Update(msg)
	}
	return m, cmd
}

func (m ImportPane) handleResultKey(msg tea.KeyPressMsg) (ImportPane, tea.Cmd) {
	if msg.String() == "esc" {
		return m, closeCmd
	}
	// Any other key returns to the form so the user can fix an error and
	// retry, or import another spec, without leaving the modal.
	m.mode = modeForm
	m.statusMsg = ""
	m.focusIndex = 0
	m.focusCurrent()
	return m, nil
}

func (m ImportPane) submit() (ImportPane, tea.Cmd) {
	path := strings.TrimSpace(m.pathInput.Value())
	if path == "" {
		m.statusMsg, m.statusErr = "spec file path is required", true
		return m, nil
	}
	m.importing = true
	m.statusErr = false
	m.statusMsg = "Importing..."
	group := strings.TrimSpace(m.groupInput.Value())
	baseURL := strings.TrimSpace(m.baseURLInput.Value())
	return m, RunImportCmd(path, baseURL, group, m.force, m.db)
}

func (m *ImportPane) blurAll() {
	m.pathInput.Blur()
	m.groupInput.Blur()
	m.baseURLInput.Blur()
}

func (m *ImportPane) focusCurrent() {
	switch m.focusIndex {
	case 0:
		m.pathInput.Focus()
	case 1:
		m.groupInput.Focus()
	case 2:
		m.baseURLInput.Focus()
	}
}
