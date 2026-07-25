package ui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestSidebarVimCountNavigation(t *testing.T) {
	tests := []struct {
		name  string
		items int
		start int
		keys  string
		want  int
	}{
		{name: "single j", items: 12, keys: "j", want: 1},
		{name: "ten down", items: 12, keys: "10j", want: 10},
		{name: "ten up wraps", items: 12, keys: "10k", want: 2},
		{name: "large count wraps", items: 12, start: 11, keys: "14j", want: 1},
		{name: "count resets after motion", items: 12, keys: "2jj", want: 3},
		{name: "unrelated key clears count", items: 12, keys: "2xj", want: 1},
		{name: "empty list remains safe", keys: "10j", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sidebar := NewSidebar(nil, keybindings.DefaultKeyMap())
			sidebar.SetRequests(sidebarTestItems(tt.items))
			if tt.items > 0 {
				sidebar.requestsList.Select(tt.start)
			}

			for _, input := range tt.keys {
				_, _ = sidebar.Update(sidebarKeyPress(input))
			}

			if got := sidebar.requestsList.Index(); got != tt.want {
				t.Fatalf("selected index = %d, want %d after %q", got, tt.want, tt.keys)
			}
		})
	}
}

func TestSidebarCommandTrailUsesReservedBottomRow(t *testing.T) {
	sidebar := NewSidebar(nil, keybindings.DefaultKeyMap())
	sidebar.SetRequests(sidebarTestItems(12))
	sidebar.SetSize(30, 20)

	for _, input := range "jjj10jk5k" {
		_, _ = sidebar.Update(sidebarKeyPress(input))
	}

	view := sidebar.View()
	if got := lipgloss.Height(view); got != 20 {
		t.Fatalf(
			"sidebar height = %d, want 20 (list = %d, trail = %q)",
			got,
			lipgloss.Height(sidebar.requestsList.View()),
			sidebar.commandTrail,
		)
	}
	if !strings.Contains(view, "j 10j k 5k") {
		t.Fatalf("sidebar command trail missing rolling commands: %q", view)
	}
}

func TestSidebarCommandTrailShowsPendingCountAndKeepsTenCharacters(t *testing.T) {
	sidebar := NewSidebar(nil, keybindings.DefaultKeyMap())
	sidebar.SetRequests(sidebarTestItems(12))
	sidebar.SetSize(30, 20)

	for _, input := range "jjj10j" {
		_, _ = sidebar.Update(sidebarKeyPress(input))
	}
	_, _ = sidebar.Update(sidebarKeyPress('5'))

	if view := sidebar.View(); !strings.Contains(view, "j j 10j 5") {
		t.Fatalf("pending count is not visible in command trail: %q", view)
	}

	_, _ = sidebar.Update(sidebarKeyPress('k'))
	if got := sidebar.commandTrail; got != "j j 10j 5k" {
		t.Fatalf("command trail = %q, want newest 10 characters", got)
	}
}

func sidebarTestItems(count int) []list.Item {
	items := make([]list.Item, count)
	for i := range items {
		items[i] = RequestItem{title: fmt.Sprintf("request-%d", i)}
	}
	return items
}

func sidebarKeyPress(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Text: string(code)})
}
