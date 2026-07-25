package responsepane

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestResponseTabNavigationKeys(t *testing.T) {
	tests := []struct {
		name  string
		start TabIndex
		key   tea.KeyPressMsg
		want  TabIndex
	}{
		{name: "l advances", start: TabBody, key: responseKeyPress('l', "l"), want: TabHeaders},
		{name: "h moves backward", start: TabTiming, key: responseKeyPress('h', "h"), want: TabHeaders},
		{name: "l wraps", start: TabTiming, key: responseKeyPress('l', "l"), want: TabBody},
		{name: "h wraps", start: TabBody, key: responseKeyPress('h', "h"), want: TabTiming},
		{name: "one selects body", start: TabTiming, key: responseKeyPress('1', "1"), want: TabBody},
		{name: "two selects headers", start: TabBody, key: responseKeyPress('2', "2"), want: TabHeaders},
		{name: "three selects timing", start: TabBody, key: responseKeyPress('3', "3"), want: TabTiming},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := SetupResponsePane(keybindings.DefaultKeyMap())
			pane.activeTab = int(tt.start)

			updated, _ := pane.Update(tt.key)

			if got := TabIndex(updated.activeTab); got != tt.want {
				t.Fatalf("active tab = %d, want %d", got, tt.want)
			}
		})
	}
}

func responseKeyPress(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Text: text})
}
