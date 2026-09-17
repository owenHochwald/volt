package responsepane

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// SaveResponseBodyMsg reports the outcome of saveResponseBodyCmd - a
// background file write (see handleSaveToFileKey), so ResponsePane can
// show a success/failure status line once it completes.
type SaveResponseBodyMsg struct {
	Path string
	Err  error
}

// Update handles Bubble Tea messages and state transitions
func (m *ResponsePane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case SaveResponseBodyMsg:
		if msg.Err != nil {
			m.saveStatusMsg = fmt.Sprintf("failed to save: %v", msg.Err)
			m.saveStatusErr = true
		} else {
			m.saveStatusMsg = fmt.Sprintf("saved to %s", msg.Path)
			m.saveStatusErr = false
		}
		return m, nil

	case tea.KeyMsg:
		// The filename prompt (see SaveResponseRaw) claims every key while
		// open, same as groupspane's create/rename prompt - none of the
		// tab/scroll/copy bindings below should fire while typing a
		// filename (e.g. "y" in "my-response.json" must not also trigger
		// CopyResponse).
		if m.savingToFile {
			return m.handleSaveToFileKey(msg)
		}

		// Direct tab access
		if keybindings.Matches(msg, m.keys.DirectTab) {
			switch msg.String() {
			case "1":
				m.activeTab = int(TabBody)
				m.updateViewportForActiveTab()
			case "2":
				m.activeTab = int(TabHeaders)
				m.updateViewportForActiveTab()
			case "3":
				m.activeTab = int(TabTiming)
				m.updateViewportForActiveTab()
			case "4":
				// Load test mode only has 3 tabs (Overview/Latency/Errors)
				// - there's no load-test equivalent of "the request that
				// was sent" to jump to (the JobConfig's request doesn't
				// change per stats update).
				if !m.isLoadTest {
					m.activeTab = int(TabRequest)
					m.updateViewportForActiveTab()
				}
			}
		}

		// Tab navigation
		if keybindings.Matches(msg, m.keys.TabNavPrev) {
			maxTabs := m.getMaxTabs()
			m.activeTab = (m.activeTab - 1 + maxTabs) % maxTabs
			m.updateViewportForActiveTab()
		}
		if keybindings.Matches(msg, m.keys.TabNavNext) {
			maxTabs := m.getMaxTabs()
			m.activeTab = (m.activeTab + 1) % maxTabs
			m.updateViewportForActiveTab()
		}

		// Copy handling
		if keybindings.Matches(msg, m.keys.CopyResponse) {
			if m.Response != nil && !m.isLoadTest {
				return m, m.copyToClipboard(m.Response.Body)
			}
		}

		// Raw/pretty toggle - only meaningful on the Body tab, which is
		// the only one formatContentByType touches at all.
		if keybindings.Matches(msg, m.keys.ToggleRawBody) {
			if m.Response != nil && !m.isLoadTest && TabIndex(m.activeTab) == TabBody {
				m.rawBody = !m.rawBody
				m.updateViewportForActiveTab()
			}
			return m, nil
		}

		// Save-to-file prompt - always the raw response body (see
		// saveResponseBodyCmd), regardless of ToggleRawBody's current
		// display state or which tab happens to be active, same as
		// CopyResponse above.
		if keybindings.Matches(msg, m.keys.SaveResponseRaw) {
			if m.Response != nil && !m.isLoadTest {
				m.savingToFile = true
				m.saveStatusMsg = ""
				m.filenameInput.SetValue("")
				m.filenameInput.Focus()
				return m, textinput.Blink
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleSaveToFileKey handles input while the filename prompt is open
// (see SaveResponseRaw) - enter confirms and dispatches the save, esc
// cancels back to normal browsing, anything else is forwarded to
// filenameInput. Mirrors groupspane's create/rename prompt handling.
func (m *ResponsePane) handleSaveToFileKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.savingToFile = false
		m.filenameInput.Blur()
		return m, nil
	}
	if msg.String() == tea.KeyEnter.String() {
		path, err := resolveSavePath(m.filenameInput.Value())
		if err != nil {
			m.saveStatusMsg = err.Error()
			m.saveStatusErr = true
			return m, nil
		}
		m.savingToFile = false
		m.filenameInput.Blur()
		return m, saveResponseBodyCmd(path, m.Response.Body)
	}

	var cmd tea.Cmd
	m.filenameInput, cmd = m.filenameInput.Update(msg)
	return m, cmd
}

// resolveSavePath validates and expands the user-typed filename: a
// leading "~" is expanded to the home directory (bare os.WriteFile doesn't
// do this on its own - a raw "~/x" would otherwise create a literal
// directory named "~"), and a blank filename is rejected outright rather
// than silently doing nothing. Anything else (relative or absolute) is
// passed straight through to os.WriteFile, same as internal/cli/output.go
// already does for its own -output flag.
func resolveSavePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("filename cannot be empty")
	}
	if raw == "~" || strings.HasPrefix(raw, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve ~: %w", err)
		}
		raw = filepath.Join(home, strings.TrimPrefix(raw, "~"))
	}
	return raw, nil
}

// saveResponseBodyCmd writes content (always the raw response body, never
// the formatted/highlighted view - see ToggleRawBody) to path in the
// background, off the UI goroutine, reporting the outcome via
// SaveResponseBodyMsg.
func saveResponseBodyCmd(path, content string) tea.Cmd {
	return func() tea.Msg {
		err := os.WriteFile(path, []byte(content), 0644)
		return SaveResponseBodyMsg{Path: path, Err: err}
	}
}

// copyToClipboard copies content to the system clipboard
func (m ResponsePane) copyToClipboard(content string) tea.Cmd {
	return func() tea.Msg {
		clipboard.WriteAll(content)
		return nil
	}
}

// updateViewportForActiveTab updates the viewport content based on the active tab
func (m *ResponsePane) updateViewportForActiveTab() {
	if m.isLoadTest {
		m.updateLoadTestTabContent()
		return
	}

	if m.Response == nil {
		return
	}

	var content string
	switch TabIndex(m.activeTab) {
	case TabBody:
		switch {
		case m.Response.Error != "":
			content = m.Response.Error
		case m.rawBody:
			content = m.Response.Body
		default:
			contentType := m.Response.ParseContentType()
			content = formatContentByType(m.Response.Body, contentType)
		}
	case TabHeaders:
		content = m.renderHeaders()
	case TabTiming:
		content = m.renderTiming()
	case TabRequest:
		content = m.renderRequest()
	}
	m.viewport.SetContent(content)
}

// updateLoadTestTabContent updates the viewport content for load test tabs
func (m *ResponsePane) updateLoadTestTabContent() {
	if m.LoadTestStats == nil {
		m.viewport.SetContent("No data")
		return
	}

	var content string
	switch TabIndex(m.activeTab) {
	case TabLoadTestOverview:
		content = m.renderLoadTestOverview()
	case TabLoadTestLatency:
		content = m.renderLoadTestLatency()
	case TabLoadTestErrors:
		content = m.renderLoadTestErrors()
	}
	m.viewport.SetContent(content)
}
