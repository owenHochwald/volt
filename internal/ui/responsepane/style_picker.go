package responsepane

import (
	"fmt"
	"sort"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/owenHochwald/Volt/internal/ui"
)

// stylePickerChromeLines is the header + hint line StylePickerListView
// wraps the list itself in - subtracted from stylePickerHeight so the
// list's own height doesn't overflow the box it's rendered into (see
// resizeStyleList).
const stylePickerChromeLines = 2

// styleItem adapts a chroma style name to list.Item for styleList below.
type styleItem string

func (s styleItem) Title() string       { return string(s) }
func (s styleItem) Description() string { return "" }
func (s styleItem) FilterValue() string { return string(s) }

// allChromaStyleNames returns every style chroma ships (see
// chroma/v2/styles.Names), sorted for a stable, predictable list order.
func allChromaStyleNames() []string {
	names := append([]string(nil), chromastyles.Names()...)
	sort.Strings(names)
	return names
}

// newStyleList builds the picker's list, fuzzy-filterable via list's own
// built-in "/" (the same convention the sidebar already uses), cursor
// starting on current. DisableQuitKeybindings matters here the same way
// it does for the sidebar's own list - bubbles' list.Model otherwise
// treats a bare "q"/esc (outside of filtering) as "quit the whole
// program", which cancelling the picker (esc, in handleStylePickerKey)
// only half-guards against (it only ever sees esc, never a stray "q").
func newStyleList(current string) list.Model {
	names := allChromaStyleNames()
	items := make([]list.Item, len(names))
	selected := 0
	for i, n := range names {
		items[i] = styleItem(n)
		if n == current {
			selected = i
		}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)

	l := list.New(items, delegate, 0, 0)
	l.Title = "Body Style"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()
	l.Select(selected)
	return l
}

// openStylePicker opens the "t" picker (see ResponsePane.pickingStyle) -
// nothing is applied or persisted until confirmStylePicker (enter); the
// live preview shown while browsing (see effectiveBodyStyle, used by
// updateViewportForActiveTab) is purely a function of wherever the list's
// cursor currently sits, so cancelling (esc) has nothing to undo. Sizes
// the list from stylePickerWidth/Height (see SetStylePickerSize), the
// request pane's own box dimensions - already known by the time any key
// could have opened this, since a WindowSizeMsg always arrives before
// any user input does.
func (m *ResponsePane) openStylePicker() {
	m.pickingStyle = true
	m.styleList = newStyleList(m.bodyStyleName)
	m.resizeStyleList()
	m.updateViewportForActiveTab()
}

// resizeStyleList sizes styleList to stylePickerWidth/Height (see
// SetStylePickerSize) minus stylePickerChromeLines - called on open and
// whenever the pane itself is resized while the picker is open.
func (m *ResponsePane) resizeStyleList() {
	height := max(m.stylePickerHeight-stylePickerChromeLines, 3)
	m.styleList.SetSize(max(m.stylePickerWidth, 1), height)
}

// handleStylePickerKey handles input while the style picker is open -
// esc cancels (only outside of the list's own "/" filter-typing mode,
// which esc should just clear instead - see the filtering guard below),
// enter confirms and persists the selected style (see confirmStylePicker,
// same guard), anything else is forwarded to styleList (including its
// own "/" filter and cursor movement), after which the live preview is
// refreshed to match wherever the cursor now sits (see
// effectiveBodyStyle).
func (m *ResponsePane) handleStylePickerKey(msg tea.KeyPressMsg) (*ResponsePane, tea.Cmd) {
	filtering := m.styleList.FilterState() == list.Filtering
	if msg.String() == "esc" && !filtering {
		m.pickingStyle = false
		m.updateViewportForActiveTab()
		return m, nil
	}
	if msg.String() == "enter" && !filtering {
		return m, m.confirmStylePicker()
	}

	var cmd tea.Cmd
	m.styleList, cmd = m.styleList.Update(msg)
	m.updateViewportForActiveTab()
	return m, cmd
}

// confirmStylePicker applies the list's currently selected style,
// re-renders the real content with it, and persists it (see
// design.SaveBodyStyle) so the next app start restores it too.
func (m *ResponsePane) confirmStylePicker() tea.Cmd {
	if it, ok := m.styleList.SelectedItem().(styleItem); ok {
		m.bodyStyleName = string(it)
	}
	m.pickingStyle = false
	m.updateViewportForActiveTab()
	return ui.SaveBodyStyleCmd(m.bodyStyleName)
}

// effectiveBodyStyle is what formatContentByType should actually
// highlight with right now: whatever the picker's list cursor currently
// sits on while it's open (a live preview through this pane's own normal
// Body rendering - already safe with arbitrarily long real response
// lines, unlike a separate hand-rolled preview snippet would be),
// otherwise the applied bodyStyleName.
func (m *ResponsePane) effectiveBodyStyle() string {
	if m.pickingStyle {
		if it, ok := m.styleList.SelectedItem().(styleItem); ok {
			return string(it)
		}
	}
	return m.bodyStyleName
}

// StylePickerListView renders the style picker for app/view.go to place
// in the request pane's own box while it's open (see IsPickingStyle) -
// this pane's own View() keeps rendering normally alongside it, live-
// previewing whatever the list's cursor sits on (see effectiveBodyStyle).
func (m ResponsePane) StylePickerListView() string {
	header := m.styles.Text.Label.Render(fmt.Sprintf("Body syntax style (current: %s)", m.bodyStyleName))
	hint := footerHelpStyle.Render("enter: apply & save · esc: cancel · /: filter")
	return lipgloss.JoinVertical(lipgloss.Left, header, m.styleList.View(), hint)
}
