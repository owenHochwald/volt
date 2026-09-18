package importpane

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

type mode int

const (
	modeForm mode = iota
	modeResult
)

// ImportPane is the modal for importing request URLs from a local OpenAPI
// 3.x spec, optionally into a request group (created if missing).
type ImportPane struct {
	width, height int
	Focused       bool

	keys keybindings.KeyMap
	db   *storage.SQLiteStorage

	mode       mode
	focusIndex int // 0=path, 1=group, 2=base URL, 3=force toggle

	pathInput    textinput.Model
	groupInput   textinput.Model
	baseURLInput textinput.Model
	// force, toggled with space while the "force" field is focused,
	// overwrites already-saved requests (same group+method+url) with the
	// spec's current name/headers/body instead of skipping them - the TUI
	// equivalent of `volt import -force`. Left as-is across opens, same as
	// the text fields above, since reimporting into the same group is the
	// common case and a user who wants force-refresh usually wants it every
	// time they're iterating on a spec.
	force bool

	importing bool
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

// SetupImportPane constructs an unopened ImportPane; call Open() before
// displaying it.
func SetupImportPane(db *storage.SQLiteStorage, keys keybindings.KeyMap) ImportPane {
	return ImportPane{
		height:       17,
		width:        70,
		keys:         keys,
		db:           db,
		pathInput:    newTextInput("path/to/openapi.yaml", 500, 60),
		groupInput:   newTextInput("(optional) group name", 40, 60),
		baseURLInput: newTextInput("(optional) http://localhost:3000", 200, 60),
	}
}

func (m ImportPane) Init() tea.Cmd {
	return nil
}

// Open resets the pane to its form, ready for input. Field values are
// intentionally left as-is across opens, since re-importing into the same
// group/server is the common case.
func (m *ImportPane) Open() {
	m.mode = modeForm
	m.Focused = true
	m.importing = false
	m.statusMsg = ""
	m.statusErr = false
	m.focusIndex = 0
	m.pathInput.Focus()
	m.groupInput.Blur()
	m.baseURLInput.Blur()
}

func (m *ImportPane) SetFocused(focused bool) {
	m.Focused = focused
}

func (m *ImportPane) SetWidth(width int) {
	m.width = width
	w := max(width-10, 20)
	m.pathInput.SetWidth(w)
	m.groupInput.SetWidth(w)
	m.baseURLInput.SetWidth(w)
}

func (m *ImportPane) SetHeight(height int) {
	m.height = height
}
