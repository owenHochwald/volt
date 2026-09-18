package shortcutpane

import (
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// SetupShortcutPane sets up the shortcut pane for use
func SetupShortcutPane(keys keybindings.KeyMap, optionalStyles ...design.Styles) ShortcutPane {
	styles := design.NewStyles(design.DefaultTheme())
	if len(optionalStyles) > 0 {
		styles = optionalStyles[0]
	}

	return ShortcutPane{
		activeTab: 0,
		height:    30,
		width:     40,
		Focused:   false,
		keys:      keys,
		tabs:      GetShortcutTabs(keys),
		styles:    styles,
	}

}
