package app

import (
	"context"

	tea "charm.land/bubbletea/v2"
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
	// setFocusedPanel), so a zoomed response never silently stays zoomed
	// once you've moved on from it.
	zoomedResponse bool

	width, height int

	loadTestUpdates  <-chan *http.LoadTestStats
	loadTestCancel   context.CancelFunc
	loadTestCanceled bool
	showHelpModal    bool
	showGroupsModal  bool
	showImportModal  bool
	notification     ui.Notification
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
		width:         80,
		height:        24,
	}
	m.setFocusedPanel(focusedPanel)
	m.applyLayout(m.currentLayout())
	return m
}

func (m Model) Init() tea.Cmd {
	return m.sidebarPane.Init()
}

// resizeContentPanels re-applies the current layout (see layout.go's
// currentLayout/applyLayout) to every panel - called on every
// WindowSizeMsg, and also whenever something else changes what
// currentLayout computes without a terminal resize of its own (e.g.
// ToggleZoomResponse, "z") - otherwise these panels' own persisted size
// (view.go's own rendering doesn't mutate it back - see requestPane.
// SetHeight's own comment) would stay stale until the next real resize,
// silently throwing off things like pgup/pgdn's scroll-by-page size.
func (m *Model) resizeContentPanels() {
	m.applyLayout(m.currentLayout())
}
