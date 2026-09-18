package groupspane

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

type focusTarget int

const (
	focusGroups focusTarget = iota
	focusServers
)

type mode int

const (
	modeBrowse mode = iota
	modeCreateGroup
	modeRenameGroup
	modeCreateServer
	modeRenameServer
	modeEditServerURL
	modeEditServerHeaders
)

// pendingDeleteKind distinguishes what a pending delete confirmation applies
// to, since groups and servers are removed through different db calls and
// deleting a group also cascades to the requests in it.
type pendingDeleteKind int

const (
	pendingDeleteGroup pendingDeleteKind = iota
	pendingDeleteServer
)

// pendingDeleteItem remembers which group or server a DeleteItem ('d') press
// applies to while awaiting the next keypress as its y/n answer.
type pendingDeleteItem struct {
	kind pendingDeleteKind
	name string
}

// GroupsPane is the modal for managing request groups and the servers they
// can point at. Groups group saved requests one level deep; a group's
// requests resolve their (possibly relative) URL against its assigned
// server, so reassigning a server redirects the whole group at once.
type GroupsPane struct {
	width, height int
	Focused       bool

	keys   keybindings.KeyMap
	db     *storage.SQLiteStorage
	styles design.Styles

	target focusTarget
	mode   mode

	groups list.Model
	// servers shows only the servers belonging to whichever group is
	// currently selected/highlighted in groups (see refreshGroupServers)
	// - what browsing displays, and what rename/delete/URL/headers edits
	// operate on. A server created here ("n" while target==focusServers)
	// is likewise always connected to that selected group right away
	// (see handleServerFormKey) - there's no server that exists outside
	// of some group's scope from this UI's perspective.
	servers list.Model

	nameInput    textinput.Model
	urlInput     textinput.Model
	formFocusURL bool

	// headersInput edits a server's default headers (see modeEditServerHeaders)
	// as comma-separated "key = value" pairs, one per line - the same format
	// requestpane's own Headers textarea uses, parsed with the same
	// utils.ParseKeyValuePairs/ParseMapToString.
	headersInput textarea.Model

	// renamingName remembers which item a rename/URL-edit in progress
	// applies to (captured by name, since SQLite CRUD here is keyed by
	// name, not id).
	renamingName string

	// pendingDelete holds the group/server awaiting a y/n confirmation after
	// DeleteItem ('d') was pressed, or nil when nothing is pending. Deleting
	// is irreversible (deleting a group also deletes the requests in it), so
	// it's confirmed rather than acted on immediately.
	pendingDelete *pendingDeleteItem

	statusMsg string
	statusErr bool
}

func newTextInput(placeholder string, charLimit, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	ti.SetWidth(width)
	return ti
}

// SetupGroupsPane constructs an unopened GroupsPane; call Open() to load
// data and make it ready to display.
func SetupGroupsPane(db *storage.SQLiteStorage, keys keybindings.KeyMap, optionalStyles ...design.Styles) GroupsPane {
	styles := design.NewStyles(design.DefaultTheme())
	if len(optionalStyles) > 0 {
		styles = optionalStyles[0]
	}

	groups := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	groups.Title = "Groups"
	groups.SetShowHelp(false)

	servers := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	servers.Title = "Servers"
	servers.SetShowHelp(false)

	headersInput := textarea.New()
	headersInput.Placeholder = "Authorization = Bearer ...,\nX-Env = prod,"
	headersInput.SetWidth(40)

	return GroupsPane{
		height:       25,
		width:        70,
		keys:         keys,
		db:           db,
		styles:       styles,
		groups:       groups,
		servers:      servers,
		nameInput:    newTextInput("name", 40, 40),
		urlInput:     newTextInput("http://host:port", 200, 40),
		headersInput: headersInput,
	}
}

// SetStyles updates the theme-derived styles this pane renders with (see
// design.Styles) - called whenever the app's theme changes, the same way
// every other pane picks up a live theme switch.
func (m *GroupsPane) SetStyles(styles design.Styles) {
	m.styles = styles
}

func (m GroupsPane) Init() tea.Cmd {
	return nil
}

// Open resets the pane to its browse state and reloads groups/servers from
// the database - call this every time the modal is opened, since data may
// have changed since it was last shown (e.g. via `volt import` from a
// separate terminal, or a prior open of this same modal).
func (m *GroupsPane) Open() {
	m.mode = modeBrowse
	m.Focused = true
	m.statusMsg = ""
	m.statusErr = false
	m.pendingDelete = nil
	m.reload()
}

func (m *GroupsPane) reload() {
	groups, _ := m.db.ListGroups()
	allServers, _ := m.db.ListServers()

	serverNames := make(map[int64]string, len(allServers))
	for _, s := range allServers {
		serverNames[s.ID] = s.Name
	}

	groupItems := make([]list.Item, 0, len(groups))
	for _, g := range groups {
		serverLabel := "(none)"
		if g.ServerID != nil {
			if name, ok := serverNames[*g.ServerID]; ok {
				serverLabel = name
			}
		}
		groupItems = append(groupItems, groupItem{group: g, server: serverLabel})
	}
	m.groups.SetItems(groupItems)
	m.groups.Title = fmt.Sprintf("Groups (%d)", len(groupItems))

	m.refreshGroupServers()
}

// refreshGroupServers repopulates the Servers list from whichever group is
// currently selected/highlighted in Groups (see db.ListGroupServers) -
// browsing only ever shows that group's own servers, not every server in
// the database. Called after reload and after any keystroke that could
// have moved the Groups selection (see handleBrowseKey), so the two never
// drift out of sync.
func (m *GroupsPane) refreshGroupServers() {
	group, ok := m.selectedGroup()
	if !ok {
		m.servers.SetItems(nil)
		m.servers.Title = "Servers (0)"
		return
	}
	servers, _ := m.db.ListGroupServers(group.group.ID)
	serverItems := make([]list.Item, 0, len(servers))
	for _, s := range servers {
		serverItems = append(serverItems, serverItem{server: s})
	}
	m.servers.SetItems(serverItems)
	m.servers.Title = fmt.Sprintf("Servers (%d) · %s", len(serverItems), group.group.Name)
}

// IsEditing reports whether a create/rename/edit form is currently
// capturing keystrokes, as opposed to browsing the groups/servers lists.
// Used by the embedding shortcutpane (see shortcutpane/update.go) to decide
// whether a global key (the surface switch, the whole-modal close) should
// intercept a keypress, or let it reach the focused field untouched -
// otherwise typing e.g. "h" or "q" into a name field would be swallowed as
// a surface switch or a close instead of the literal character.
func (m GroupsPane) IsEditing() bool {
	return m.mode != modeBrowse
}

func (m *GroupsPane) SetFocused(focused bool) {
	m.Focused = focused
}

func (m *GroupsPane) SetWidth(width int) {
	m.width = width
	m.resize()
}

func (m *GroupsPane) SetHeight(height int) {
	m.height = height
	m.resize()
}

func (m *GroupsPane) resize() {
	innerWidth := max(m.width-8, 20)
	listWidth := innerWidth/2 - 2
	listHeight := max(m.height-14, 5)
	m.groups.SetSize(listWidth, listHeight)
	m.servers.SetSize(listWidth, listHeight)
	m.headersInput.SetWidth(innerWidth)
}
