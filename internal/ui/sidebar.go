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

	db   *storage.SQLiteStorage
	keys keybindings.KeyMap
}

func (s *SidebarPane) SetRequests(items []list.Item) {
	s.requestsList = list.New(items, list.NewDefaultDelegate(), s.width, s.height)
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
	case tea.KeyMsg:
		if keybindings.Matches(msg, s.keys.DeleteRequest) {
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
			currentIndex := s.requestsList.Index()
			if currentIndex == 0 {
				s.requestsList.Select(len(s.requestsList.Items()) - 1)
			} else {
				s.requestsList.Select(currentIndex - 1)
			}
			return s, nil
		}
		if keybindings.Matches(msg, s.keys.NavDown) {
			currentIndex := s.requestsList.Index()
			itemCount := len(s.requestsList.Items()) - 1
			if currentIndex == itemCount {
				s.requestsList.Select(0)
			} else {
				s.requestsList.Select(currentIndex + 1)
			}
			return s, nil
		}
	}

	s.requestsList, cmd = s.requestsList.Update(msg)

	return s, cmd
}

func (s *SidebarPane) View() string {
	helpText := HelpStyle.Render("Press ? for help")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		s.requestsList.View(),
		lipgloss.NewStyle().Height(s.height-1).Render(""),
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
	s.requestsList.SetSize(width, height)
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
