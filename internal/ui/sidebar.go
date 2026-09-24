package ui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/paginator"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// HelpStyle wraps the group filter/help line with the same
// margin the rest of the sidebar's chrome used to share (see the removed
// internal/ui/styles.go) - purely layout, not a themed color, so it stays
// a plain local style rather than moving into design.Styles.
var HelpStyle = lipgloss.NewStyle().Margin(1, 2)

type RequestItem struct {
	title, desc, group string
	Request            *http.Request
}

func (i RequestItem) Title() string       { return i.title }
func (i RequestItem) Description() string { return i.desc }

// FilterValue is what "/" search matches against. It deliberately uses the
// request's full, untruncated method+URL (not desc, which is cut down to
// fit the sidebar's width) so searching for something that only appears in
// the truncated-away part of a long URL still finds it - plus the name and
// group, so any of those can be searched by.
func (i RequestItem) FilterValue() string {
	value := i.title
	if i.Request != nil {
		value += " " + i.Request.Method + " " + i.Request.URL
	}
	if i.group != "" {
		value += " " + i.group
	}
	return value
}

type SidebarPane struct {
	panelFocused  bool
	height, width int

	requestsList    list.Model
	selectedRequest *RequestItem

	desiredCursorIndex int
	navigationCount    int
	hasNavigationCount bool

	// allRequests/groupNames hold the last load's full, unfiltered request
	// set and its group-id-to-name lookup, kept around so cycling the group
	// filter (see groupOptions) can re-derive requestsList without a fresh
	// database round-trip.
	allRequests []http.Request
	groupNames  map[int64]string

	// groupOptions is the group filter's cycle: index 0 is always
	// allGroupsOption (show every request), each following entry one
	// group's name, sorted. Cycled with "[" / "]" and shown as the
	// sidebar's first line, above the request list - so a workspace with
	// many saved requests across groups can narrow the list down instead
	// of always scrolling through everything at once.
	groupOptions     []string
	groupFilterIndex int

	// pendingGroupFilterName is the group filter saved from a previous
	// session (see storage.LoadLastGroupFilter), loaded once at
	// construction - groupOptions doesn't exist yet at that point (it's
	// only built once the database's groups are loaded - see
	// RequestsLoadingMsg), so this just remembers the *name* to look for
	// once it does. restoredGroupFilter tracks whether that's already
	// happened, since RequestsLoadingMsg can fire more than once per
	// session (e.g. after a save/delete/import) and only the very first
	// one should pull from disk - every later one already has the user's
	// own current-session selection to preserve instead (see
	// selectedGroupName in RequestsLoadingMsg's handling).
	pendingGroupFilterName string
	restoredGroupFilter    bool

	// pendingDelete holds the item awaiting a y/n confirmation after
	// DeleteRequest ('d') was pressed, or nil when nothing is pending.
	// Deleting a saved request is irreversible, so it's confirmed rather
	// than acted on immediately.
	pendingDelete *RequestItem

	db     *storage.SQLiteStorage
	keys   keybindings.KeyMap
	styles design.Styles
}

// allGroupsOption is the group filter's choice for "show every request,
// regardless of group" - always groupOptions[0].
const allGroupsOption = "All groups"

// contentWidth is the width available for the sidebar's own single-line
// content (the group filter line, and the help hint/delete-confirmation
// line below the list) - used to explicitly *cap* that text to the
// sidebar's actual rendered width via MaxWidth (which truncates - unlike
// Width, which wraps to multiple lines). Without this, these lines
// rendered at their natural (untruncated) length - fine as long as they
// happened to be short, but the help hint is long enough to definitely
// overflow at the sidebar's minimum width (20). Left unconstrained here,
// that overflow still got resolved eventually, just one layer up, when
// app/view.go's outer SidebarStyle.Width(...) call *wrapped* the too-wide
// content to multiple lines - at a width listHeight()'s line-count math
// knew nothing about, so the extra wrapped lines silently pushed the
// sidebar's total rendered height past what SetSize had reserved for it,
// overflowing its box (and, since nothing then constrained *that*,
// sometimes the whole terminal).
func (s *SidebarPane) contentWidth() int {
	return max(s.width-4, 1) // matches HelpStyle's Margin(1, 2) - 2 cols each side
}

// listWidth returns the width to hand to the requests list itself
// (list.New / SetSize), compensating for a quirk in bubbles' own
// list.Model.titleView(): it truncates the title bar's content to
// m.width-spinnerWidth (spinnerWidth==1, reserved even when the spinner is
// hidden), then wraps that in TitleBar's style, which adds 2 cols of left
// padding back - so the title bar's true rendered width ends up m.width+1,
// one column wider than requested. Left uncompensated, the outer box (which
// *is* constrained to s.width) re-wraps that one line to a second line,
// inflating the sidebar's total rendered height past what listHeight()
// reserved for it. Passing s.width-1 here cancels that out exactly.
func (s *SidebarPane) listWidth() int {
	return max(s.width-1, 1)
}

// renderGroupLine renders the group filter's "[/] ‹ current ›" line,
// capped to contentWidth (groupFilterStyle carries no margin of its own,
// unlike HelpStyle, so no extra adjustment needed there) so it can never
// silently grow past the single line listHeight() reserves for it.
func (s *SidebarPane) renderGroupLine() string {
	current := allGroupsOption
	if s.groupFilterIndex >= 0 && s.groupFilterIndex < len(s.groupOptions) {
		current = s.groupOptions[s.groupFilterIndex]
	}
	return groupFilterStyle.MaxWidth(s.width).Render(fmt.Sprintf("[/] ‹ %s ›", current))
}

// renderHelpText renders the sidebar's pinned bottom line - the normal
// search/filter hint, or (while a delete is pending) the y/n confirmation
// prompt - capped to contentWidth for the same reason as renderGroupLine.
func (s *SidebarPane) renderHelpText() string {
	if s.pendingDelete != nil {
		return HelpStyle.MaxWidth(s.width).Render(
			lipgloss.NewStyle().Foreground(lipgloss.Color("196")).MaxWidth(s.contentWidth()).
				Render(fmt.Sprintf("Delete %q? y/enter=yes, any other key=no", s.pendingDelete.title)),
		)
	}
	return HelpStyle.MaxWidth(s.width).Render("ctrl+f: filter • [/]: group filter • ? for help")
}

// listHeight returns the height available for the request list itself,
// reserving exactly as many rows as renderGroupLine/renderHelpText/
// renderSelectedDescriptionLine actually render to at the sidebar's
// current width (see contentWidth) - not a fixed guess - so all of them
// together always add up to s.height exactly, regardless of how narrow
// the sidebar gets or whether the selected request even has a
// description to show.
func (s *SidebarPane) listHeight() int {
	reserved := lipgloss.Height(s.renderGroupLine()) + lipgloss.Height(s.renderHelpText())
	if desc := s.renderSelectedDescriptionLine(); desc != "" {
		reserved += lipgloss.Height(desc)
	}
	return max(s.height-reserved, 0)
}

// renderSelectedDescriptionLine renders the currently selected request's
// OpenAPI operation description (see http.Request.Description), or ""
// if it has none - a hand-typed request, or one imported from a spec
// that didn't declare one, takes up no extra space at all. Wrapped
// (not just truncated to one line) since a real description can run to a
// full sentence or two, but capped at maxDescriptionLines so one long
// enough to matter can't quietly swallow most of the sidebar's height.
func (s *SidebarPane) renderSelectedDescriptionLine() string {
	item, ok := s.SelectedItem()
	if !ok || item.Request == nil || item.Request.Description == "" {
		return ""
	}
	const maxDescriptionLines = 2
	wrapped := lipgloss.NewStyle().Width(max(s.width-2, 10)).Render(item.Request.Description)
	lines := strings.Split(wrapped, "\n")
	if len(lines) > maxDescriptionLines {
		lines = lines[:maxDescriptionLines]
		lines[maxDescriptionLines-1] = strings.TrimRight(lines[maxDescriptionLines-1], " ") + "…"
	}
	return s.styles.Text.Faint.Render(strings.Join(lines, "\n"))
}

// currentGroupFilter returns the selected group's name, or "" when
// groupOptions[0] (allGroupsOption) is selected - meaning don't filter.
func (s *SidebarPane) currentGroupFilter() string {
	if s.groupFilterIndex <= 0 || s.groupFilterIndex >= len(s.groupOptions) {
		return ""
	}
	return s.groupOptions[s.groupFilterIndex]
}

// saveGroupFilterCmd persists the current group filter (see
// currentGroupFilter/SaveLastGroupFilterCmd) so the next app start
// restores it - called every time "[" / "]" changes the selection,
// including back to "" ("All groups" is itself a deliberately saved
// choice, not the absence of one - see storage.SaveLastGroupFilter). A
// no-op Cmd when there's no database (some test setups construct a
// SidebarPane with a nil db).
func (s *SidebarPane) saveGroupFilterCmd() tea.Cmd {
	if s.db == nil {
		return nil
	}
	return SaveLastGroupFilterCmd(s.db, s.currentGroupFilter())
}

// cycleGroupFilter moves the group filter by delta (wrapping) and rebuilds
// the visible request list to match.
func (s *SidebarPane) cycleGroupFilter(delta int) {
	n := len(s.groupOptions)
	if n == 0 {
		return
	}
	s.groupFilterIndex = ((s.groupFilterIndex+delta)%n + n) % n
	s.rebuildItems()
}

// rebuildItems re-derives requestsList from allRequests/groupNames under the
// current group filter, without needing a fresh database load - called both
// after a database reload and whenever the group filter itself changes.
func (s *SidebarPane) rebuildItems() {
	filterGroup := s.currentGroupFilter()
	items := make([]list.Item, 0, len(s.allRequests))
	for _, req := range s.allRequests {
		var groupLabel string
		if req.GroupID != nil {
			if name, ok := s.groupNames[*req.GroupID]; ok {
				groupLabel = name
			}
		}
		if filterGroup != "" && groupLabel != filterGroup {
			continue
		}
		items = append(items, RequestItem{
			title:   req.Name,
			desc:    s.describeURL(req.URL, groupLabel),
			group:   groupLabel,
			Request: &req,
		})
	}
	s.SetRequests(items)
	s.requestsList.Title = fmt.Sprintf("Saved (%d)", len(items))
}

// IsFiltering reports whether the user is actively typing a "/" search
// query, so callers outside this package (the global '?' help shortcut)
// can avoid stealing keys - notably '?' itself, which is a legitimate
// character to search for in a URL's query string - from the filter input.
func (s *SidebarPane) IsFiltering() bool {
	return s.requestsList.FilterState() == list.Filtering
}

// WantsEscForItself reports whether esc means something to the sidebar
// right now, other than the app's global quit-arming (see app/update.go's
// overlayOpen) - true while either:
//
//   - actively typing a filter query (see IsFiltering) - bubbles' own
//     list.Model already handles esc correctly here (list.KeyMap's
//     CancelWhileFiltering backs out of typing, discarding the query),
//     reached via the plain s.requestsList.Update(msg) at the bottom of
//     Update.
//   - A delete confirmation is pending (see pendingDelete) - Update's own
//     "any key but y/Y/enter cancels" already treats esc as a cancel.
//
// Deliberately does NOT include an already-*applied* filter (not being
// typed) - clearing that moved off esc entirely, onto ctrl+x (see
// configureFilterKeys), specifically so esc can't wipe it by accident;
// esc while merely browsing a filtered list now falls through to the
// app's ordinary quit-arming/panel-switch behavior instead, same as with
// no filter active at all.
func (s *SidebarPane) WantsEscForItself() bool {
	return s.IsFiltering() || s.pendingDelete != nil
}

// describeURL builds an item's description line, filling as much of the
// available sidebar width as the URL needs instead of always truncating
// to a fixed handful of characters - a wider sidebar (see Model.sidebarWidth)
// should actually show more of the URL, not leave the extra space blank.
// When it doesn't fit, the tail (usually the more distinguishing part -
// the path, not the shared scheme/host) is kept, with a leading "…".
func (s *SidebarPane) describeURL(url, groupLabel string) string {
	prefix := ""
	if groupLabel != "" {
		prefix = "[" + groupLabel + "] "
	}

	// Leave a little room for the list item's own border/padding so the
	// line doesn't wrap.
	const chrome = 4
	budget := max(s.width-len(prefix)-chrome, 10)

	if len(url) > budget {
		url = "…" + url[len(url)-(budget-1):]
	}
	return prefix + url
}

func (s *SidebarPane) SetRequests(items []list.Item) {
	s.requestsList = list.New(items, list.NewDefaultDelegate(), s.listWidth(), s.listHeight())
	s.requestsList.SetShowHelp(false)
	s.requestsList.DisableQuitKeybindings()
	// list.New defaults its paginator to dots - "10/15" (as main used to
	// show) is more useful once there are enough saved requests to paginate.
	s.requestsList.Paginator.Type = paginator.Arabic
	configureFilterKeys(&s.requestsList)
	s.clearNavigationCount()
}

// configureFilterKeys remaps list.Model's own default filter keybindings -
// starting a filter ("/") and clearing an already-applied one (esc) - away
// from their defaults, on a freshly (re)constructed list (see SetRequests/
// NewSidebar, both of which build a brand new list.Model from scratch, so
// this has to be reapplied every time, not just once at startup).
//
// "/" moved to ctrl+f: reported as confusing alongside two other things
// that also mention "/" for something else entirely - the response
// panel's own real regex search (also bound to "/", a genuinely different
// feature, not just a differently-named version of this filter), and this
// same line's own "[/]: group filter" hint (bracket keys, unrelated to the
// bare "/" this used to also claim right next to it).
//
// ClearFilter moved to ctrl+x: esc is the universal back/cancel key
// everywhere else in Volt, easy to hit on reflex while just browsing an
// already-filtered list - wiping the filter with no request to. Esc still
// cancels *typing* one (list.KeyMap's CancelWhileFiltering, untouched
// here) - only clearing one that's already applied and no longer being
// edited moves off esc.
func configureFilterKeys(l *list.Model) {
	l.KeyMap.Filter = key.NewBinding(key.WithKeys("ctrl+f"), key.WithHelp("ctrl+f", "filter"))
	l.KeyMap.ClearFilter = key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "clear filter"))
}

func (s *SidebarPane) Init() tea.Cmd {
	return LoadRequestsCmd(s.db)
}

func (s *SidebarPane) SetFocused(focused bool) {
	s.panelFocused = focused
	if !focused {
		s.clearNavigationCount()
	}
}

func (s *SidebarPane) SetStyles(styles design.Styles) {
	s.styles = styles
}

func (s *SidebarPane) Update(msg tea.Msg) (*SidebarPane, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case RequestsLoadingMsg:
		if msg.Err != nil {
			s.allRequests = nil
			s.SetRequests([]list.Item{})
			s.requestsList.Title = "Saved (0)"
			return s, nil

		}
		// Captured before groupOptions is rebuilt below, so the selected
		// group can be found again by *name* afterwards - reusing the old
		// index directly would silently point at a different group if the
		// set of groups shrank or reordered (e.g. the selected one was
		// deleted), the same class of bug a plain bounds check can't catch.
		selectedGroupName := s.currentGroupFilter()

		// The very first load of a session has nothing selected yet
		// (selectedGroupName == ""), so this is where a filter restored
		// from a previous session (see pendingGroupFilterName) gets its
		// one chance to apply - every later reload this session already
		// has the user's own current selection to preserve above instead,
		// which must win over a since-stale value from disk.
		if !s.restoredGroupFilter {
			s.restoredGroupFilter = true
			if selectedGroupName == "" && s.pendingGroupFilterName != "" {
				selectedGroupName = s.pendingGroupFilterName
			}
		}

		// Best-effort: a group name is a display nicety, so a lookup
		// failure just falls back to showing no group label and no filter
		// options beyond allGroupsOption.
		groupNames := map[int64]string{}
		var groupNamesSorted []string
		if groups, err := s.db.ListGroups(); err == nil {
			for _, g := range groups {
				groupNames[g.ID] = g.Name
				groupNamesSorted = append(groupNamesSorted, g.Name)
			}
		}
		sort.Strings(groupNamesSorted)
		s.groupNames = groupNames
		s.groupOptions = append([]string{allGroupsOption}, groupNamesSorted...)

		s.groupFilterIndex = 0
		if selectedGroupName != "" {
			for i, name := range s.groupOptions {
				if name == selectedGroupName {
					s.groupFilterIndex = i
					break
				}
			}
		}

		s.allRequests = msg.Requests
		s.rebuildItems()

		visibleCount := len(s.requestsList.Items())
		if s.desiredCursorIndex >= 0 && visibleCount > 0 {
			cursorPos := min(s.desiredCursorIndex, visibleCount-1)
			s.requestsList.Select(cursorPos)
			s.desiredCursorIndex = -1
		}
		return s, nil
	case tea.KeyPressMsg:
		// A pending delete swallows the very next key as its y/n answer,
		// checked before anything else (including the filter-typing guard
		// right below) - while confirming, that's the only thing any key
		// should mean.
		if s.pendingDelete != nil {
			item := s.pendingDelete
			s.pendingDelete = nil
			if msg.String() != "y" && msg.String() != "Y" && msg.String() != "enter" {
				return s, nil
			}

			currentIndex := s.requestsList.Index()
			itemCount := len(s.requestsList.VisibleItems())
			if itemCount > 1 {
				if currentIndex == itemCount-1 {
					s.desiredCursorIndex = currentIndex - 1
				} else {
					s.desiredCursorIndex = currentIndex
				}
			} else {
				s.desiredCursorIndex = 0
			}
			return s, DeleteRequestCmd(s.db, item.Request.ID)
		}

		if keybindings.Matches(msg, s.keys.NavCount) {
			s.appendNavigationDigit(msg.Code)
			return s, nil
		}

		// While the user is actively typing a "/" filter query, every key
		// must go straight to the list's own filter text input instead of
		// being intercepted below - otherwise typing e.g. "json" would
		// trigger DeleteRequest ('d') or navigation ('j'/'k') on whatever's
		// currently selected instead of being typed into the search box.
		if s.requestsList.FilterState() == list.Filtering {
			break
		}
		if keybindings.Matches(msg, s.keys.DeleteRequest) {
			s.clearNavigationCount()
			item, ok := s.SelectedItem()
			if !ok || item.Request == nil || item.Request.ID == 0 {
				return s, nil
			}
			// Deleting is irreversible - confirm before actually doing it
			// (see pendingDelete above), rather than deleting on the spot.
			s.pendingDelete = &item
			return s, nil
		}
		// "[" / "]" rather than h/l/left/right: those are also bubbles'
		// list.Model's own default PrevPage/NextPage keys (along with
		// pgup/pgdown/b/f/u/d), so binding the group filter to them would
		// silently steal the list's own page-by-page scrolling out from
		// under it - exactly the regression this replaced.
		if msg.String() == "[" {
			s.cycleGroupFilter(-1)
			return s, s.saveGroupFilterCmd()
		}
		if msg.String() == "]" {
			s.cycleGroupFilter(1)
			return s, s.saveGroupFilterCmd()
		}

		// Navigation override - wrapped to cycle. Two things bubbles'
		// list.Select() doesn't do on its own, both handled below:
		//
		//  - Index()/Select() operate on the currently visible (i.e.
		//    filtered, when a filter is active) list, so the wrap bound
		//    must come from VisibleItems(), not Items() (always the full,
		//    unfiltered set) - using the latter selected an index past the
		//    end of a filtered list and crashed on render.
		//  - Select(index) does *unclamped* arithmetic (page = index /
		//    perPage, cursor = index % perPage) - given an out-of-range
		//    index, it leaves the cursor pointing past the real items
		//    instead of rejecting it, and Index() keeps faithfully
		//    reporting that same bad value back afterwards. A wrap check
		//    written as current == bound therefore never fires again once
		//    the cursor has drifted out of range this way - each further
		//    "down" press just drifts it further, and it takes exactly as
		//    many "up" presses to walk back into valid range (or, far
		//    enough out, something downstream indexes past the actual
		//    slice and panics). Using >=/< instead of == self-corrects in
		//    a single press no matter how far off the cursor already is.
		if keybindings.Matches(msg, s.keys.NavUp) {
			s.moveSelection(-s.consumeNavigationCount())
			return s, nil
		}
		if keybindings.Matches(msg, s.keys.NavDown) {
			s.moveSelection(s.consumeNavigationCount())
			return s, nil
		}
		if keybindings.Matches(msg, s.keys.NavFirst) {
			s.clearNavigationCount()
			s.requestsList.Select(0)
			return s, nil
		}
		if keybindings.Matches(msg, s.keys.NavLast) {
			s.clearNavigationCount()
			if itemCount := len(s.requestsList.Items()); itemCount > 0 {
				s.requestsList.Select(itemCount - 1)
			}
			return s, nil
		}
		s.clearNavigationCount()
	}

	s.requestsList, cmd = s.requestsList.Update(msg)

	return s, cmd
}

// groupFilterStyle renders the sidebar's group filter line - the first
// thing shown, above the request list itself, so it reads as a field you
// pick before the list it narrows down.
var groupFilterStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

// appendNavigationDigit/consumeNavigationCount/moveSelection use
// VisibleItems(), not Items() (always the full, unfiltered set) - same
// reasoning as NavUp/NavDown above: bounding a count or a wrap against the
// unfiltered count can point past the end of an active "/" filter's
// narrower list and crash on render.
func (s *SidebarPane) appendNavigationDigit(code rune) {
	digit := int(code - '0')
	if digit < 0 || digit > 9 {
		s.clearNavigationCount()
		return
	}
	s.hasNavigationCount = true
	itemCount := len(s.requestsList.VisibleItems())
	if itemCount == 0 {
		s.navigationCount = 0
		return
	}
	s.navigationCount = (s.navigationCount*10 + digit) % itemCount
}

func (s *SidebarPane) consumeNavigationCount() int {
	if !s.hasNavigationCount {
		return 1
	}
	count := s.navigationCount
	s.clearNavigationCount()
	return count
}

func (s *SidebarPane) clearNavigationCount() {
	s.navigationCount = 0
	s.hasNavigationCount = false
}

func (s *SidebarPane) moveSelection(delta int) {
	itemCount := len(s.requestsList.VisibleItems())
	if itemCount == 0 {
		return
	}
	next := (s.requestsList.Index() + delta) % itemCount
	if next < 0 {
		next += itemCount
	}
	s.requestsList.Select(next)
}

func (s *SidebarPane) View() string {
	// "[" / "]" work no matter where the list cursor is - there's no
	// separate "focus" to move back to - but that's not obvious from the
	// brackets alone, so the hint is spelled out in renderHelpText too, not
	// just in the group line itself (which can be easy to miss).
	lines := []string{s.renderGroupLine(), s.requestsList.View()}
	if desc := s.renderSelectedDescriptionLine(); desc != "" {
		lines = append(lines, desc)
	}
	lines = append(lines, s.renderHelpText())
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (s *SidebarPane) SelectedItem() (RequestItem, bool) {
	if item := s.requestsList.SelectedItem(); item != nil {
		if reqItem, ok := item.(RequestItem); ok {
			return reqItem, true
		}
	}
	return RequestItem{}, false
}

func (s *SidebarPane) SetSize(width, height int) {
	s.width = max(width, 1)
	s.height = max(height, 1)
	s.requestsList.SetSize(s.listWidth(), s.listHeight())
}

func NewSidebar(db *storage.SQLiteStorage, keys keybindings.KeyMap, optionalStyles ...design.Styles) *SidebarPane {
	styles := design.NewStyles(design.DefaultTheme())
	if len(optionalStyles) > 0 {
		styles = optionalStyles[0]
	}

	loadingItems := []list.Item{
		RequestItem{
			title:   "Loading...",
			desc:    "Loading saved requests...",
			Request: nil,
		},
	}

	var pendingGroupFilterName string
	if db != nil {
		pendingGroupFilterName, _ = db.LoadLastGroupFilter()
	}

	sidebar := &SidebarPane{
		panelFocused:           false,
		height:                 10,
		width:                  10,
		db:                     db,
		keys:                   keys,
		styles:                 styles,
		requestsList:           list.New(loadingItems, list.NewDefaultDelegate(), 0, 0),
		groupOptions:           []string{allGroupsOption},
		pendingGroupFilterName: pendingGroupFilterName,
	}
	sidebar.requestsList.Title = "Saved (Loading...)"
	sidebar.requestsList.SetShowHelp(false)
	sidebar.requestsList.DisableQuitKeybindings()
	sidebar.requestsList.Paginator.Type = paginator.Arabic
	configureFilterKeys(&sidebar.requestsList)

	return sidebar
}
