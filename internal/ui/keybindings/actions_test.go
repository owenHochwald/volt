package keybindings

import (
	"slices"
	"testing"

	"charm.land/bubbles/v2/key"
)

func TestDefaultRegistryDefinesCoreActions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id       ActionID
		contexts []Context
		keys     []string
	}{
		{ActionForceQuit, []Context{ContextGlobal}, []string{"ctrl+c"}},
		{ActionQuit, []Context{ContextGlobal}, []string{"esc"}},
		{ActionGlobalHelp, []Context{ContextGlobal}, []string{"f1"}},
		{ActionContextHelp, []Context{ContextSidebar, ContextRequest, ContextResponse}, []string{"?"}},
		{ActionPreviousPanel, []Context{ContextGlobal}, []string{"ctrl+e", "alt+h"}},
		{ActionNextPanel, []Context{ContextGlobal}, []string{"ctrl+w", "alt+l"}},
		{ActionNavigationCount, []Context{ContextSidebar}, []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}},
		{ActionPreviousField, []Context{ContextRequest}, []string{"shift+tab"}},
		{ActionNextField, []Context{ContextRequest}, []string{"tab"}},
		{ActionSubmit, []Context{ContextRequest}, []string{"alt+enter"}},
		{ActionActivateControl, []Context{ContextRequest}, []string{"enter"}},
	}

	registry := DefaultRegistry()
	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.id), func(t *testing.T) {
			t.Parallel()

			action, ok := registry.Action(tt.id)
			if !ok {
				t.Fatalf("missing action %q", tt.id)
			}
			if !slices.Equal(action.Contexts, tt.contexts) {
				t.Fatalf("contexts = %v, want %v", action.Contexts, tt.contexts)
			}
			if !slices.Equal(action.Keys, tt.keys) {
				t.Fatalf("keys = %v, want %v", action.Keys, tt.keys)
			}
			if action.Description == "" {
				t.Fatal("description must not be empty")
			}
			if action.Priority <= 0 {
				t.Fatalf("priority = %d, want positive value", action.Priority)
			}
		})
	}
}

type keyString string

func (k keyString) String() string {
	return string(k)
}

func TestEveryRegisteredKeyMatchesItsBinding(t *testing.T) {
	t.Parallel()

	for _, action := range DefaultRegistry().Actions() {
		action := action
		t.Run(string(action.ID), func(t *testing.T) {
			t.Parallel()
			binding := action.Binding()
			for _, registeredKey := range action.Keys {
				if !key.Matches(keyString(registeredKey), binding) {
					t.Errorf("registered key %q does not match its generated binding", registeredKey)
				}
			}
		})
	}
}

func TestEveryActionAppearsOnlyInItsDeclaredContexts(t *testing.T) {
	t.Parallel()

	registry := DefaultRegistry()
	contexts := []Context{
		ContextSidebar,
		ContextRequest,
		ContextResponse,
		ContextHelp,
	}
	for _, action := range registry.Actions() {
		for _, context := range contexts {
			got := slices.ContainsFunc(registry.ActionsFor(context), func(candidate Action) bool {
				return candidate.ID == action.ID
			})
			want := slices.Contains(action.Contexts, ContextGlobal) || slices.Contains(action.Contexts, context)
			if got != want {
				t.Errorf("%s in %s = %t, want %t", action.ID, context, got, want)
			}
		}
	}
}

func TestHelpGroupsAreGeneratedFromDeclaredRegistryActions(t *testing.T) {
	t.Parallel()

	keyMap := DefaultKeyMap()
	for _, group := range keyMap.GetKeyGroups() {
		actions := keyMap.Registry.ActionsDeclaredFor(group.Context)
		if len(group.Bindings) != len(actions) {
			t.Fatalf("%s help bindings = %d, registry actions = %d", group.Context, len(group.Bindings), len(actions))
		}
		for i, action := range actions {
			if !slices.Equal(group.Bindings[i].Keys(), action.Keys) {
				t.Errorf("%s help keys = %v, registry keys = %v", action.ID, group.Bindings[i].Keys(), action.Keys)
			}
			if help := group.Bindings[i].Help(); help.Key != action.KeyHelp || help.Desc != action.Description {
				t.Errorf("%s help = %+v, want %q %q", action.ID, help, action.KeyHelp, action.Description)
			}
		}
	}
}

func TestDefaultRegistryReservesEditingKeys(t *testing.T) {
	t.Parallel()

	registry := DefaultRegistry()
	for _, action := range registry.Actions() {
		for _, key := range action.Keys {
			if key == "up" || key == "down" || key == "left" || key == "right" {
				t.Errorf("%s claims editor arrow key %q", action.ID, key)
			}
			if key == "tab" && action.ID != ActionNextField {
				t.Errorf("%s claims tab; only next field may use it", action.ID)
			}
			if key == "shift+tab" && action.ID != ActionPreviousField {
				t.Errorf("%s claims shift+tab; only previous field may use it", action.ID)
			}
			if key == "enter" && action.ID == ActionSubmit {
				t.Error("submit must not claim plain enter")
			}
		}
	}
}

func TestDefaultRegistryHasNoAmbiguousKeysWithinAContext(t *testing.T) {
	t.Parallel()

	registry := DefaultRegistry()
	for _, context := range []Context{
		ContextSidebar,
		ContextRequest,
		ContextResponse,
		ContextHelp,
	} {
		owners := map[string]ActionID{}
		for _, action := range registry.ActionsFor(context) {
			for _, key := range action.Keys {
				if previous, exists := owners[key]; exists {
					t.Errorf("%s: key %q belongs to both %s and %s", context, key, previous, action.ID)
				}
				owners[key] = action.ID
			}
		}
	}
}

func TestActionsForContextIncludesGlobalsAndSortsByPriority(t *testing.T) {
	t.Parallel()

	actions := DefaultRegistry().ActionsFor(ContextRequest)
	if len(actions) == 0 {
		t.Fatal("request context has no actions")
	}

	seenGlobalHelp := false
	for i, action := range actions {
		if action.ID == ActionGlobalHelp {
			seenGlobalHelp = true
		}
		if i > 0 && actions[i-1].Priority < action.Priority {
			t.Fatalf("actions are not sorted by descending priority: %v before %v", actions[i-1], action)
		}
	}
	if !seenGlobalHelp {
		t.Fatal("request actions must include globally available help")
	}
}

func TestDefaultKeyMapIsGeneratedFromRegistry(t *testing.T) {
	t.Parallel()

	keyMap := DefaultKeyMap()
	tests := []struct {
		id      ActionID
		binding []string
	}{
		{ActionForceQuit, keyMap.ForceQuit.Keys()},
		{ActionQuit, keyMap.Quit.Keys()},
		{ActionGlobalHelp, keyMap.GlobalHelp.Keys()},
		{ActionContextHelp, keyMap.ContextHelp.Keys()},
		{ActionPreviousPanel, keyMap.PreviousPanel.Keys()},
		{ActionNextPanel, keyMap.NextPanel.Keys()},
		{ActionNavigationCount, keyMap.NavCount.Keys()},
		{ActionPreviousField, keyMap.PrevField.Keys()},
		{ActionNextField, keyMap.NextField.Keys()},
		{ActionSubmit, keyMap.SendRequest.Keys()},
		{ActionActivateControl, keyMap.ActivateControl.Keys()},
		{ActionCancelLoadTest, keyMap.CancelLoadTest.Keys()},
	}

	for _, tt := range tests {
		action, ok := keyMap.Registry.Action(tt.id)
		if !ok {
			t.Fatalf("key map registry is missing %s", tt.id)
		}
		if !slices.Equal(tt.binding, action.Keys) {
			t.Errorf("%s binding = %v, registry keys = %v", tt.id, tt.binding, action.Keys)
		}
	}
}
