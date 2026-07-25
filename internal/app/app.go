package app

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/buildinfo"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ui/requestpane"
	"github.com/owenHochwald/Volt/internal/ui/responsepane"
	"github.com/owenHochwald/Volt/internal/ui/shortcutpane"
	"github.com/owenHochwald/Volt/internal/utils"
)

type Model struct {
	db     *storage.SQLiteStorage
	keys   keybindings.KeyMap
	theme  design.Theme
	styles design.Styles

	sidebarPane  *ui.SidebarPane
	requestPane  requestpane.RequestPane
	responsePane *responsepane.ResponsePane
	headerPane   *ui.Header
	shortcutPane shortcutpane.ShortcutPane

	savedRequests []http.Request

	focusedPanel utils.Panel

	width, height int

	loadTestUpdates  <-chan *http.LoadTestStats
	loadTestCancel   context.CancelFunc
	loadTestCanceled bool
	showHelpModal    bool
	notification     ui.Notification
	quitArmed        bool
	quitSequence     uint64
}

func SetupModel(db *storage.SQLiteStorage) Model {
	keys := keybindings.DefaultKeyMap()
	theme := design.DefaultTheme()
	styles := design.NewStyles(theme)
	responsePane := responsepane.SetupResponsePane(keys, styles)
	shortcutPane := shortcutpane.SetupShortcutPane(keys, styles)

	m := Model{
		db:            db,
		keys:          keys,
		theme:         theme,
		styles:        styles,
		sidebarPane:   ui.NewSidebar(db, keys, styles),
		requestPane:   requestpane.SetupRequestPane(db, keys, styles),
		responsePane:  &responsePane,
		shortcutPane:  shortcutPane,
		focusedPanel:  utils.SidebarPanel,
		headerPane:    ui.SetupHeader(buildinfo.Version(), styles),
		showHelpModal: false,
		width:         80,
		height:        24,
	}
	m.setFocusedPanel(utils.SidebarPanel)
	m.applyLayout(calculateLayout(m.width, m.height))
	return m
}

func (m Model) Init() tea.Cmd {
	return m.sidebarPane.Init()
}
