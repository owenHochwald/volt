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
	"github.com/owenHochwald/Volt/internal/ui/design"
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

	keys   keybindings.KeyMap
	styles design.Styles

	urlInput     textinput.Model
	composeInput textinput.Model
	focus        focusField

	status status

	session *ws.Session
	log     []ws.LogEntry
}

// SetupWSPane constructs an unopened WSPane; call Open() before
// displaying it.
func SetupWSPane(keys keybindings.KeyMap, optionalStyles ...design.Styles) WSPane {
	styles := design.NewStyles(design.DefaultTheme())
	if len(optionalStyles) > 0 {
		styles = optionalStyles[0]
	}

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
		styles:       styles,
		urlInput:     urlInput,
		composeInput: composeInput,
	}
}

// SetStyles updates the theme-derived styles used to render this pane -
// called once at startup and again whenever the active theme changes (see
// app.Model.applyTheme), the same way requestpane/responsepane already do.
func (m *WSPane) SetStyles(styles design.Styles) {
	m.styles = styles
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
// points to - Open() calls this rather than duplicating it, so opening
// the modal always leaves the right textinput.Model actually Focus()-ed
// (bubbles' textinput otherwise silently ignores keystrokes while
// unfocused). Unlike Open(), this never resets m.focus itself.
func (m *WSPane) SetFocused(focused bool) {
	m.Focused = focused
	if focused {
		m.focusCurrent()
	} else {
		m.blurAll()
	}
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

// SetURL pre-fills the URL field - called once at startup with whatever
// address last connected successfully (see storage.LoadLastWSURL), since
// a session typically keeps reconnecting to the same one.
func (m *WSPane) SetURL(url string) {
	m.urlInput.SetValue(url)
}
