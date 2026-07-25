package ui

import (
	"fmt"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
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
