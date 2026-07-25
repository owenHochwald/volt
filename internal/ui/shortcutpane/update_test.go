package shortcutpane

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestHelpTabNavigationKeys(t *testing.T) {
	tests := []struct {
		name  string
		start TabIndex
		key   tea.KeyPressMsg
		want  TabIndex
	}{
		{name: "l advances", start: Global, key: helpKeyPress('l', "l"), want: Sidebar},
		{name: "h moves backward", start: Response, key: helpKeyPress('h', "h"), want: Request},
		{name: "l wraps", start: Response, key: helpKeyPress('l', "l"), want: Global},
		{name: "h wraps", start: Global, key: helpKeyPress('h', "h"), want: Response},
		{name: "one selects global", start: Response, key: helpKeyPress('1', "1"), want: Global},
		{name: "two selects sidebar", start: Global, key: helpKeyPress('2', "2"), want: Sidebar},
		{name: "three selects request", start: Global, key: helpKeyPress('3', "3"), want: Request},
		{name: "four selects response", start: Global, key: helpKeyPress('4', "4"), want: Response},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := SetupShortcutPane(keybindings.DefaultKeyMap())
			pane.activeTab = int(tt.start)

			updated, _ := pane.Update(tt.key)

			if got := TabIndex(updated.activeTab); got != tt.want {
				t.Fatalf("active tab = %d, want %d", got, tt.want)
			}
		})
	}
}

func helpKeyPress(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Text: text})
}
