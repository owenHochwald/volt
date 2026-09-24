package requestpane

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/history"
	"github.com/owenHochwald/Volt/internal/http"
)

// requestHistoryChromeLines mirrors responsepane's historyViewerChromeLines
// - the header + hint line HistoryViewerListView wraps the list itself in,
// subtracted from historyViewerHeight so the list's own height doesn't
// overflow its box (see resizeHistoryList).
const requestHistoryChromeLines = 2

// requestHistoryItem adapts one history.Entry to list.Item for
// newRequestHistoryList below. position is its index into
// history.Entries() (newest first) at the moment the list was built -
// carried along so "enter" (see handleRequestHistoryViewerKey) can resume
// stepRequestHistory from exactly here, the same way responsepane's own
// viewHistoryEntry locates a selected item's true absolute position.
type requestHistoryItem struct {
	entry    history.Entry
	position int
}

func (i requestHistoryItem) Title() string {
	method, url := "", ""
	if i.entry.SentRequest != nil {
		method, url = i.entry.SentRequest.Method, i.entry.SentRequest.URL
	}
	return fmt.Sprintf("%s %s", method, url)
}

func (i requestHistoryItem) Description() string {
	return i.entry.SentAt.Format("15:04:05")
}

func (i requestHistoryItem) FilterValue() string {
	if i.entry.SentRequest == nil {
		return ""
	}
	return i.entry.SentRequest.Method + " " + i.entry.SentRequest.URL
}

// newRequestHistoryList builds the viewer's list from entries (see
// history.Store.Entries, already newest-first).
func newRequestHistoryList(entries []history.Entry) list.Model {
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = requestHistoryItem{entry: e, position: i}
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)

	l := list.New(items, delegate, 0, 0)
	l.Title = fmt.Sprintf("Request history (%d)", len(items))
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	return l
}

// openHistoryViewer opens the "H" viewer (see IsViewingHistory) over every
// request sent this session - nothing is changed by opening it, only by
// "enter" (see handleRequestHistoryViewerKey).
func (m *RequestPane) openHistoryViewer() {
	m.viewingHistory = true
	m.historyList = newRequestHistoryList(m.history.Entries())
	m.resizeHistoryList()
}

// resizeHistoryList sizes historyList to historyViewerWidth/Height (see
// SetHistoryViewerSize) minus requestHistoryChromeLines - called on open
// and whenever the pane itself is resized while the viewer is open.
func (m *RequestPane) resizeHistoryList() {
	height := max(m.historyViewerHeight-requestHistoryChromeLines, 3)
	m.historyList.SetSize(max(m.historyViewerWidth, 1), height)
}

// requestFromHistoryEntry builds a fresh, schema-less request from a past
// send's fully-resolved SentRequest (path/query params substituted,
// server/jar headers and {{variable}}s already filled in - the exact
// shape resolvedRequest built right before sending it - see sync.go). No
// PathParamSchema/QueryParamSchema/BodyFieldSchema/ParamValues/GroupID/
// ServerID carry over: those describe the *template* the original request
// was built from, which the resolved URL/body no longer match (a
// {cluster_id} placeholder, or a {{token}} typed into a param value, is
// gone from SentRequest - replaced by the literal value that was actually
// sent). Loading it schema-less is what makes the URL/body fields show
// exactly what went out on the wire, not the template that produced it -
// the whole point of browsing this history from the request pane rather
// than just rereading it off the response pane's own read-only "H".
// ID is deliberately left at 0 too: this is a fresh, unsaved snapshot to
// edit and resend, not a reopen of whatever request (if any) originally
// produced it - ctrl+s here creates a new saved request rather than
// silently overwriting one just because a resolved historical copy
// happened to carry over its ID.
func requestFromHistoryEntry(sent *http.Request) *http.Request {
	headers := make(map[string]string, len(sent.Headers))
	for k, v := range sent.Headers {
		headers[k] = v
	}
	return &http.Request{
		Name:    sent.Name,
		Method:  sent.Method,
		URL:     sent.URL,
		Headers: headers,
		Body:    sent.Body,
	}
}

// loadHistoryEntryAtPosition loads entry's sent request into the editable
// form (see requestFromHistoryEntry) and records position (an index into
// history.Entries()) so stepRequestHistory continues from exactly here -
// snapshotting whatever was loaded/being edited before, the first time
// this runs (see preHistoryRequest's own doc comment).
func (m *RequestPane) loadHistoryEntryAtPosition(entry history.Entry, position int) {
	if entry.SentRequest == nil {
		return
	}
	if !m.browsingHistory {
		// syncRequest first: m.Request's Method/URL/Name/Headers/Body are
		// only ever synced from the UI's own fields at specific points
		// (save, submit, copy-as-curl - see sync.go), not after every
		// keystroke, so Snapshot() alone could otherwise capture stale
		// pre-edit values instead of whatever's actually currently typed.
		m.syncRequest()
		snapshot := m.Snapshot()
		m.preHistoryRequest = &snapshot
	}
	m.browsingHistory = true
	m.historyPosition = position
	m.reinitRequestPane(requestFromHistoryEntry(entry.SentRequest))
}

// stepRequestHistory moves "]" (delta -1, newer) / "[" (delta +1, older)
// through the same history ViewRequestHistory ("H") browses, one entry at
// a time, without opening its full list overlay - a no-op past either end
// (nothing newer than "not browsing", nothing older than the oldest
// recorded entry).
func (m *RequestPane) stepRequestHistory(delta int) {
	entries := m.history.Entries()
	if len(entries) == 0 {
		return
	}

	if !m.browsingHistory {
		// Not browsing yet - "newer" has nowhere to go, and "older" starts
		// browsing at the newest entry (position 0).
		if delta <= 0 {
			return
		}
		m.loadHistoryEntryAtPosition(entries[0], 0)
		return
	}

	next := m.historyPosition + delta
	switch {
	case next < 0:
		// Stepped "newer" past the newest entry - restore whatever was
		// loaded/being edited before history browsing started instead of
		// just stopping at position 0 again.
		m.browsingHistory = false
		m.historyPosition = 0
		if m.preHistoryRequest != nil {
			m.reinitRequestPane(m.preHistoryRequest)
			m.preHistoryRequest = nil
		}
	case next >= len(entries):
		return // already at the oldest entry
	default:
		m.loadHistoryEntryAtPosition(entries[next], next)
	}
}

// handleRequestHistoryViewerKey handles input while the sent-request
// viewer is open - esc closes it (only outside of the list's own "/"
// filter-typing mode, which esc should just clear instead, same guard
// responsepane's own history/jar viewers use), "enter" loads the selected
// entry into the editable form and closes the viewer, anything else is
// forwarded to historyList (including its own "/" filter and cursor
// movement).
func (m *RequestPane) handleRequestHistoryViewerKey(msg tea.KeyPressMsg) (RequestPane, tea.Cmd) {
	filtering := m.historyList.FilterState() == list.Filtering
	if msg.String() == "esc" && !filtering {
		m.viewingHistory = false
		return *m, nil
	}
	if !filtering && msg.String() == "enter" {
		if item, ok := m.historyList.SelectedItem().(requestHistoryItem); ok {
			m.loadHistoryEntryAtPosition(item.entry, item.position)
		}
		m.viewingHistory = false
		return *m, nil
	}

	var cmd tea.Cmd
	m.historyList, cmd = m.historyList.Update(msg)
	return *m, cmd
}

// IsViewingHistory reports whether the sent-request viewer (see
// ViewRequestHistory, "H") is currently open - checked by app/update.go so
// esc closes the viewer first instead of the panel-switch/quit-arm
// swallowing it before the viewer ever sees it, the same reason
// responsepane.ResponsePane.IsViewingHistory exists.
func (m *RequestPane) IsViewingHistory() bool {
	return m.viewingHistory
}

// SetHistoryViewerSize sizes the sent-request viewer's list (see
// HistoryViewerListView) - this pane's own box dimensions, called from
// app/layout.go's applyLayout the same way SetWidth/SetHeight already
// are. Stored regardless of whether the viewer is open right now, so
// openHistoryViewer has a real size to start from instead of whatever the
// zero value would size the list to.
func (m *RequestPane) SetHistoryViewerSize(width, height int) {
	m.historyViewerWidth = width
	m.historyViewerHeight = height
	if m.viewingHistory {
		m.resizeHistoryList()
	}
}

// HistoryViewerListView renders the sent-request viewer for app/view.go to
// place in this pane's own box while it's open - see IsViewingHistory.
func (m RequestPane) HistoryViewerListView() string {
	header := m.styles.Text.Label.Render("Sent requests (this session)")
	hint := footerHelpStyle.Render("enter: load into form · esc: close · /: filter")
	return lipgloss.JoinVertical(lipgloss.Left, header, m.historyList.View(), hint)
}
