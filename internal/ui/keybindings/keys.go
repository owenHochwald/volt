package keybindings

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all application keybindings
type KeyMap struct {
	// Global
	Quit        key.Binding
	ShowHelp    key.Binding
	CyclePanel  key.Binding
	EscapePanel key.Binding
	OpenGroups  key.Binding
	OpenImport  key.Binding
	// FocusSidebar/FocusRequest/FocusResponse jump directly to a panel
	// (f1/f2/f3) regardless of what's currently focused or being typed into
	// - CyclePanel (shift+tab) can't reliably fill that role since the
	// request pane also binds shift+tab to its own PrevField. Two other
	// options were tried and ruled out first: ctrl+1/2/3 isn't a real,
	// distinguishable key combination over classic terminal input (most
	// terminals don't send a separate code for it at all, and ctrl+3 is
	// indistinguishable from plain Escape); alt+1/2/3 IS distinguishable,
	// but most terminal emulators grab alt+digit themselves for their own
	// tab switching before the keystroke ever reaches the app. F-keys are
	// what terminal TUIs (e.g. htop) conventionally use instead, and
	// they're essentially never intercepted by the terminal itself.
	FocusSidebar  key.Binding
	FocusRequest  key.Binding
	FocusResponse key.Binding

	// Sidebar
	LoadRequest   key.Binding
	DeleteRequest key.Binding
	NavUp         key.Binding
	NavDown       key.Binding

	// Request Pane
	SendRequest      key.Binding
	SaveRequest      key.Binding
	ResetRequest     key.Binding
	ClearParams      key.Binding
	ToggleLoadTest   key.Binding
	NextField        key.Binding
	PrevField        key.Binding
	ChangeMethodNext key.Binding
	ChangeMethodPrev key.Binding
	ScrollFieldsUp   key.Binding
	ScrollFieldsDown key.Binding

	// Response Pane
	CopyResponse    key.Binding
	ToggleRawBody   key.Binding
	SaveResponseRaw key.Binding
	TabNavNext      key.Binding
	TabNavPrev      key.Binding
	DirectTab       key.Binding // 1,2,3,4
	ScrollUp        key.Binding
	ScrollDown      key.Binding

	// Help Modal
	CloseHelp key.Binding
	NextTab   key.Binding
	PrevTab   key.Binding

	// Groups & Servers Modal / Import Modal
	CloseGroups key.Binding
	CloseImport key.Binding
	CreateItem  key.Binding
	RenameItem  key.Binding
	DeleteItem  key.Binding
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
		FocusSidebar: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("f1", "focus sidebar"),
		),
		FocusRequest: key.NewBinding(
			key.WithKeys("f2"),
			key.WithHelp("f2", "focus request"),
		),
		FocusResponse: key.NewBinding(
			key.WithKeys("f3"),
			key.WithHelp("f3", "focus response"),
		),
		// EscapePanel steps back one panel at a time (response -> request
		// -> sidebar), the same direction as the panel layout itself reads
		// left to right - not straight to the sidebar from anywhere.
		EscapePanel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back a panel"),
		),
		OpenGroups: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "groups & servers"),
		),
		OpenImport: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "import OpenAPI spec"),
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
		// ResetRequest clears the pane back to a brand-new, unsaved
		// request - the same blank state the app starts on - so a request
		// that's been loaded and edited (typed-over field values, a
		// different method, etc.) can be discarded without reloading the
		// app or re-selecting a different request first just to lose the
		// current one's state.
		ResetRequest: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "reset request"),
		),
		// ClearParams resets every path/body/array field back to its
		// schema's own defaults, discarding anything typed over them -
		// unlike ResetRequest above, this keeps the request's own identity
		// (schema/URL/method/name), only clearing its *values*. Saving
		// afterward (ctrl+s) persists that cleared state too, same as any
		// other edit.
		ClearParams: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("ctrl+x", "clear params"),
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
		ScrollFieldsUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "scroll fields up"),
		),
		ScrollFieldsDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "scroll fields down"),
		),

		// Response Pane
		CopyResponse: key.NewBinding(
			key.WithKeys("y", "Y"),
			key.WithHelp("y/Y", "copy response"),
		),
		// ToggleRawBody switches the Body tab between its normal
		// formatted/highlighted rendering (see formatContentByType) and
		// the response body exactly as received - useful when the pretty
		// version hides or reformats something (whitespace, a body that
		// isn't actually the content-type it claims to be, etc.) that
		// matters to see byte-for-byte.
		ToggleRawBody: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "toggle raw body"),
		),
		// SaveResponseRaw opens a filename prompt (see ResponsePane's
		// savingToFile) to save the response body - always the raw bytes
		// as received, regardless of ToggleRawBody's current display
		// state - to a file on disk.
		SaveResponseRaw: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save raw body to file"),
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
			key.WithKeys("1", "2", "3", "4"),
			key.WithHelp("1-4", "jump to tab"),
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

		// Groups & Servers Modal / Import Modal
		CloseGroups: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q/esc", "close"),
		),
		CloseImport: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
		CreateItem: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new"),
		),
		RenameItem: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename"),
		),
		DeleteItem: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
	}
}
