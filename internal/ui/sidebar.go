package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

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

	db   *storage.SQLiteStorage
	keys keybindings.KeyMap
}

// helpHintLines is how many rows View()'s "Press ? for help" hint occupies
// (HelpStyle's Margin(1, 2) adds a blank line above and below the text
// itself), reserved out of the list's height so the two together fit
// within the space SetSize was given instead of overflowing it.
const helpHintLines = 3

func (s *SidebarPane) listHeight() int {
	return max(s.height-helpHintLines, 0)
}

// IsFiltering reports whether the user is actively typing a "/" search
// query, so callers outside this package (the global '?' help shortcut)
// can avoid stealing keys - notably '?' itself, which is a legitimate
// character to search for in a URL's query string - from the filter input.
func (s *SidebarPane) IsFiltering() bool {
	return s.requestsList.FilterState() == list.Filtering
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
	s.requestsList = list.New(items, list.NewDefaultDelegate(), s.width, s.listHeight())
	s.requestsList.SetShowHelp(false)
}

func (s *SidebarPane) Init() tea.Cmd {
	return LoadRequestsCmd(s.db)
}

func (s *SidebarPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case RequestsLoadingMsg:
		if msg.Err != nil {
			s.SetRequests([]list.Item{})
			s.requestsList.Title = "Saved (0)"
			return s, nil

		}
		// Best-effort: a group name is a display nicety, so a lookup
		// failure just falls back to showing no group label.
		groupNames := map[int64]string{}
		if groups, err := s.db.ListGroups(); err == nil {
			for _, g := range groups {
				groupNames[g.ID] = g.Name
			}
		}
		items := make([]list.Item, 0, len(msg.Requests))
		for _, req := range msg.Requests {
			var groupLabel string
			if req.GroupID != nil {
				if name, ok := groupNames[*req.GroupID]; ok {
					groupLabel = name
				}
			}
			items = append(items, RequestItem{
				title:   req.Name,
				desc:    s.describeURL(req.URL, groupLabel),
				group:   groupLabel,
				Request: &req,
			})
		}
		s.SetRequests(items)
		s.requestsList.Title = fmt.Sprintf("Saved (%d)", len(s.requestsList.Items()))

		if s.desiredCursorIndex >= 0 && len(items) > 0 {
			cursorPos := min(s.desiredCursorIndex, len(items)-1)
			s.requestsList.Select(cursorPos)
			s.desiredCursorIndex = -1
		}
		return s, nil
	case tea.KeyMsg:
		// While the user is actively typing a "/" filter query, every key
		// must go straight to the list's own filter text input instead of
		// being intercepted below - otherwise typing e.g. "json" would
		// trigger DeleteRequest ('d') or navigation ('j'/'k') on whatever's
		// currently selected instead of being typed into the search box.
		if s.requestsList.FilterState() == list.Filtering {
			break
		}
		if keybindings.Matches(msg, s.keys.DeleteRequest) {
			item, ok := s.SelectedItem()
			if !ok || item.Request == nil || item.Request.ID == 0 {
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
			if n := len(s.requestsList.VisibleItems()); n > 0 {
				next := s.requestsList.Index() - 1
				if next < 0 {
					next = n - 1
				}
				s.requestsList.Select(next)
			}
			return s, nil
		}
		if keybindings.Matches(msg, s.keys.NavDown) {
			if n := len(s.requestsList.VisibleItems()); n > 0 {
				next := s.requestsList.Index() + 1
				if next >= n {
					next = 0
				}
				s.requestsList.Select(next)
			}
			return s, nil
		}
	}

	s.requestsList, cmd = s.requestsList.Update(msg)

	return s, cmd
}

func (s *SidebarPane) View() string {
	helpText := HelpStyle.Render("/ to search • ? for help")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		s.requestsList.View(),
		helpText,
	)
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
	s.width = width
	s.height = height
	s.requestsList.SetSize(width, s.listHeight())
}

func NewSidebar(db *storage.SQLiteStorage, keys keybindings.KeyMap) *SidebarPane {
	loadingItems := []list.Item{
		RequestItem{
			title:   "Loading...",
			desc:    "Loading saved requests...",
			Request: nil,
		},
	}

	sidebar := &SidebarPane{
		panelFocused: false,
		height:       10,
		width:        10,
		db:           db,
		keys:         keys,
		requestsList: list.New(loadingItems, list.NewDefaultDelegate(), 0, 0),
	}
	sidebar.requestsList.Title = "Saved (Loading...)"
	sidebar.requestsList.SetShowHelp(false)

	return sidebar
}
