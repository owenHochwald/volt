package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/groupspane"
	"github.com/owenHochwald/Volt/internal/ui/importpane"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ui/requestpane"
	"github.com/owenHochwald/Volt/internal/ui/responsepane"
	"github.com/owenHochwald/Volt/internal/ui/shortcutpane"
	"github.com/owenHochwald/Volt/internal/utils"
)

type Model struct {
	db   *storage.SQLiteStorage
	keys keybindings.KeyMap

	sidebarPane  *ui.SidebarPane
	requestPane  requestpane.RequestPane
	responsePane *responsepane.ResponsePane
	headerPane   *ui.Header
	shortcutPane shortcutpane.ShortcutPane
	groupsPane   groupspane.GroupsPane
	importPane   importpane.ImportPane

	savedRequests []http.Request

	focusedPanel utils.Panel

	// zoomedResponse expands the response panel to the right column's
	// full height, hiding the request panel (see ToggleZoomResponse,
	// "z") - always reset whenever focus leaves the response panel (see
	// focusPanel), so a zoomed response never silently stays zoomed once
	// you've moved on from it.
	zoomedResponse bool

	// showHeader shows/hides the top banner (logo + help hint) - toggled
	// by ToggleHeader ("f4"), persisted (see storage.SaveShowHeader/
	// LoadShowHeader, loaded once at SetupModel) so the next app start
	// restores whatever visibility was last chosen.
	showHeader bool

	width, height int

	loadTestUpdates <-chan *http.LoadTestStats
	showHelpModal   bool
	showGroupsModal bool
	showImportModal bool
}

func SetupModel(db *storage.SQLiteStorage) Model {
	keys := keybindings.DefaultKeyMap()
	responsePane := responsepane.SetupResponsePane(db, keys)
	shortcutPane := shortcutpane.SetupShortcutPane(keys)
	groupsPane := groupspane.SetupGroupsPane(db, keys)
	importPane := importpane.SetupImportPane(db, keys)
	requestPane := requestpane.SetupRequestPane(db, keys)

	// Restore the last request actually sent (see requestpane's
	// handleSubmit, which is what saves it in the first place - never a
	// request that was merely being edited, so nothing to restore means
	// either a fresh database or one where nothing's been sent yet, not
	// an error) so the pane picks up exactly where the previous run left
	// off, including every field that was filled in - not just which
	// request was open.
	focusedPanel := utils.SidebarPanel
	if db != nil {
		if lastSent, err := db.LoadLastSentRequest(); err == nil && lastSent != nil {
			requestPane.ApplySnapshot(lastSent)
			focusedPanel = utils.RequestPanel
		}
	}

	showHeader := true
	if db != nil {
		if show, err := db.LoadShowHeader(); err == nil {
			showHeader = show
		}
	}

	m := Model{
		db:            db,
		keys:          keys,
		sidebarPane:   ui.NewSidebar(db, keys),
		requestPane:   requestPane,
		responsePane:  &responsePane,
		shortcutPane:  shortcutPane,
		groupsPane:    groupsPane,
		importPane:    importPane,
		focusedPanel:  focusedPanel,
		headerPane:    ui.SetupHeader(),
		showHelpModal: false,
		showHeader:    showHeader,
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return m.sidebarPane.Init()
}

// resizeContentPanels re-sizes the sidebar, request pane, and style
// picker to whatever sidebarWidth/sidebarContentHeight/requestHeight/
// requestWidth (see layout.go) currently compute - called on every
// WindowSizeMsg, and also whenever something else changes what those
// functions return without a terminal resize of its own (see
// ToggleHeader, "f4") - otherwise these panels' own persisted
// size (view.go's own calls to SetHeight/SetWidth only ever mutate
// View()'s local, discarded copy - see requestPane.SetHeight's own
// comment) would stay stale until the next real resize, silently
// throwing off things like pgup/pgdn's scroll-by-page size.
func (m *Model) resizeContentPanels() {
	// sidebarWidth/sidebarContentHeight are shared with view.go's
	// rendering (see layout.go) - they must match the box the sidebar is
	// actually rendered into, otherwise the list wraps its content to fit
	// a differently-sized box than it thinks it has, so the wrapped line
	// count (and therefore the sidebar's rendered height) drifts as
	// different items are selected, visibly shifting the right panel.
	m.sidebarPane.SetSize(m.sidebarWidth(), m.sidebarContentHeight())

	m.requestPane.SetHeight(m.requestHeight())
	m.requestPane.SetWidth(m.requestWidth())

	// The style picker's list (see PickBodyStyle) is sized to this same
	// box - it's rendered there in its place while open (see view.go) -
	// so it needs to track resizes the same way.
	m.responsePane.SetStylePickerSize(m.requestWidth(), m.requestHeight())
}

// focusPanel switches focus to p - the one place update.go's various
// focus-changing key handlers assign m.focusedPanel, so that leaving the
// response panel by any of them (esc, f1/f2, shift+tab, loading a
// request from the sidebar, a response arriving, ...) reliably un-zooms
// it (see zoomedResponse/ToggleZoomResponse) too, rather than needing
// every one of those call sites to separately remember to.
func (m *Model) focusPanel(p utils.Panel) {
	m.focusedPanel = p
	if p != utils.ResponsePanel {
		m.zoomedResponse = false
	}
}
