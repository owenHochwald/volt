package keybindings

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Matches checks if a key message matches a binding
func Matches(msg tea.KeyMsg, binding key.Binding) bool {
	return key.Matches(msg, binding)
}
