package app

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/authjar"
	"github.com/owenHochwald/Volt/internal/buildinfo"
	"github.com/owenHochwald/Volt/internal/history"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/importpane"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ui/requestpane"
	"github.com/owenHochwald/Volt/internal/ui/responsepane"
	"github.com/owenHochwald/Volt/internal/ui/shortcutpane"
	"github.com/owenHochwald/Volt/internal/ui/wspane"
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
	importPane   importpane.ImportPane
	wsPane       wspane.WSPane

	savedRequests []http.Request

	focusedPanel utils.Panel

	// zoomedResponse expands the response panel to the right column's
	// full height, hiding the request panel (see ToggleZoomResponse,
	// "z") - always reset whenever focus leaves the response panel (see
	// setFocusedPanel), so a zoomed response never silently stays zoomed
	// once you've moved on from it.
	zoomedResponse bool

	width, height int
	startupFrame  int

	loadTestUpdates  <-chan *http.LoadTestStats
	loadTestCancel   context.CancelFunc
	loadTestCanceled bool
	showHelpModal    bool
	showImportModal  bool
	showWSModal      bool
	notification     ui.Notification
	themeSession     design.ThemeLoadResult
	themeSessionOpen bool
	quitArmed        bool
	quitSequence     uint64

	// autoRememberCookies/autoRememberAuthHeaders mirror what's persisted
	// in config.yaml (see design.LoadAutoRememberCookies/
	// LoadAutoRememberAuthHeaders) and what responsePane currently has set
	// (see SetAutoRememberCookies/SetAutoRememberAuthHeaders) - kept here
	// too so a Toggle*Msg (see shortcutpane.ToggleAutoRememberCookiesMsg)
	// and the Settings surface being reopened later both have a single
	// source of truth to read from (see openHelp/openGroups).
	autoRememberCookies     bool
	autoRememberAuthHeaders bool
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
	shortcutPane := shortcutpane.SetupShortcutPane(keys, db, styles)
	importPane := importpane.SetupImportPane(db, keys)
	wsPane := wspane.SetupWSPane(keys)
	requestPane := requestpane.SetupRequestPane(db, keys, styles)

	// jar is shared between requestPane (which injects what's remembered
	// into outgoing requests) and responsePane (which captures/views/
	// clears it) - in memory only, for this process' lifetime, never
	// persisted (see internal/authjar).
	jar := authjar.NewStore()
	requestPane.SetJar(jar)
	responsePane.SetJar(jar)

	// history remembers a bounded number of past sends this session - in
	// memory only, per the user's explicit choice, cleared on every
	// restart (see internal/history) - browsed from the response pane
	// alone, so only it needs a reference.
	responsePane.SetHistory(history.NewStore())

	// autoRememberCookies/autoRememberAuthHeaders default to false (a
	// fresh install never starts silently remembering session state) -
	// LoadAutoRemember* returning an error (a malformed config.yaml) is
	// treated the same as "unset", the same tolerant fallback
	// design.LoadUserTheme itself uses.
	autoRememberCookies, _ := design.LoadAutoRememberCookies()
	autoRememberAuthHeaders, _ := design.LoadAutoRememberAuthHeaders()
	responsePane.SetAutoRememberCookies(autoRememberCookies)
	responsePane.SetAutoRememberAuthHeaders(autoRememberAuthHeaders)

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
		db:                      db,
		keys:                    keys,
		theme:                   theme,
		styles:                  styles,
		themeSource:             appearance.Source,
		sidebarPane:             ui.NewSidebar(db, keys, styles),
		requestPane:             requestPane,
		responsePane:            &responsePane,
		shortcutPane:            shortcutPane,
		importPane:              importPane,
		wsPane:                  wsPane,
		focusedPanel:            focusedPanel,
		headerPane:              ui.SetupHeader(buildinfo.Version(), styles),
		showHelpModal:           false,
		width:                   minTerminalWidth,
		height:                  minTerminalHeight,
		autoRememberCookies:     autoRememberCookies,
		autoRememberAuthHeaders: autoRememberAuthHeaders,
	}
	if appearance.Warning != "" {
		m.notification = ui.Notification{
			Level: ui.NotificationWarning,
			Text:  appearance.Warning,
		}
	}
	m.setFocusedPanel(focusedPanel)
	m.applyLayout(m.currentLayout())
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.sidebarPane.Init(),
		startupAdvanceCmd(ui.HeaderFrameCompressed, m.theme.Motion),
	}
	if m.themeSource == "adaptive" {
		cmds = append(cmds, tea.RequestBackgroundColor)
	}
	return tea.Batch(cmds...)
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
