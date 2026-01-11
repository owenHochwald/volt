package shortcutpane

import "github.com/owenHochwald/volt/internal/ui/keybindings"

// Shortcut represents a keyboard shortcut and its description
type Shortcut struct {
	Key         string
	Description string
}

// ShortcutTab represents a category of shortcuts
type ShortcutTab struct {
	Name      string
	Shortcuts []Shortcut
}

// GetShortcutTabs generates help text from actual keybindings
func GetShortcutTabs(keys keybindings.KeyMap) []ShortcutTab {
	tabs := []ShortcutTab{}

	for _, group := range keys.GetKeyGroups() {
		shortcuts := []Shortcut{}
		for _, binding := range group.Bindings {
			help := binding.Help()
			shortcuts = append(shortcuts, Shortcut{
				Key:         help.Key,
				Description: help.Desc,
			})
		}
		tabs = append(tabs, ShortcutTab{
			Name:      group.Name,
			Shortcuts: shortcuts,
		})
	}

	return tabs
}
