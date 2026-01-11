package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/volt/internal/http"
	"github.com/owenHochwald/volt/internal/storage"
	"github.com/owenHochwald/volt/internal/ui"
	"github.com/owenHochwald/volt/internal/ui/keybindings"
	"github.com/owenHochwald/volt/internal/ui/requestpane"
	"github.com/owenHochwald/volt/internal/ui/responsepane"
	"github.com/owenHochwald/volt/internal/ui/shortcutpane"
	"github.com/owenHochwald/volt/internal/utils"
)

type Model struct {
	db   *storage.SQLiteStorage
	keys keybindings.KeyMap

	sidebarPane  *ui.SidebarPane
	requestPane  requestpane.RequestPane
	responsePane *responsepane.ResponsePane
	headerPane   *ui.Header
	shortcutPane shortcutpane.ShortcutPane

	savedRequests []http.Request

	focusedPanel utils.Panel

	width, height int

	loadTestUpdates <-chan *http.LoadTestStats
	showHelpModal   bool
}

func SetupModel(db *storage.SQLiteStorage) Model {
	keys := keybindings.DefaultKeyMap()
	responsePane := responsepane.SetupResponsePane(keys)
	shortcutPane := shortcutpane.SetupShortcutPane(keys)

	m := Model{
		db:            db,
		keys:          keys,
		sidebarPane:   ui.NewSidebar(db, keys),
		requestPane:   requestpane.SetupRequestPane(db, keys),
		responsePane:  &responsePane,
		shortcutPane:  shortcutPane,
		focusedPanel:  utils.SidebarPanel,
		headerPane:    ui.SetupHeader(),
		showHelpModal: false,
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return m.sidebarPane.Init()
}
