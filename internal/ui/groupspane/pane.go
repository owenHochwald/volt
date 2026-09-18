package groupspane

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/storage"
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
	modeAssignServer
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

	keys keybindings.KeyMap
	db   *storage.SQLiteStorage

	target focusTarget
	mode   mode

	groups  list.Model
	servers list.Model

	nameInput    textinput.Model
	urlInput     textinput.Model
	formFocusURL bool

	// headersInput edits a server's default headers (see modeEditServerHeaders)
	// as comma-separated "key = value" pairs, one per line - the same format
	// requestpane's own Headers textarea uses, parsed with the same
	// utils.ParseKeyValuePairs/ParseMapToString.
	headersInput textarea.Model

	// renamingName/assigningGroup remember which item a rename/URL-edit/
	// server-assignment in progress applies to (captured by name, since
	// SQLite CRUD here is keyed by name, not id).
	renamingName   string
	assigningGroup string

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
	ti.Width = width
	return ti
}

// SetupGroupsPane constructs an unopened GroupsPane; call Open() to load
// data and make it ready to display.
func SetupGroupsPane(db *storage.SQLiteStorage, keys keybindings.KeyMap) GroupsPane {
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
		groups:       groups,
		servers:      servers,
		nameInput:    newTextInput("name", 40, 40),
		urlInput:     newTextInput("http://host:port", 200, 40),
		headersInput: headersInput,
	}
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
	servers, _ := m.db.ListServers()

	serverNames := make(map[int64]string, len(servers))
	for _, s := range servers {
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

	serverItems := make([]list.Item, 0, len(servers))
	for _, s := range servers {
		serverItems = append(serverItems, serverItem{server: s})
	}
	m.servers.SetItems(serverItems)
	m.servers.Title = fmt.Sprintf("Servers (%d)", len(serverItems))
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
