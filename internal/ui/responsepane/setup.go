package responsepane

import (
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/owenHochwald/volt/internal/ui/keybindings"
)

// SetupResponsePane creates and initializes a new ResponsePane with default values
func SetupResponsePane(keys keybindings.KeyMap) ResponsePane {
	return ResponsePane{
		viewport:   viewport.New(20, 10),
		width:      20,
		height:     30,
		activeTab:  int(TabBody), // Start on Body tab
		isLoadTest: false,
		keys:       keys,
	}
}
