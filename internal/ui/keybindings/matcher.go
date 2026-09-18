package keybindings

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Matches checks if a key message matches a binding
func Matches(msg tea.KeyPressMsg, binding key.Binding) bool {
	return key.Matches(msg, binding)
}
