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

	width, height int

	loadTestUpdates <-chan *http.LoadTestStats
	showHelpModal   bool
	showGroupsModal bool
	showImportModal bool
}

func SetupModel(db *storage.SQLiteStorage) Model {
	keys := keybindings.DefaultKeyMap()
	responsePane := responsepane.SetupResponsePane(keys)
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
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return m.sidebarPane.Init()
}
