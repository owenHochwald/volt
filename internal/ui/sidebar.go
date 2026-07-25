package ui

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

const sidebarCommandWindow = 10

type RequestItem struct {
	title, desc string
	Request     *http.Request
}

func (i RequestItem) Title() string       { return i.title }
func (i RequestItem) Description() string { return i.desc }
func (i RequestItem) FilterValue() string { return i.title }

type SidebarPane struct {
	panelFocused  bool
	height, width int

	requestsList    list.Model
	selectedRequest *RequestItem

	desiredCursorIndex int
	navigationCount    int
	hasNavigationCount bool
	pendingCommand     string
	commandTrail       string

	db     *storage.SQLiteStorage
	keys   keybindings.KeyMap
	styles design.Styles
}

func (s *SidebarPane) SetRequests(items []list.Item) {
	s.requestsList = list.New(items, list.NewDefaultDelegate(), s.width, s.listHeight())
	s.requestsList.SetShowHelp(false)
	s.requestsList.DisableQuitKeybindings()
	s.clearNavigationCount()
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

func (s *SidebarPane) Update(msg tea.Msg) (*SidebarPane, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case RequestsLoadingMsg:
		if msg.Err != nil {
			s.SetRequests([]list.Item{})
			s.requestsList.Title = "Saved (0)"
			return s, nil

		}
		items := make([]list.Item, 0, len(msg.Requests))
		for _, req := range msg.Requests {
			items = append(items, RequestItem{
				title:   req.Name,
				desc:    req.URL[max(len(req.URL)-10, 0):],
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
	case tea.KeyPressMsg:
		if keybindings.Matches(msg, s.keys.NavCount) {
			s.pendingCommand += string(msg.Code)
			s.appendNavigationDigit(msg.Code)
			return s, nil
		}

		if keybindings.Matches(msg, s.keys.DeleteRequest) {
			s.recordCommand("d")
			s.clearNavigationCount()
			item, ok := s.SelectedItem()
			if !ok || item.Request == nil || item.Request.ID == 0 {
				return s, nil
			}

			currentIndex := s.requestsList.Index()
			itemCount := len(s.requestsList.Items())
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

		// Navigation override - wrapped to cycle
		if keybindings.Matches(msg, s.keys.NavUp) {
			s.recordCommand(s.pendingCommand + "k")
			s.moveSelection(-s.consumeNavigationCount())
			return s, nil
		}
		if keybindings.Matches(msg, s.keys.NavDown) {
			s.recordCommand(s.pendingCommand + "j")
			s.moveSelection(s.consumeNavigationCount())
			return s, nil
		}
		if keybindings.Matches(msg, s.keys.NavFirst) {
			s.recordCommand("g")
			s.clearNavigationCount()
			s.requestsList.Select(0)
			return s, nil
		}
		if keybindings.Matches(msg, s.keys.NavLast) {
			s.recordCommand("G")
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

func (s *SidebarPane) appendNavigationDigit(code rune) {
	digit := int(code - '0')
	if digit < 0 || digit > 9 {
		s.clearNavigationCount()
		return
	}
	s.hasNavigationCount = true
	itemCount := len(s.requestsList.Items())
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
	s.pendingCommand = ""
}

func (s *SidebarPane) recordCommand(command string) {
	if command == "" {
		return
	}
	if s.commandTrail != "" {
		s.commandTrail += " "
	}
	s.commandTrail += command
	s.commandTrail = trailingRunes(s.commandTrail, sidebarCommandWindow)
}

func trailingRunes(value string, limit int) string {
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	return string(runes[len(runes)-limit:])
}

func (s *SidebarPane) moveSelection(delta int) {
	itemCount := len(s.requestsList.Items())
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
	trail := s.commandTrail
	if s.pendingCommand != "" {
		if trail != "" {
			trail += " "
		}
		trail += s.pendingCommand
		trail = trailingRunes(trail, sidebarCommandWindow)
	}

	commandLine := s.styles.Text.Muted.Render("›")
	if trail != "" {
		commandLine = lipgloss.JoinHorizontal(
			lipgloss.Left,
			commandLine,
			" ",
			s.styles.Action.Primary.Render(trail),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left, s.requestsList.View(), commandLine)
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
	s.requestsList.SetSize(s.width, s.listHeight())
}

func (s *SidebarPane) listHeight() int {
	return max(s.height-1, 1)
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

	sidebar := &SidebarPane{
		panelFocused: false,
		height:       10,
		width:        10,
		db:           db,
		keys:         keys,
		styles:       styles,
		requestsList: list.New(loadingItems, list.NewDefaultDelegate(), 0, 0),
	}
	sidebar.requestsList.Title = "Saved (Loading...)"
	sidebar.requestsList.SetShowHelp(false)
	sidebar.requestsList.DisableQuitKeybindings()

	return sidebar
}
