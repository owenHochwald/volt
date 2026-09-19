package wspane

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ws"
)

func (m WSPane) Update(msg tea.Msg) (WSPane, tea.Cmd) {
	switch msg := msg.(type) {
	case ConnectResultMsg:
		if msg.Err != nil {
			m.status = statusDisconnected
			m.appendSystem("Error connecting: " + msg.Err.Error())
			return m, nil
		}
		m.session = msg.Session
		m.status = statusConnected
		m.appendSystem("Connected to " + strings.TrimSpace(m.urlInput.Value()))
		return m, waitForEventCmd(m.session)

	case EventMsg:
		if msg.Event.Closed {
			m.status = statusDisconnected
			m.session = nil
			if msg.Event.Err != nil {
				m.appendSystem("Disconnected: " + msg.Event.Err.Error())
			} else {
				m.appendSystem("Disconnected")
			}
			return m, nil
		}
		m.log = append(m.log, msg.Event.Entry)
		if m.session != nil {
			// Still connected - keep draining this session's channel for
			// the next event.
			return m, waitForEventCmd(m.session)
		}
		return m, nil

	case SendResultMsg:
		if msg.Err != nil {
			m.appendSystem("Error sending message: " + msg.Err.Error())
			return m, nil
		}
		m.log = append(m.log, ws.LogEntry{Kind: ws.EntrySent, Text: msg.Text, At: time.Now()})
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *WSPane) appendSystem(text string) {
	m.log = append(m.log, ws.LogEntry{Kind: ws.EntrySystem, Text: text, At: time.Now()})
}

func (m WSPane) handleKey(msg tea.KeyPressMsg) (WSPane, tea.Cmd) {
	if msg.String() == "esc" {
		return m, closeCmd
	}
	if msg.String() == "tab" || msg.String() == "shift+tab" {
		m.blurAll()
		if msg.String() == "tab" {
			m.focus = (m.focus + 1) % focusFieldCount
		} else {
			m.focus = (m.focus + focusFieldCount - 1) % focusFieldCount
		}
		m.focusCurrent()
		return m, nil
	}
	if msg.String() == "enter" {
		return m.activate()
	}

	var cmd tea.Cmd
	switch m.focus {
	case focusURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case focusCompose:
		m.composeInput, cmd = m.composeInput.Update(msg)
	}
	return m, cmd
}

// activate is the "primary action" Enter always triggers, meaning
// depends on focus: composing a message sends it, anywhere else toggles
// connect/disconnect - so typing a URL and pressing Enter connects
// without needing to tab to a separate button first.
func (m WSPane) activate() (WSPane, tea.Cmd) {
	if m.focus == focusCompose {
		if m.status != statusConnected {
			return m, nil
		}
		text := strings.TrimSpace(m.composeInput.Value())
		if text == "" {
			return m, nil
		}
		m.composeInput.SetValue("")
		return m, sendCmd(m.session, text)
	}

	switch m.status {
	case statusConnected:
		if m.session != nil {
			m.session.Close()
		}
		return m, nil
	case statusConnecting:
		return m, nil
	default:
		url := strings.TrimSpace(m.urlInput.Value())
		if url == "" {
			m.appendSystem("Enter a URL first")
			return m, nil
		}
		m.status = statusConnecting
		return m, connectCmd(url)
	}
}

func (m *WSPane) blurAll() {
	m.urlInput.Blur()
	m.composeInput.Blur()
}

func (m *WSPane) focusCurrent() {
	switch m.focus {
	case focusURL:
		m.urlInput.Focus()
	case focusCompose:
		m.composeInput.Focus()
	}
}
