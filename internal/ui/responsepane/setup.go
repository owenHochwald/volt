package responsepane

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// SetupResponsePane creates and initializes a new ResponsePane with default values
func SetupResponsePane(keys keybindings.KeyMap) ResponsePane {
	filenameInput := textinput.New()
	filenameInput.Placeholder = "response.json"
	filenameInput.CharLimit = 500
	filenameInput.Width = 50

	searchInput := textinput.New()
	searchInput.Placeholder = "search query"
	searchInput.CharLimit = 500
	searchInput.Width = 50

	return ResponsePane{
		viewport:       viewport.New(20, 10),
		width:          20,
		height:         30,
		activeTab:      int(TabBody), // Start on Body tab
		isLoadTest:     false,
		filenameInput:  filenameInput,
		searchInput:    searchInput,
		searchMatchIdx: -1,
		keys:           keys,
	}
}
