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
	db          *storage.SQLiteStorage
	keys        keybindings.KeyMap
	theme       design.Theme
	styles      design.Styles
	themeSource string

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
	themeSession     design.ThemeLoadResult
	themeSessionOpen bool
	quitArmed        bool
	quitSequence     uint64
}

func SetupModel(db *storage.SQLiteStorage, optionalAppearance ...design.ThemeLoadResult) Model {
	keys := keybindings.DefaultKeyMap()
	appearance := design.ThemeLoadResult{
		Theme:  design.DefaultTheme(),
		Source: "default",
	}
	if len(optionalAppearance) > 0 {
		appearance = optionalAppearance[0]
	}
	theme := appearance.Theme
	styles := design.NewStyles(theme)
	responsePane := responsepane.SetupResponsePane(keys, styles)
	shortcutPane := shortcutpane.SetupShortcutPane(keys, styles)

	m := Model{
		db:            db,
		keys:          keys,
		theme:         theme,
		styles:        styles,
		themeSource:   appearance.Source,
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
	if appearance.Warning != "" {
		m.notification = ui.Notification{
			Level: ui.NotificationWarning,
			Text:  appearance.Warning,
		}
	}
	m.setFocusedPanel(utils.SidebarPanel)
	m.applyLayout(calculateLayout(m.width, m.height))
	return m
}

func (m Model) Init() tea.Cmd {
	if m.themeSource == "adaptive" {
		return tea.Batch(m.sidebarPane.Init(), tea.RequestBackgroundColor)
	}
	return m.sidebarPane.Init()
}
