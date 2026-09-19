// Package wspane is the WebSocket live-session modal - a fully separate
// screen from the HTTP request/response panes (see the session's own
// notes: WebSocket is a long-lived, bidirectional connection, not a
// single request/response round trip, so it doesn't share their tab/
// viewport machinery). Opened with ctrl+k, hidden (not torn down) with
// esc - the underlying *ws.Session, if connected, keeps running and
// logging messages in the background while hidden, the same way a load
// test keeps running while the user is looking at another panel.
package wspane

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ws"
)

// status is the connection's current state, driving both the status
// badge (see view.go) and what the connect/disconnect toggle does next.
type status int

const (
	statusDisconnected status = iota
	statusConnecting
	statusConnected
)

// focusField is which of this pane's three focusable controls has the
// cursor - deliberately not a dynamic components() list the way
// requestpane's is, since this screen's shape never changes.
type focusField int

const (
	focusURL focusField = iota
	focusConnectButton
	focusCompose
	focusFieldCount
)

// WSPane is the WebSocket modal's model - the session, its message log,
// and the three input controls (URL, connect/disconnect button, compose)
// that drive it.
type WSPane struct {
	width, height int
	Focused       bool

	keys keybindings.KeyMap

	urlInput     textinput.Model
	composeInput textinput.Model
	focus        focusField

	status status

	// docked reports whether this pane is currently rendered as a real
	// panel (app.Model.wsDocked) rather than a floating overlay - only
	// affects the footer hint text (see footerHint), since esc's "hide"
	// behavior only applies while floating (see app/update.go's Quit
	// handling, which special-cases showWSModal, not wsDocked).
	docked bool

	session *ws.Session
	log     []ws.LogEntry
}

// SetupWSPane constructs an unopened WSPane; call Open() before
// displaying it.
func SetupWSPane(keys keybindings.KeyMap) WSPane {
	urlInput := textinput.New()
	urlInput.Placeholder = "ws://localhost:8080/socket"
	urlInput.CharLimit = 500
	urlInput.SetWidth(60)

	composeInput := textinput.New()
	composeInput.Placeholder = "message to send"
	composeInput.CharLimit = 2000
	composeInput.SetWidth(60)

	return WSPane{
		width:        70,
		height:       20,
		keys:         keys,
		urlInput:     urlInput,
		composeInput: composeInput,
	}
}

func (m WSPane) Init() tea.Cmd {
	return nil
}

// Open focuses the URL field and marks the pane focused - deliberately
// doesn't reset anything else (the URL, the log, an in-progress or live
// session), so reopening the modal after hiding it with esc picks up
// exactly where it was left, the point of not tearing the session down
// on hide in the first place.
func (m *WSPane) Open() {
	m.focus = focusURL
	m.SetFocused(true)
}

// SetFocused activates or deactivates whichever control m.focus currently
// points to (the docked panel's equivalent of Open's urlInput.Focus() -
// called on every focus change, whether cycling in via ctrl+w/ctrl+e
// while docked or opening the floating modal) - without this, a
// freshly-docked pane's URL/compose textinput.Model would never have had
// Focus() called on it at all and would silently ignore keystrokes,
// since bubbles' textinput only accepts input while focused. Unlike
// Open(), this never resets m.focus itself, so cycling away from and
// back to a docked panel preserves whichever field was last selected.
func (m *WSPane) SetFocused(focused bool) {
	m.Focused = focused
	if focused {
		m.focusCurrent()
	} else {
		m.blurAll()
	}
}

// IsEditing reports whether a text field (URL or compose), rather than
// the connect/disconnect button, currently has the cursor - checked by
// app/update.go the same way requestpane.IsEditing is, so a global
// single-key binding like "?" (context help) doesn't hijack a literal
// "?" typed into a compose message or URL query string.
func (m WSPane) IsEditing() bool {
	return m.focus == focusURL || m.focus == focusCompose
}

func (m *WSPane) SetWidth(width int) {
	m.width = width
	w := max(width-10, 20)
	m.urlInput.SetWidth(w)
	m.composeInput.SetWidth(w)
}

func (m *WSPane) SetHeight(height int) {
	m.height = height
}

// SetDocked records whether this pane is currently shown docked (a real
// panel) or floating (a modal) - see the docked field's own doc.
func (m *WSPane) SetDocked(docked bool) {
	m.docked = docked
}
