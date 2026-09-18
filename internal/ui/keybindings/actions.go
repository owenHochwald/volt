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
	// ContextGroups/ContextImport are the Groups & Servers modal (ctrl+g)
	// and the interactive OpenAPI import modal (ctrl+n) - each its own
	// context (not ContextGlobal) since their bindings only apply while
	// that modal is open, not everywhere.
	ContextGroups Context = "Groups"
	ContextImport Context = "Import"
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

	ActionOpenGroups ActionID = "open_groups"
	ActionOpenImport ActionID = "open_import"

	ActionResetRequest     ActionID = "reset_request"
	ActionClearParams      ActionID = "clear_params"
	ActionScrollFieldsUp   ActionID = "scroll_fields_up"
	ActionScrollFieldsDown ActionID = "scroll_fields_down"

	ActionToggleRawBody      ActionID = "toggle_raw_body"
	ActionToggleWrap         ActionID = "toggle_wrap"
	ActionSaveResponseRaw    ActionID = "save_response_raw"
	ActionGotoTop            ActionID = "goto_top"
	ActionGotoBottom         ActionID = "goto_bottom"
	ActionGotoLine           ActionID = "goto_line"
	ActionSearchResponse     ActionID = "search_response"
	ActionSearchNext         ActionID = "search_next"
	ActionSearchPrev         ActionID = "search_prev"
	ActionSearchNextFromPos  ActionID = "search_next_from_position"
	ActionSearchPrevFromPos  ActionID = "search_prev_from_position"
	ActionPickBodyStyle      ActionID = "pick_body_style"
	ActionToggleZoomResponse ActionID = "toggle_zoom_response"

	ActionCloseGroups ActionID = "close_groups"
	ActionCloseImport ActionID = "close_import"
	ActionCreateItem  ActionID = "create_item"
	ActionRenameItem  ActionID = "rename_item"
	ActionDeleteItem  ActionID = "delete_item"
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

		{ActionSubmit, []Context{ContextRequest}, []string{"alt+enter", "ctrl+p"}, "alt+enter", "send request", 70},
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
		{ActionDirectTab, []Context{ContextResponse}, []string{"1", "2", "3", "4"}, "1-4", "jump to tab", 60},
		{ActionScrollUp, []Context{ContextResponse}, []string{"k"}, "k", "scroll up", 55},
		{ActionScrollDown, []Context{ContextResponse}, []string{"j"}, "j", "scroll down", 55},
		{ActionPageUp, []Context{ContextResponse}, []string{"ctrl+u"}, "ctrl+u", "half page up", 50},
		{ActionPageDown, []Context{ContextResponse}, []string{"ctrl+d"}, "ctrl+d", "half page down", 50},

		{ActionCloseHelp, []Context{ContextHelp}, []string{"q", "?"}, "q/?", "close help", 80},
		{ActionPreviousHelpTab, []Context{ContextHelp}, []string{"h"}, "h", "previous section", 70},
		{ActionNextHelpTab, []Context{ContextHelp}, []string{"l"}, "l", "next section", 70},
		{ActionDirectHelpTab, []Context{ContextHelp}, []string{"1", "2", "3", "4", "5", "6"}, "1-6", "jump to section", 60},

		{ActionOpenGroups, []Context{ContextGlobal}, []string{"ctrl+g"}, "ctrl+g", "groups & servers", 40},
		{ActionOpenImport, []Context{ContextGlobal}, []string{"ctrl+n"}, "ctrl+n", "import OpenAPI spec", 35},

		{ActionResetRequest, []Context{ContextRequest}, []string{"ctrl+r"}, "ctrl+r", "reset request", 45},
		{ActionClearParams, []Context{ContextRequest}, []string{"ctrl+x"}, "ctrl+x", "clear params", 44},
		{ActionScrollFieldsUp, []Context{ContextRequest}, []string{"pgup"}, "pgup", "scroll fields up", 40},
		{ActionScrollFieldsDown, []Context{ContextRequest}, []string{"pgdown"}, "pgdown", "scroll fields down", 40},

		{ActionToggleRawBody, []Context{ContextResponse}, []string{"r"}, "r", "toggle raw body", 48},
		{ActionToggleWrap, []Context{ContextResponse}, []string{"w"}, "w", "toggle wrap", 47},
		{ActionSaveResponseRaw, []Context{ContextResponse}, []string{"s"}, "s", "save raw body to file", 46},
		{ActionToggleZoomResponse, []Context{ContextResponse}, []string{"z"}, "z", "zoom response", 45},
		{ActionPickBodyStyle, []Context{ContextResponse}, []string{"t"}, "t", "pick body style", 44},
		{ActionSearchResponse, []Context{ContextResponse}, []string{"/"}, "/", "search", 43},
		{ActionSearchNext, []Context{ContextResponse}, []string{"n"}, "n", "next match", 42},
		{ActionSearchPrev, []Context{ContextResponse}, []string{"b"}, "b", "previous match", 41},
		{ActionSearchNextFromPos, []Context{ContextResponse}, []string{"N"}, "N", "next match from here", 40},
		{ActionSearchPrevFromPos, []Context{ContextResponse}, []string{"P"}, "P", "previous match from here", 39},
		{ActionGotoTop, []Context{ContextResponse}, []string{"g"}, "gg", "go to top", 38},
		{ActionGotoBottom, []Context{ContextResponse}, []string{"G"}, "G", "go to bottom", 37},
		// GotoLine has no keystroke of its own to match on - typing digits
		// right after "g" (see ActionGotoTop) is what actually triggers it
		// (see ResponsePane.gPending) - this exists purely so it's listed.
		{ActionGotoLine, []Context{ContextResponse}, nil, "{n}g", "go to line n", 36},

		{ActionCloseGroups, []Context{ContextGroups}, []string{"q", "esc"}, "q/esc", "close", 80},
		{ActionCreateItem, []Context{ContextGroups}, []string{"n"}, "n", "new", 70},
		{ActionRenameItem, []Context{ContextGroups}, []string{"r"}, "r", "rename", 65},
		{ActionDeleteItem, []Context{ContextGroups}, []string{"d"}, "d", "delete", 60},

		{ActionCloseImport, []Context{ContextImport}, []string{"esc"}, "esc", "close", 80},
	})
}
