package keybindings

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Matches checks if a key message matches a binding
func Matches(msg tea.KeyPressMsg, binding key.Binding) bool {
	if !binding.Enabled() {
		return false
	}
	keystroke := msg.Keystroke()
	for _, registered := range binding.Keys() {
		if keystroke == registered {
			return true
		}
	}
	if msg.Mod.Contains(tea.ModCtrl) ||
		msg.Mod.Contains(tea.ModAlt) ||
		msg.Mod.Contains(tea.ModMeta) ||
		msg.Mod.Contains(tea.ModHyper) ||
		msg.Mod.Contains(tea.ModSuper) {
		return false
	}
	return key.Matches(msg, binding)
}
