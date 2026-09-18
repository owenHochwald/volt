package shortcutpane

import (
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/groupspane"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// SetupShortcutPane sets up the shortcut pane for use. db may be nil in
// tests that never switch to the Groups surface - GroupsPane.Open (see
// SetSurface) is the first thing that would touch it.
func SetupShortcutPane(keys keybindings.KeyMap, db *storage.SQLiteStorage, optionalStyles ...design.Styles) ShortcutPane {
	styles := design.NewStyles(design.DefaultTheme())
	if len(optionalStyles) > 0 {
		styles = optionalStyles[0]
	}

	return ShortcutPane{
		activeTab:  0,
		height:     30,
		width:      40,
		Focused:    false,
		keys:       keys,
		tabs:       GetShortcutTabs(keys),
		styles:     styles,
		groupsPane: groupspane.SetupGroupsPane(db, keys),
	}

}
