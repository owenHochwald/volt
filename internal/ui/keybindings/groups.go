package keybindings

import (
	"strings"

	"charm.land/bubbles/v2/key"
)

// KeyGroup represents a category of keybindings for help display.
type KeyGroup struct {
	Name     string
	Context  Context
	Bindings []key.Binding
}

// CompactHelp returns a short registry-derived hint line for the status bar.
func (km KeyMap) CompactHelp(context Context, limit int) string {
	actions := append(
		km.Registry.ActionsDeclaredFor(context),
		km.Registry.ActionsDeclaredFor(ContextGlobal)...,
	)
	if limit > 0 && len(actions) > limit {
		actions = actions[:limit]
	}

	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		parts = append(parts, action.KeyHelp+" "+action.Description)
	}
	return strings.Join(parts, " • ")
}

// GetKeyGroups generates contextual help groups from the action registry.
func (km KeyMap) GetKeyGroups() []KeyGroup {
	contexts := []Context{
		ContextGlobal,
		ContextSidebar,
		ContextRequest,
		ContextResponse,
	}
	groups := make([]KeyGroup, 0, len(contexts))
	for _, context := range contexts {
		actions := km.Registry.ActionsDeclaredFor(context)
		bindings := make([]key.Binding, 0, len(actions))
		for _, action := range actions {
			bindings = append(bindings, action.Binding())
		}
		groups = append(groups, KeyGroup{
			Name:     string(context),
			Context:  context,
			Bindings: bindings,
		})
	}
	return groups
}
