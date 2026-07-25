package responsepane

import (
	"charm.land/bubbles/v2/viewport"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// SetupResponsePane creates and initializes a new ResponsePane with default values
func SetupResponsePane(keys keybindings.KeyMap) ResponsePane {
	return ResponsePane{
		viewport:   viewport.New(viewport.WithWidth(20), viewport.WithHeight(10)),
		width:      20,
		height:     30,
		activeTab:  int(TabBody), // Start on Body tab
		isLoadTest: false,
		keys:       keys,
	}
}
