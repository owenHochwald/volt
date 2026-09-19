package responsepane

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/history"
)

// historyViewerChromeLines mirrors jarViewerChromeLines - the header +
// hint line HistoryViewerListView wraps the list itself in, subtracted
// from historyViewerHeight so the list's own height doesn't overflow its
// box (see resizeHistoryList).
const historyViewerChromeLines = 2

// historyEntryItem adapts one history.Entry to list.Item for
// historyList below.
type historyEntryItem struct {
	entry history.Entry
}

func (i historyEntryItem) Title() string {
	status := "ERR"
	if i.entry.Response != nil && i.entry.Response.Error == "" {
		status = fmt.Sprintf("%d", i.entry.Response.StatusCode)
	}
	method, url := "", ""
	if i.entry.SentRequest != nil {
		method, url = i.entry.SentRequest.Method, i.entry.SentRequest.URL
	}
	return fmt.Sprintf("%s · %s %s", status, method, url)
}

func (i historyEntryItem) Description() string {
	duration := ""
	if i.entry.Response != nil {
		duration = fmt.Sprintf(" · %d ms", i.entry.Response.Duration.Milliseconds())
	}
	return i.entry.SentAt.Format("15:04:05") + duration
}

func (i historyEntryItem) FilterValue() string {
	if i.entry.SentRequest == nil {
		return ""
	}
	return i.entry.SentRequest.Method + " " + i.entry.SentRequest.URL
}

// newHistoryList builds the viewer's list from entries (see
// history.Store.Entries, already newest-first).
func newHistoryList(entries []history.Entry) list.Model {
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = historyEntryItem{entry: e}
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)

	l := list.New(items, delegate, 0, 0)
	l.Title = fmt.Sprintf("History (%d)", len(items))
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	return l
}

// openHistoryViewer opens the "H" viewer (see ResponsePane.viewingHistory)
// over every past send this session has recorded - nothing is changed by
// opening it, only by "d"/"D" (delete/clear) or "enter" (view - see
// handleHistoryViewerKey).
func (m *ResponsePane) openHistoryViewer() {
	m.viewingHistory = true
	m.historyList = newHistoryList(m.history.Entries())
	m.resizeHistoryList()
}

// resizeHistoryList sizes historyList to historyViewerWidth/Height (see
// SetHistoryViewerSize) minus historyViewerChromeLines - called on open
// and whenever the pane itself is resized while the viewer is open.
func (m *ResponsePane) resizeHistoryList() {
	height := max(m.historyViewerHeight-historyViewerChromeLines, 3)
	m.historyList.SetSize(max(m.historyViewerWidth, 1), height)
}

// refreshHistoryList rebuilds historyList from the store's current state -
// called after "d"/"D" change what's actually stored, so the list
// reflects the deletion instead of showing an entry that no longer
// exists.
func (m *ResponsePane) refreshHistoryList() {
	m.historyList = newHistoryList(m.history.Entries())
	m.resizeHistoryList()
}

// viewHistoryEntry makes a past send the currently displayed response -
// see displayResponse's own doc comment for why this bypasses SetResponse
// (no re-recording into history, no re-capturing cookies/auth for what's
// actually an old response, not a new arrival).
func (m *ResponsePane) viewHistoryEntry(entry history.Entry) {
	m.displayResponse(entry.Response, entry.SentRequest)
}

// handleHistoryViewerKey handles input while the past-sends viewer is
// open - esc closes it (only outside of the list's own "/" filter-typing
// mode, which esc should just clear instead, same guard
// handleJarViewerKey uses), "enter" views the selected entry and closes
// the viewer, "d" deletes the selected entry, "D" clears the whole
// history, anything else is forwarded to historyList (including its own
// "/" filter and cursor movement).
func (m *ResponsePane) handleHistoryViewerKey(msg tea.KeyPressMsg) (*ResponsePane, tea.Cmd) {
	filtering := m.historyList.FilterState() == list.Filtering
	if msg.String() == "esc" && !filtering {
		m.viewingHistory = false
		return m, nil
	}
	if !filtering {
		switch msg.String() {
		case "enter":
			if item, ok := m.historyList.SelectedItem().(historyEntryItem); ok {
				m.viewHistoryEntry(item.entry)
			}
			m.viewingHistory = false
			return m, nil
		case "x":
			if item, ok := m.historyList.SelectedItem().(historyEntryItem); ok {
				m.startDiff(item.entry)
			}
			m.viewingHistory = false
			return m, nil
		case "d":
			if item, ok := m.historyList.SelectedItem().(historyEntryItem); ok {
				m.history.Delete(item.entry.ID)
				m.refreshHistoryList()
			}
			return m, nil
		case "D":
			m.history.Clear()
			m.refreshHistoryList()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.historyList, cmd = m.historyList.Update(msg)
	return m, cmd
}

// HistoryViewerListView renders the past-sends viewer for app/view.go to
// place in the request pane's own box while it's open - see
// IsViewingHistory - the same slot StylePickerListView/JarViewerListView
// already use, mutually exclusive with them (all three claim every key
// while open - see ResponsePane.Update).
func (m ResponsePane) HistoryViewerListView() string {
	header := m.styles.Text.Label.Render("Past sends (this session)")
	hint := footerHelpStyle.Render("enter: view · x: diff vs current · d: delete · D: clear all · esc: close · /: filter")
	return lipgloss.JoinVertical(lipgloss.Left, header, m.historyList.View(), hint)
}
