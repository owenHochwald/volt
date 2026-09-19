package responsepane

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/authjar"
)

// jarViewerChromeLines mirrors stylePickerChromeLines - the header + hint
// line JarViewerListView wraps the list itself in, subtracted from
// jarViewerHeight so the list's own height doesn't overflow its box (see
// resizeJarList).
const jarViewerChromeLines = 2

// jarEntryKind distinguishes a cookie from an authorization header in the
// viewer's list - both render as jarEntryItem, but delete needs to know
// which of the jar's two maps to remove from.
type jarEntryKind int

const (
	jarEntryCookie jarEntryKind = iota
	jarEntryAuthHeader
)

// jarEntryItem adapts one remembered cookie or authorization header to
// list.Item for jarList below.
type jarEntryItem struct {
	kind  jarEntryKind
	name  string
	value string
}

func (i jarEntryItem) Title() string {
	if i.kind == jarEntryAuthHeader {
		return "🔑 " + i.name
	}
	return "🍪 " + i.name
}
func (i jarEntryItem) Description() string { return i.value }
func (i jarEntryItem) FilterValue() string { return i.name }

// newJarList builds the viewer's list from snapshot - cookies first, then
// authorization headers, each already sorted by name (see
// authjar.Store.Snapshot).
func newJarList(snapshot authjar.Snapshot) list.Model {
	items := make([]list.Item, 0, len(snapshot.Cookies)+len(snapshot.AuthHeaders))
	for _, c := range snapshot.Cookies {
		items = append(items, jarEntryItem{kind: jarEntryCookie, name: c.Name, value: c.Value})
	}
	for _, h := range snapshot.AuthHeaders {
		items = append(items, jarEntryItem{kind: jarEntryAuthHeader, name: h.Name, value: h.Value})
	}

	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(0)

	l := list.New(items, delegate, 0, 0)
	l.Title = "Saved for " + snapshot.Host
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	return l
}

// openJarViewer opens the "v" viewer (see ResponsePane.viewingJar) over
// whatever's currently remembered for the response's host - nothing is
// captured or changed by opening it, only by "d"/"D" (see
// handleJarViewerKey) or CaptureAuth ("c") separately.
func (m *ResponsePane) openJarViewer() {
	m.viewingJar = true
	m.jarList = newJarList(m.jar.Snapshot(m.currentHost()))
	m.resizeJarList()
}

// resizeJarList sizes jarList to jarViewerWidth/Height (see
// SetJarViewerSize) minus jarViewerChromeLines - called on open and
// whenever the pane itself is resized while the viewer is open.
func (m *ResponsePane) resizeJarList() {
	height := max(m.jarViewerHeight-jarViewerChromeLines, 3)
	m.jarList.SetSize(max(m.jarViewerWidth, 1), height)
}

// refreshJarList rebuilds jarList from the jar's current state - called
// after "d"/"D" change what's actually stored, so the list reflects the
// deletion instead of showing an entry that no longer exists.
func (m *ResponsePane) refreshJarList() {
	m.jarList = newJarList(m.jar.Snapshot(m.currentHost()))
	m.resizeJarList()
}

// handleJarViewerKey handles input while the saved-cookies/auth-headers
// viewer is open - esc closes it (only outside of the list's own "/"
// filter-typing mode, which esc should just clear instead, same guard
// handleStylePickerKey uses), "d" deletes the selected entry, "D" clears
// everything remembered for this host, anything else is forwarded to
// jarList (including its own "/" filter and cursor movement).
func (m *ResponsePane) handleJarViewerKey(msg tea.KeyPressMsg) (*ResponsePane, tea.Cmd) {
	filtering := m.jarList.FilterState() == list.Filtering
	if msg.String() == "esc" && !filtering {
		m.viewingJar = false
		return m, nil
	}
	if !filtering {
		host := m.currentHost()
		switch msg.String() {
		case "d":
			if entry, ok := m.jarList.SelectedItem().(jarEntryItem); ok {
				if entry.kind == jarEntryAuthHeader {
					m.jar.DeleteAuthHeader(host, entry.name)
				} else {
					m.jar.DeleteCookie(host, entry.name)
				}
				m.refreshJarList()
			}
			return m, nil
		case "D":
			m.jar.Clear(host)
			m.refreshJarList()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.jarList, cmd = m.jarList.Update(msg)
	return m, cmd
}

// JarViewerListView renders the saved-cookies/auth-headers viewer for
// app/view.go to place in the request pane's own box while it's open -
// see IsViewingJar - the same slot StylePickerListView already uses,
// mutually exclusive with it (both claim every key while open - see
// ResponsePane.Update).
func (m ResponsePane) JarViewerListView() string {
	host := m.currentHost()
	header := m.styles.Text.Label.Render(fmt.Sprintf("Saved cookies & auth for %s", host))
	hint := footerHelpStyle.Render("d: delete · D: clear all · esc: close · /: filter")
	return lipgloss.JoinVertical(lipgloss.Left, header, m.jarList.View(), hint)
}
