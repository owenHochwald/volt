package wspane

import (
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ws"
)

// ConnectResultMsg carries the outcome of a background dial (see
// connectCmd) - Session is nil on failure.
type ConnectResultMsg struct {
	Session *ws.Session
	Err     error
}

func connectCmd(url string) tea.Cmd {
	return func() tea.Msg {
		session, err := ws.Connect(url)
		return ConnectResultMsg{Session: session, Err: err}
	}
}

// EventMsg carries one asynchronous event from a live session (see
// waitForEventCmd) - a received message, or the connection closing.
type EventMsg struct {
	Event ws.Event
}

// waitForEventCmd blocks for the next event on session's own channel and
// re-arms itself (see WSPane.Update) after every one but the last -
// mirrors ui.WaitForLoadTestUpdatesCmd's re-arming poll, the established
// pattern for draining a long-lived channel through bubbletea's Cmd/Msg
// loop instead of a goroutine that writes directly into the model.
func waitForEventCmd(session *ws.Session) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-session.Events()
		if !ok {
			return EventMsg{Event: ws.Event{Closed: true}}
		}
		return EventMsg{Event: event}
	}
}

// SendResultMsg carries the outcome of a background Send (see sendCmd).
type SendResultMsg struct {
	Text string
	Err  error
}

func sendCmd(session *ws.Session, text string) tea.Cmd {
	return func() tea.Msg {
		err := session.Send(text)
		return SendResultMsg{Text: text, Err: err}
	}
}

// CloseWSModalMsg signals the parent app to hide this modal - the
// session, if any, keeps running (see the package doc).
type CloseWSModalMsg struct{}

func closeCmd() tea.Msg { return CloseWSModalMsg{} }
