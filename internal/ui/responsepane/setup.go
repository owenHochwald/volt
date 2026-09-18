package responsepane

import (
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// SetupResponsePane creates and initializes a new ResponsePane with
// default values.
func SetupResponsePane(keys keybindings.KeyMap, optionalStyles ...design.Styles) ResponsePane {
	styles := design.NewStyles(design.DefaultTheme())
	if len(optionalStyles) > 0 {
		styles = optionalStyles[0]
	}

	filenameInput := textinput.New()
	filenameInput.Placeholder = "response.json"
	filenameInput.CharLimit = 500
	filenameInput.SetWidth(50)

	searchInput := textinput.New()
	searchInput.Placeholder = "search query (regex)"
	searchInput.CharLimit = 500
	searchInput.SetWidth(50)

	bodyStyleName := defaultBodyStyle
	if saved, err := design.LoadBodyStyle(); err == nil && saved != "" {
		bodyStyleName = saved
	}

	return ResponsePane{
		viewport:       viewport.New(viewport.WithWidth(20), viewport.WithHeight(10)),
		width:          20,
		height:         30,
		activeTab:      int(TabBody), // Start on Body tab
		isLoadTest:     false,
		filenameInput:  filenameInput,
		searchInput:    searchInput,
		searchMatchIdx: -1,
		bodyStyleName:  bodyStyleName,
		keys:           keys,
		styles:         styles,
	}
}
