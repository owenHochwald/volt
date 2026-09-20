package responsepane

import (
	"strings"
	"time"
)

// ArmWSStream starts the WebSocket tab's capture window - called on every
// successful message send from the WebSocket pane (see app/update.go's
// wspane.SendResultMsg handling), but only takes effect the first time:
// once armed, it stays armed (and the tab, once shown, stays shown and
// keeps growing) for the rest of the session. Re-clearing on every send
// was tried and reverted - it meant sending a second message could kick
// you off the tab entirely (back to Body, with nothing to show if no
// HTTP response had ever been sent either), which fights the very "live"
// behavior the tab exists for. Only messages received after the *first*
// send are ever shown at all (see AppendWSData); anything the socket
// pushed before that isn't retroactively displayed.
func (m *ResponsePane) ArmWSStream() {
	m.wsArmed = true
}

// AppendWSData records one message received from the WebSocket pane's
// session (see app/update.go's wspane.EventMsg handling) - ignored until
// ArmWSStream has run at least once (see its own doc), so a stream the
// socket pushes before any send doesn't end up misleadingly attributed to
// "what came back" from one.
func (m *ResponsePane) AppendWSData(text string, at time.Time) {
	if !m.wsArmed {
		return
	}
	m.wsLines = append(m.wsLines, at.Format("15:04:05")+"  "+text)
	if TabIndex(m.activeTab) == TabWebSocket {
		m.updateViewportForActiveTab()
	}
}

// hasWSTab reports whether the WebSocket tab currently has anything to
// show - the sole condition for it to exist in the tab bar at all (see
// tabs.go's renderTabs/getMaxTabs).
func (m ResponsePane) hasWSTab() bool {
	return len(m.wsLines) > 0
}

// renderWSStream renders the WebSocket tab's content - plain, timestamped
// lines, oldest first, growing as more arrive (see AppendWSData). No
// syntax highlighting or raw/pretty toggle: it's a live log, not a
// response body.
func (m ResponsePane) renderWSStream() string {
	if len(m.wsLines) == 0 {
		return m.styles.Text.Faint.Render("(no messages yet)")
	}
	return strings.Join(m.wsLines, "\n")
}
