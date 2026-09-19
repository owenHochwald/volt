package keybindings

import "charm.land/bubbles/v2/key"

// KeyMap exposes typed bindings while the Registry remains their source of
// truth.
type KeyMap struct {
	Registry Registry

	// Global
	ForceQuit     key.Binding
	Quit          key.Binding
	GlobalHelp    key.Binding
	ContextHelp   key.Binding
	PreviousPanel key.Binding
	NextPanel     key.Binding
	OpenGroups    key.Binding
	OpenImport    key.Binding

	// Sidebar
	LoadRequest   key.Binding
	DeleteRequest key.Binding
	NavCount      key.Binding
	NavUp         key.Binding
	NavDown       key.Binding
	NavFirst      key.Binding
	NavLast       key.Binding

	// Request pane
	SendRequest      key.Binding
	ActivateControl  key.Binding
	SaveRequest      key.Binding
	ResetRequest     key.Binding
	ClearParams      key.Binding
	CopyAsCurl       key.Binding
	ToggleLoadTest   key.Binding
	NextField        key.Binding
	PrevField        key.Binding
	ChangeMethodNext key.Binding
	ChangeMethodPrev key.Binding
	ScrollFieldsUp   key.Binding
	ScrollFieldsDown key.Binding

	// Response pane
	CopyResponse    key.Binding
	CancelLoadTest  key.Binding
	ToggleRawBody   key.Binding
	ToggleWrap      key.Binding
	SaveResponseRaw key.Binding
	TabNavNext      key.Binding
	TabNavPrev      key.Binding
	DirectTab       key.Binding
	ScrollUp        key.Binding
	ScrollDown      key.Binding
	PageUp          key.Binding
	PageDown        key.Binding

	// GotoTop/GotoBottom/GotoLine are vim/less-style navigation over
	// whichever tab's content is currently in the viewport (Body - raw or
	// pretty - Headers, Timing, Request all share it, so these work the
	// same regardless of which is active). GotoTop's key ("g") only *arms*
	// the sequence - ResponsePane.gPending decides from the next keystroke
	// whether it's "gg" (bare, go to the very top), digits followed by "g"
	// (go to that line number - see GotoLine, display-only since it has no
	// single keystroke of its own), or anything else (cancelled). GotoBottom
	// ("G") needs no such sequence - it's a single keystroke to the very
	// bottom.
	GotoTop    key.Binding
	GotoBottom key.Binding
	GotoLine   key.Binding

	// SearchResponse opens a less/vim-style "/" search prompt over
	// whatever's currently in the viewport; SearchNext/SearchPrev ("n"/"b")
	// cycle through the matches it finds, wrapping around from wherever
	// the *last search move* (a search, "n", "b", "N", or "P") left off.
	// SearchNextFromPosition/SearchPrevFromPosition ("N"/"P") do the same
	// forward/backward search, but relative to wherever the view is
	// currently scrolled to instead - e.g. after "gg"/"G"/"g{n}g" moved the
	// view somewhere the remembered match sequence knows nothing about.
	SearchResponse         key.Binding
	SearchNext             key.Binding
	SearchPrev             key.Binding
	SearchNextFromPosition key.Binding
	SearchPrevFromPosition key.Binding

	// PickBodyStyle ("t") opens a fuzzy-filterable list of every syntax
	// highlighting style chroma ships, applied live as the list cursor
	// moves - see responsepane's style_picker.go.
	PickBodyStyle key.Binding

	// ToggleZoomResponse ("z") expands the response panel to the right
	// column's full height, hiding the request panel, while focused on
	// the response panel - the same key un-zooms it again, and so does
	// leaving the response panel any other way (see app.Model.setFocusedPanel).
	ToggleZoomResponse key.Binding

	// CaptureAuth ("c") remembers the current response's Set-Cookie
	// headers and the sent request's Authorization header for its host
	// (see internal/authjar), regardless of the auto-remember settings -
	// those only gate whether the same capture also happens automatically
	// on every response (see responsepane.ResponsePane.SetResponse).
	// ViewAuthJar ("v") opens a list of everything currently remembered
	// for the response's host, letting individual entries be deleted or
	// everything cleared at once - see responsepane's jar_viewer.go.
	CaptureAuth key.Binding
	ViewAuthJar key.Binding

	// ViewHistory ("H") opens a list of every send this session has
	// recorded (see internal/history) - "enter" on one shows it as the
	// currently displayed response, "d"/"D" delete one/clear all - see
	// responsepane's history_viewer.go.
	ViewHistory key.Binding

	// Help modal
	CloseHelp     key.Binding
	NextTab       key.Binding
	PrevTab       key.Binding
	DirectHelpTab key.Binding

	// Groups & Servers Modal / Import Modal
	CloseGroups key.Binding
	CloseImport key.Binding
	CreateItem  key.Binding
	RenameItem  key.Binding
	DeleteItem  key.Binding
}

// DefaultKeyMap generates every binding from the action registry.
func DefaultKeyMap() KeyMap {
	registry := DefaultRegistry()
	return KeyMap{
		Registry:               registry,
		ForceQuit:              registry.MustBinding(ActionForceQuit),
		Quit:                   registry.MustBinding(ActionQuit),
		GlobalHelp:             registry.MustBinding(ActionGlobalHelp),
		ContextHelp:            registry.MustBinding(ActionContextHelp),
		PreviousPanel:          registry.MustBinding(ActionPreviousPanel),
		NextPanel:              registry.MustBinding(ActionNextPanel),
		OpenGroups:             registry.MustBinding(ActionOpenGroups),
		OpenImport:             registry.MustBinding(ActionOpenImport),
		LoadRequest:            registry.MustBinding(ActionLoadRequest),
		DeleteRequest:          registry.MustBinding(ActionDeleteRequest),
		NavCount:               registry.MustBinding(ActionNavigationCount),
		NavUp:                  registry.MustBinding(ActionNavigateUp),
		NavDown:                registry.MustBinding(ActionNavigateDown),
		NavFirst:               registry.MustBinding(ActionNavigateFirst),
		NavLast:                registry.MustBinding(ActionNavigateLast),
		SendRequest:            registry.MustBinding(ActionSubmit),
		ActivateControl:        registry.MustBinding(ActionActivateControl),
		SaveRequest:            registry.MustBinding(ActionSaveRequest),
		ResetRequest:           registry.MustBinding(ActionResetRequest),
		ClearParams:            registry.MustBinding(ActionClearParams),
		CopyAsCurl:             registry.MustBinding(ActionCopyAsCurl),
		ToggleLoadTest:         registry.MustBinding(ActionToggleLoadTest),
		NextField:              registry.MustBinding(ActionNextField),
		PrevField:              registry.MustBinding(ActionPreviousField),
		ChangeMethodNext:       registry.MustBinding(ActionNextMethod),
		ChangeMethodPrev:       registry.MustBinding(ActionPreviousMethod),
		ScrollFieldsUp:         registry.MustBinding(ActionScrollFieldsUp),
		ScrollFieldsDown:       registry.MustBinding(ActionScrollFieldsDown),
		CopyResponse:           registry.MustBinding(ActionCopyResponse),
		CancelLoadTest:         registry.MustBinding(ActionCancelLoadTest),
		ToggleRawBody:          registry.MustBinding(ActionToggleRawBody),
		ToggleWrap:             registry.MustBinding(ActionToggleWrap),
		SaveResponseRaw:        registry.MustBinding(ActionSaveResponseRaw),
		TabNavNext:             registry.MustBinding(ActionNextTab),
		TabNavPrev:             registry.MustBinding(ActionPreviousTab),
		DirectTab:              registry.MustBinding(ActionDirectTab),
		ScrollUp:               registry.MustBinding(ActionScrollUp),
		ScrollDown:             registry.MustBinding(ActionScrollDown),
		PageUp:                 registry.MustBinding(ActionPageUp),
		PageDown:               registry.MustBinding(ActionPageDown),
		GotoTop:                registry.MustBinding(ActionGotoTop),
		GotoBottom:             registry.MustBinding(ActionGotoBottom),
		GotoLine:               registry.MustBinding(ActionGotoLine),
		SearchResponse:         registry.MustBinding(ActionSearchResponse),
		SearchNext:             registry.MustBinding(ActionSearchNext),
		SearchPrev:             registry.MustBinding(ActionSearchPrev),
		SearchNextFromPosition: registry.MustBinding(ActionSearchNextFromPos),
		SearchPrevFromPosition: registry.MustBinding(ActionSearchPrevFromPos),
		PickBodyStyle:          registry.MustBinding(ActionPickBodyStyle),
		ToggleZoomResponse:     registry.MustBinding(ActionToggleZoomResponse),
		CaptureAuth:            registry.MustBinding(ActionCaptureAuth),
		ViewAuthJar:            registry.MustBinding(ActionViewAuthJar),
		ViewHistory:            registry.MustBinding(ActionViewHistory),
		CloseHelp:              registry.MustBinding(ActionCloseHelp),
		NextTab:                registry.MustBinding(ActionNextHelpTab),
		PrevTab:                registry.MustBinding(ActionPreviousHelpTab),
		DirectHelpTab:          registry.MustBinding(ActionDirectHelpTab),
		CloseGroups:            registry.MustBinding(ActionCloseGroups),
		CloseImport:            registry.MustBinding(ActionCloseImport),
		CreateItem:             registry.MustBinding(ActionCreateItem),
		RenameItem:             registry.MustBinding(ActionRenameItem),
		DeleteItem:             registry.MustBinding(ActionDeleteItem),
	}
}
