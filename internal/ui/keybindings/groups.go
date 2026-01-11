package keybindings

import "github.com/charmbracelet/bubbles/key"

// KeyGroup represents a category of keybindings for help display
type KeyGroup struct {
	Name     string
	Bindings []key.Binding
}

// GetKeyGroups returns keybindings organized by context for help display
func (km KeyMap) GetKeyGroups() []KeyGroup {
	return []KeyGroup{
		{
			Name: "Global",
			Bindings: []key.Binding{
				km.ShowHelp,
				km.CyclePanel,
				km.Quit,
			},
		},
		{
			Name: "Sidebar",
			Bindings: []key.Binding{
				km.LoadRequest,
				km.DeleteRequest,
				km.NavUp,
				km.NavDown,
			},
		},
		{
			Name: "Request",
			Bindings: []key.Binding{
				km.SendRequest,
				km.SaveRequest,
				km.ToggleLoadTest,
				km.NextField,
				km.ChangeMethodNext,
			},
		},
		{
			Name: "Response",
			Bindings: []key.Binding{
				km.DirectTab,
				km.TabNavNext,
				km.CopyResponse,
				km.ScrollUp,
			},
		},
	}
}
