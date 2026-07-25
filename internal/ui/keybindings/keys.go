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
	ToggleLoadTest   key.Binding
	NextField        key.Binding
	PrevField        key.Binding
	ChangeMethodNext key.Binding
	ChangeMethodPrev key.Binding

	// Response pane
	CopyResponse   key.Binding
	CancelLoadTest key.Binding
	TabNavNext     key.Binding
	TabNavPrev     key.Binding
	DirectTab      key.Binding
	ScrollUp       key.Binding
	ScrollDown     key.Binding
	PageUp         key.Binding
	PageDown       key.Binding

	// Help modal
	CloseHelp     key.Binding
	NextTab       key.Binding
	PrevTab       key.Binding
	DirectHelpTab key.Binding
}

// DefaultKeyMap generates every binding from the action registry.
func DefaultKeyMap() KeyMap {
	registry := DefaultRegistry()
	return KeyMap{
		Registry:         registry,
		ForceQuit:        registry.MustBinding(ActionForceQuit),
		Quit:             registry.MustBinding(ActionQuit),
		GlobalHelp:       registry.MustBinding(ActionGlobalHelp),
		ContextHelp:      registry.MustBinding(ActionContextHelp),
		PreviousPanel:    registry.MustBinding(ActionPreviousPanel),
		NextPanel:        registry.MustBinding(ActionNextPanel),
		LoadRequest:      registry.MustBinding(ActionLoadRequest),
		DeleteRequest:    registry.MustBinding(ActionDeleteRequest),
		NavCount:         registry.MustBinding(ActionNavigationCount),
		NavUp:            registry.MustBinding(ActionNavigateUp),
		NavDown:          registry.MustBinding(ActionNavigateDown),
		NavFirst:         registry.MustBinding(ActionNavigateFirst),
		NavLast:          registry.MustBinding(ActionNavigateLast),
		SendRequest:      registry.MustBinding(ActionSubmit),
		ActivateControl:  registry.MustBinding(ActionActivateControl),
		SaveRequest:      registry.MustBinding(ActionSaveRequest),
		ToggleLoadTest:   registry.MustBinding(ActionToggleLoadTest),
		NextField:        registry.MustBinding(ActionNextField),
		PrevField:        registry.MustBinding(ActionPreviousField),
		ChangeMethodNext: registry.MustBinding(ActionNextMethod),
		ChangeMethodPrev: registry.MustBinding(ActionPreviousMethod),
		CopyResponse:     registry.MustBinding(ActionCopyResponse),
		CancelLoadTest:   registry.MustBinding(ActionCancelLoadTest),
		TabNavNext:       registry.MustBinding(ActionNextTab),
		TabNavPrev:       registry.MustBinding(ActionPreviousTab),
		DirectTab:        registry.MustBinding(ActionDirectTab),
		ScrollUp:         registry.MustBinding(ActionScrollUp),
		ScrollDown:       registry.MustBinding(ActionScrollDown),
		PageUp:           registry.MustBinding(ActionPageUp),
		PageDown:         registry.MustBinding(ActionPageDown),
		CloseHelp:        registry.MustBinding(ActionCloseHelp),
		NextTab:          registry.MustBinding(ActionNextHelpTab),
		PrevTab:          registry.MustBinding(ActionPreviousHelpTab),
		DirectHelpTab:    registry.MustBinding(ActionDirectHelpTab),
	}
}
