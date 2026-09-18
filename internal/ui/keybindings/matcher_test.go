package keybindings

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func TestMatchesUsesStructuredV2KeyData(t *testing.T) {
	tests := []struct {
		name    string
		binding string
		key     tea.KeyPressMsg
		want    bool
	}{
		{
			name:    "legacy alt event without text",
			binding: "alt+l",
			key:     tea.KeyPressMsg(tea.Key{Code: 'l', Mod: tea.ModAlt}),
			want:    true,
		},
		{
			name:    "mac option event with associated text",
			binding: "alt+l",
			key:     tea.KeyPressMsg(tea.Key{Code: 'l', BaseCode: 'l', Text: "¬", Mod: tea.ModAlt}),
			want:    true,
		},
		{
			name:    "printable text without alt modifier",
			binding: "alt+l",
			key:     tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l"}),
			want:    false,
		},
		{
			name:    "modified text does not trigger plain binding",
			binding: "l",
			key:     tea.KeyPressMsg(tea.Key{Code: 'l', Text: "l", Mod: tea.ModAlt}),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binding := key.NewBinding(key.WithKeys(tt.binding))
			if got := Matches(tt.key, binding); got != tt.want {
				t.Fatalf("Matches(%q) = %t, want %t", tt.key.Keystroke(), got, tt.want)
			}
		})
	}
}
