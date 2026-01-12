package keybindings

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all application keybindings
type KeyMap struct {
	// Global
	Quit        key.Binding
	ShowHelp    key.Binding
	CyclePanel  key.Binding
	EscapePanel key.Binding

	// Sidebar
	LoadRequest   key.Binding
	DeleteRequest key.Binding
	NavUp         key.Binding
	NavDown       key.Binding

	// Request Pane
	SendRequest      key.Binding
	SaveRequest      key.Binding
	ToggleLoadTest   key.Binding
	NextField        key.Binding
	PrevField        key.Binding
	ChangeMethodNext key.Binding
	ChangeMethodPrev key.Binding

	// Response Pane
	CopyResponse key.Binding
	TabNavNext   key.Binding
	TabNavPrev   key.Binding
	DirectTab    key.Binding // 1,2,3
	ScrollUp     key.Binding
	ScrollDown   key.Binding

	// Help Modal
	CloseHelp key.Binding
	NextTab   key.Binding
	PrevTab   key.Binding
}

// DefaultKeyMap returns the default keybinding configuration
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Global
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		ShowHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "show help"),
		),
		CyclePanel: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "cycle panels"),
		),
		EscapePanel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "return to sidebar"),
		),

		// Sidebar
		LoadRequest: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "load request"),
		),
		DeleteRequest: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete request"),
		),
		NavUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "navigate up"),
		),
		NavDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "navigate down"),
		),

		SendRequest: key.NewBinding(
			key.WithKeys("alt+enter", "ctrl+p"),
			key.WithHelp("ctrl+p", "send request"),
		),
		SaveRequest: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save request"),
		),
		ToggleLoadTest: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "toggle load test"),
		),
		NextField: key.NewBinding(
			key.WithKeys("tab", "down"),
			key.WithHelp("tab/↓", "next field"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab", "up"),
			key.WithHelp("shift+tab/↑", "previous field"),
		),
		ChangeMethodNext: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/→", "next method"),
		),
		ChangeMethodPrev: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/←", "previous method"),
		),

		// Response Pane
		CopyResponse: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y/Y", "copy response"),
		),
		TabNavNext: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("l/→", "next tab"),
		),
		TabNavPrev: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/←", "previous tab"),
		),
		DirectTab: key.NewBinding(
			key.WithKeys("1", "2", "3"),
			key.WithHelp("1-3", "jump to tab"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "scroll down"),
		),

		// Help Modal
		CloseHelp: key.NewBinding(
			key.WithKeys("q", "?", "esc"),
			key.WithHelp("q/?/esc", "close help"),
		),
		NextTab: key.NewBinding(
			key.WithKeys("l", "right", "tab"),
			key.WithHelp("l/→/tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("h", "left", "shift+tab"),
			key.WithHelp("h/←/shift+tab", "previous tab"),
		),
	}
}
