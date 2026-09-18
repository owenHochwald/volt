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
				km.FocusSidebar,
				km.FocusRequest,
				km.FocusResponse,
				km.CyclePanel,
				km.OpenGroups,
				km.OpenImport,
				km.ToggleHeader,
				km.Quit,
			},
		},
		{
			Name: "Groups & Servers",
			Bindings: []key.Binding{
				km.CreateItem,
				km.RenameItem,
				km.DeleteItem,
				km.CloseGroups,
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
				km.ResetRequest,
				km.ClearParams,
				km.ToggleLoadTest,
				km.NextField,
				km.ChangeMethodNext,
				km.ScrollFieldsDown,
			},
		},
		{
			Name: "Response",
			Bindings: []key.Binding{
				km.DirectTab,
				km.TabNavNext,
				km.CopyResponse,
				km.ToggleRawBody,
				km.ToggleWrap,
				km.SaveResponseRaw,
				km.ScrollUp,
				km.SearchResponse,
				km.SearchNext,
				km.SearchPrev,
				km.SearchNextFromPosition,
				km.SearchPrevFromPosition,
				km.GotoTop,
				km.GotoBottom,
				km.GotoLine,
				km.PickBodyStyle,
				km.ToggleZoomResponse,
			},
		},
	}
}
