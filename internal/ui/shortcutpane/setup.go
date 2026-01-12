package shortcutpane

import "github.com/owenHochwald/Volt/internal/ui/keybindings"

// SetupShortcutPane sets up the shortcut pane for use
func SetupShortcutPane(keys keybindings.KeyMap) ShortcutPane {
	return ShortcutPane{
		activeTab: 0,
		height:    30,
		width:     40,
		Focused:   false,
		keys:      keys,
		tabs:      GetShortcutTabs(keys),
	}

}
