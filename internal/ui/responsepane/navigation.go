package responsepane

import "strconv"

// scrollToLine centers line (0-based) in the viewport when possible,
// rather than placing it flush against the very top edge - used by both
// the goto-line jump below and search match navigation (see search.go),
// so a jumped-to line lands with some surrounding context visible instead
// of right at the edge. SetYOffset clamps out-of-range values on its own
// (negative goes to the top, past-the-end goes to the bottom), so no
// bounds checking is needed here.
func (m *ResponsePane) scrollToLine(line int) {
	m.viewport.SetYOffset(line - m.viewport.Height/2)
}

// isLineVisible reports whether line (0-based) is within the viewport's
// currently visible range - used by jumpToNearestMatch (see search.go) to
// tell an ordinary scroll (j/k, pgup/pgdn - the previously found match
// stays on screen) apart from a real jump elsewhere (gg/G/{n}g - it
// doesn't).
func (m *ResponsePane) isLineVisible(line int) bool {
	top := m.viewport.YOffset
	bottom := top + m.viewport.VisibleLineCount()
	return line >= top && line < bottom
}

// beginGoto arms gPending on a bare "g" keypress (see GotoTop) - the next
// keystroke decides what happens: another "g" goes straight to the top
// ("gg"), digits are buffered (see continueGoto) toward a "{n}g" jump to
// that line, and anything else cancels.
func (m *ResponsePane) beginGoto() {
	m.gPending = true
	m.gotoLineBuffer = ""
}

// continueGoto handles a keystroke while gPending is armed. It reports
// whether key was consumed as part of the sequence - false means the
// sequence was cancelled and key should fall through to Update's normal
// handling instead (so, e.g., "g" then an unrelated key like "j" still
// scrolls down rather than silently doing nothing).
func (m *ResponsePane) continueGoto(key string) (consumed bool) {
	switch {
	case key == "g":
		if m.gotoLineBuffer != "" {
			m.jumpToLine(parseGotoLineBuffer(m.gotoLineBuffer))
		} else {
			m.viewport.GotoTop()
		}
		m.gPending = false
		m.gotoLineBuffer = ""
		return true

	case len(key) == 1 && key[0] >= '0' && key[0] <= '9':
		// Capped well past any realistic response body's line count -
		// just a guard against an unbounded buffer if a key gets stuck
		// repeating.
		if len(m.gotoLineBuffer) < 9 {
			m.gotoLineBuffer += key
		}
		return true

	default:
		m.gPending = false
		m.gotoLineBuffer = ""
		return false
	}
}

// jumpToLine scrolls to line n, 1-indexed (vim/less convention - "12g"
// means line 12, not line 13).
func (m *ResponsePane) jumpToLine(n int) {
	m.scrollToLine(n - 1)
}

func parseGotoLineBuffer(buf string) int {
	n, err := strconv.Atoi(buf)
	if err != nil {
		return 0
	}
	return n
}
