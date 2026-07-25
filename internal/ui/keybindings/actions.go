package keybindings

import (
	"slices"
	"sort"

	"charm.land/bubbles/v2/key"
)

// Context identifies where an action is available.
type Context string

const (
	ContextGlobal   Context = "Global"
	ContextSidebar  Context = "Sidebar"
	ContextRequest  Context = "Request"
	ContextResponse Context = "Response"
	ContextHelp     Context = "Help"
)

// ActionID is a stable identifier used by input handling and tests.
type ActionID string

const (
	ActionForceQuit       ActionID = "force_quit"
	ActionQuit            ActionID = "quit"
	ActionGlobalHelp      ActionID = "global_help"
	ActionContextHelp     ActionID = "context_help"
	ActionPreviousPanel   ActionID = "previous_panel"
	ActionNextPanel       ActionID = "next_panel"
	ActionLoadRequest     ActionID = "load_request"
	ActionDeleteRequest   ActionID = "delete_request"
	ActionNavigationCount ActionID = "navigation_count"
	ActionNavigateUp      ActionID = "navigate_up"
	ActionNavigateDown    ActionID = "navigate_down"
	ActionNavigateFirst   ActionID = "navigate_first"
	ActionNavigateLast    ActionID = "navigate_last"
	ActionSubmit          ActionID = "submit"
	ActionActivateControl ActionID = "activate_control"
	ActionSaveRequest     ActionID = "save_request"
	ActionToggleLoadTest  ActionID = "toggle_load_test"
	ActionNextField       ActionID = "next_field"
	ActionPreviousField   ActionID = "previous_field"
	ActionNextMethod      ActionID = "next_method"
	ActionPreviousMethod  ActionID = "previous_method"
	ActionCopyResponse    ActionID = "copy_response"
	ActionCancelLoadTest  ActionID = "cancel_load_test"
	ActionNextTab         ActionID = "next_tab"
	ActionPreviousTab     ActionID = "previous_tab"
	ActionDirectTab       ActionID = "direct_tab"
	ActionScrollUp        ActionID = "scroll_up"
	ActionScrollDown      ActionID = "scroll_down"
	ActionPageUp          ActionID = "page_up"
	ActionPageDown        ActionID = "page_down"
	ActionCloseHelp       ActionID = "close_help"
	ActionNextHelpTab     ActionID = "next_help_tab"
	ActionPreviousHelpTab ActionID = "previous_help_tab"
	ActionDirectHelpTab   ActionID = "direct_help_tab"
)

// Action is the single source of truth for an interaction and its help text.
type Action struct {
	ID          ActionID
	Contexts    []Context
	Keys        []string
	KeyHelp     string
	Description string
	Priority    int
}

func (a Action) Binding() key.Binding {
	return key.NewBinding(
		key.WithKeys(a.Keys...),
		key.WithHelp(a.KeyHelp, a.Description),
	)
}

func (a Action) appliesTo(context Context) bool {
	return slices.Contains(a.Contexts, ContextGlobal) || slices.Contains(a.Contexts, context)
}

// Registry stores every advertised and dispatchable terminal action.
type Registry struct {
	actions []Action
	byID    map[ActionID]Action
}

func NewRegistry(actions []Action) Registry {
	copied := append([]Action(nil), actions...)
	byID := make(map[ActionID]Action, len(copied))
	for _, action := range copied {
		byID[action.ID] = action
	}
	return Registry{actions: copied, byID: byID}
}

func (r Registry) Action(id ActionID) (Action, bool) {
	action, ok := r.byID[id]
	return action, ok
}

func (r Registry) MustBinding(id ActionID) key.Binding {
	action, ok := r.Action(id)
	if !ok {
		panic("unknown keybinding action: " + string(id))
	}
	return action.Binding()
}

func (r Registry) Actions() []Action {
	return append([]Action(nil), r.actions...)
}

func (r Registry) ActionsFor(context Context) []Action {
	actions := make([]Action, 0, len(r.actions))
	for _, action := range r.actions {
		if action.appliesTo(context) {
			actions = append(actions, action)
		}
	}
	sort.SliceStable(actions, func(i, j int) bool {
		return actions[i].Priority > actions[j].Priority
	})
	return actions
}

// ActionsDeclaredFor returns actions assigned directly to a context without
// implicitly adding global actions. It is used to build non-duplicated help
// sections.
func (r Registry) ActionsDeclaredFor(context Context) []Action {
	actions := make([]Action, 0, len(r.actions))
	for _, action := range r.actions {
		if slices.Contains(action.Contexts, context) {
			actions = append(actions, action)
		}
	}
	sort.SliceStable(actions, func(i, j int) bool {
		return actions[i].Priority > actions[j].Priority
	})
	return actions
}

func DefaultRegistry() Registry {
	return NewRegistry([]Action{
		{ActionForceQuit, []Context{ContextGlobal}, []string{"ctrl+c"}, "ctrl+c", "quit immediately", 100},
		{ActionQuit, []Context{ContextGlobal}, []string{"esc"}, "esc esc", "back / quit", 98},
		{ActionGlobalHelp, []Context{ContextGlobal}, []string{"f1"}, "f1", "show all shortcuts", 95},
		{ActionPreviousPanel, []Context{ContextGlobal}, []string{"ctrl+e", "alt+h"}, "ctrl+e/alt+h", "previous panel", 90},
		{ActionNextPanel, []Context{ContextGlobal}, []string{"ctrl+w", "alt+l"}, "ctrl+w/alt+l", "next panel", 90},

		{ActionContextHelp, []Context{ContextSidebar, ContextRequest, ContextResponse}, []string{"?"}, "?", "show context help", 75},

		{ActionLoadRequest, []Context{ContextSidebar}, []string{"enter"}, "enter", "open request", 70},
		{ActionDeleteRequest, []Context{ContextSidebar}, []string{"d"}, "d", "delete request", 65},
		{ActionNavigationCount, []Context{ContextSidebar}, []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}, "0-9", "prefix movement count", 62},
		{ActionNavigateUp, []Context{ContextSidebar}, []string{"k"}, "k", "move up", 60},
		{ActionNavigateDown, []Context{ContextSidebar}, []string{"j"}, "j", "move down", 60},
		{ActionNavigateFirst, []Context{ContextSidebar}, []string{"g"}, "g", "first request", 55},
		{ActionNavigateLast, []Context{ContextSidebar}, []string{"G"}, "G", "last request", 55},

		{ActionSubmit, []Context{ContextRequest}, []string{"alt+enter"}, "alt+enter", "send request", 70},
		{ActionSaveRequest, []Context{ContextRequest}, []string{"ctrl+s"}, "ctrl+s", "save request", 65},
		{ActionToggleLoadTest, []Context{ContextRequest}, []string{"ctrl+l"}, "ctrl+l", "toggle load test", 65},
		{ActionNextField, []Context{ContextRequest}, []string{"tab"}, "tab", "next field", 60},
		{ActionPreviousField, []Context{ContextRequest}, []string{"shift+tab"}, "shift+tab", "previous field", 60},
		{ActionActivateControl, []Context{ContextRequest}, []string{"enter"}, "enter", "activate focused control", 55},
		{ActionNextMethod, []Context{ContextRequest}, []string{"l"}, "l", "next method", 50},
		{ActionPreviousMethod, []Context{ContextRequest}, []string{"h"}, "h", "previous method", 50},

		{ActionCopyResponse, []Context{ContextResponse}, []string{"y"}, "y", "copy response", 70},
		{ActionCancelLoadTest, []Context{ContextResponse}, []string{"ctrl+x"}, "ctrl+x", "cancel load test", 70},
		{ActionPreviousTab, []Context{ContextResponse}, []string{"h"}, "h", "previous tab", 65},
		{ActionNextTab, []Context{ContextResponse}, []string{"l"}, "l", "next tab", 65},
		{ActionDirectTab, []Context{ContextResponse}, []string{"1", "2", "3"}, "1-3", "jump to tab", 60},
		{ActionScrollUp, []Context{ContextResponse}, []string{"k"}, "k", "scroll up", 55},
		{ActionScrollDown, []Context{ContextResponse}, []string{"j"}, "j", "scroll down", 55},
		{ActionPageUp, []Context{ContextResponse}, []string{"ctrl+u"}, "ctrl+u", "half page up", 50},
		{ActionPageDown, []Context{ContextResponse}, []string{"ctrl+d"}, "ctrl+d", "half page down", 50},

		{ActionCloseHelp, []Context{ContextHelp}, []string{"q", "?"}, "q/?", "close help", 80},
		{ActionPreviousHelpTab, []Context{ContextHelp}, []string{"h"}, "h", "previous section", 70},
		{ActionNextHelpTab, []Context{ContextHelp}, []string{"l"}, "l", "next section", 70},
		{ActionDirectHelpTab, []Context{ContextHelp}, []string{"1", "2", "3", "4"}, "1-4", "jump to section", 60},
	})
}
