package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/groupspane"
	"github.com/owenHochwald/Volt/internal/ui/importpane"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ui/requestpane"
	"github.com/owenHochwald/Volt/internal/ui/responsepane"
	"github.com/owenHochwald/Volt/internal/ui/shortcutpane"
	"github.com/owenHochwald/Volt/internal/utils"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		noModalOpen := !m.showHelpModal && !m.showGroupsModal && !m.showImportModal

		// Handle '?' to toggle help modal. Also stays out of the way while
		// searching the sidebar, since '?' is a legitimate character to
		// filter a URL's query string by.
		if keybindings.Matches(msg, m.keys.ShowHelp) && noModalOpen && m.focusedPanel != utils.RequestPanel && !m.sidebarPane.IsFiltering() {
			m.showHelpModal = true
			m.shortcutPane.SetFocused(true)
			return m, nil
		}

		// If help modal is open, route ALL messages to it
		if m.showHelpModal {
			var shortcutModel tea.Model
			shortcutModel, cmd = m.shortcutPane.Update(msg)
			m.shortcutPane = shortcutModel.(shortcutpane.ShortcutPane)
			return m, cmd
		}

		// Handle ctrl+g to toggle the groups & servers manager. Unlike '?',
		// this doesn't collide with typing, so it works from any panel.
		if keybindings.Matches(msg, m.keys.OpenGroups) && noModalOpen {
			m.showGroupsModal = true
			m.groupsPane.Open()
			return m, nil
		}

		// If the groups modal is open, route ALL messages to it
		if m.showGroupsModal {
			var groupsModel tea.Model
			groupsModel, cmd = m.groupsPane.Update(msg)
			m.groupsPane = groupsModel.(groupspane.GroupsPane)
			return m, cmd
		}

		// Handle ctrl+i to toggle the OpenAPI import dialog. Unlike '?',
		// this doesn't collide with typing, so it works from any panel.
		if keybindings.Matches(msg, m.keys.OpenImport) && noModalOpen {
			m.showImportModal = true
			m.importPane.Open()
			return m, nil
		}

		// If the import modal is open, route ALL messages to it
		if m.showImportModal {
			var importModel tea.Model
			importModel, cmd = m.importPane.Update(msg)
			m.importPane = importModel.(importpane.ImportPane)
			return m, cmd
		}

		// Global key handling (only when modal is closed). CyclePanel is
		// bound to shift+tab, which the request pane also binds to its own
		// PrevField - without this exclusion, shift+tab inside the request
		// pane always cycled the app's panel focus instead of ever reaching
		// PrevField, so shift-tabbing backward through its fields silently
		// did nothing (moving focus away entirely, not just skipping a
		// field).
		if keybindings.Matches(msg, m.keys.CyclePanel) && m.focusedPanel != utils.RequestPanel {
			m.focusPanel((m.focusedPanel + 1) % 3)
			return m, nil
		}
		// f1/f2/f3 jump directly to a panel from anywhere, including
		// mid-edit in the request pane - unlike CyclePanel (shift+tab),
		// which the request pane also needs for its own PrevField and so
		// can't reliably reach a panel switch from there.
		if keybindings.Matches(msg, m.keys.FocusSidebar) && noModalOpen {
			m.focusPanel(utils.SidebarPanel)
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.FocusRequest) && noModalOpen {
			m.focusPanel(utils.RequestPanel)
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.FocusResponse) && noModalOpen {
			m.focusPanel(utils.ResponsePanel)
			return m, nil
		}
		// "f4" shows/hides the top banner from anywhere, the panels below
		// it taking over whatever space it frees (see layout.go's
		// headerHeight) - persisted so the next app start restores
		// whatever visibility was last chosen.
		if keybindings.Matches(msg, m.keys.ToggleHeader) && noModalOpen {
			m.showHeader = !m.showHeader
			m.resizeContentPanels()
			return m, ui.SaveShowHeaderCmd(m.db, m.showHeader)
		}
		if keybindings.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
		// "z" expands the response panel to the whole right column,
		// hiding the request panel, while focused on it - the same key
		// (checked before the panel-switch bindings above so it doesn't
		// need its own noModalOpen/focus juggling) un-zooms it again, and
		// so does leaving the response panel any other way (see
		// focusPanel). Meaningless (and left unhandled) anywhere else.
		if keybindings.Matches(msg, m.keys.ToggleZoomResponse) && noModalOpen && m.focusedPanel == utils.ResponsePanel {
			m.zoomedResponse = !m.zoomedResponse
			return m, nil
		}
		// "t" (PickBodyStyle) puts the style picker's list in the request
		// pane's own slot (see view.go, responsepane.ResponsePane.
		// StylePickerListView) - a zoomed response panel has no such slot
		// to show it in, so un-zoom first. Deliberately doesn't return -
		// the key still needs to reach responsePane.Update below (via the
		// normal routing at the bottom of this function), which is what
		// actually opens the picker.
		if keybindings.Matches(msg, m.keys.PickBodyStyle) && m.focusedPanel == utils.ResponsePanel {
			m.zoomedResponse = false
			// Un-zooming just now (above) only takes effect in
			// requestHeight/requestWidth's own math starting this line -
			// the next WindowSizeMsg (a real terminal resize) would
			// eventually recompute this anyway, but that might be a long
			// time from now, and the picker needs it sized correctly the
			// instant it opens.
			m.responsePane.SetStylePickerSize(m.requestWidth(), m.requestHeight())
		}
		if keybindings.Matches(msg, m.keys.EscapePanel) {
			if m.focusedPanel == utils.RequestPanel {
				m.focusPanel(utils.SidebarPanel)
				return m, nil
			}
			// Esc's own save-to-file/search/style-picker overlays (see
			// SaveResponseRaw/SearchResponse/PickBodyStyle) take priority
			// over the panel switch below - otherwise esc could never
			// actually cancel any of them, since this would always
			// intercept it first and switch panels instead.
			if m.focusedPanel == utils.ResponsePanel && !m.responsePane.IsSavingToFile() && !m.responsePane.IsSearching() && !m.responsePane.IsPickingStyle() {
				m.focusPanel(utils.RequestPanel)
				return m, nil
			}
		}
		if keybindings.Matches(msg, m.keys.LoadRequest) && !m.sidebarPane.IsFiltering() {
			// Guarded on IsFiltering(): LoadRequest is bound to enter/space,
			// which while filtering must type a space or confirm the query
			// instead of immediately loading whatever's selected.
			if m.focusedPanel == utils.SidebarPanel {
				if item, ok := m.sidebarPane.SelectedItem(); ok {
					m.focusPanel(utils.RequestPanel)
					return m, ui.SetRequestPaneRequestCmd(item.Request)
				}
			}
		}
	case shortcutpane.CloseHelpModalMsg:
		m.showHelpModal = false
		m.shortcutPane.SetFocused(false)
		return m, nil

	case groupspane.CloseGroupsModalMsg:
		m.showGroupsModal = false
		m.groupsPane.SetFocused(false)
		// Group renames/assignments made in the modal only take effect in
		// the sidebar's "[group] ..." labels once requests are reloaded.
		return m, ui.LoadRequestsCmd(m.db)

	case importpane.CloseImportModalMsg:
		m.showImportModal = false
		m.importPane.SetFocused(false)
		// Refresh the sidebar in case the import added new requests.
		return m, ui.LoadRequestsCmd(m.db)

	case importpane.ImportResultMsg:
		var importModel tea.Model
		importModel, cmd = m.importPane.Update(msg)
		m.importPane = importModel.(importpane.ImportPane)
		return m, cmd

	case http.ResultMsg:
		m.requestPane.ResultMsgCleanup()
		m.responsePane.SetResponse(msg.Response, msg.Request)
		m.focusPanel(utils.ResponsePanel)
		return m, nil

	case ui.RequestSavedMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.requestPane.MarkSaved(msg.Request)
		return m, ui.LoadRequestsCmd(m.db)

	case ui.RequestDeletedMsg:
		if msg.Err != nil {
			return m, nil
		}
		return m, ui.LoadRequestsCmd(m.db)

	case ui.RequestsLoadingMsg:
		if msg.Err != nil {
			return m, nil
		}
		var sidebarModel tea.Model
		sidebarModel, cmd = m.sidebarPane.Update(msg)
		m.sidebarPane = sidebarModel.(*ui.SidebarPane)
		return m, cmd

	case http.LoadTestStartMsg:
		updates := make(chan *http.LoadTestStats, 100)
		m.loadTestUpdates = updates

		// start load test in background
		go func() {
			msg.Config.Run(updates)
		}()

		m.responsePane.ClearLoadTestStats()
		return m, ui.WaitForLoadTestUpdatesCmd(updates, msg.Config.TotalRequests)

	case http.LoadTestStatsMsg:
		m.responsePane.SetLoadTestStats(msg.Stats)

		if m.loadTestUpdates != nil {
			return m, ui.WaitForLoadTestUpdatesCmd(m.loadTestUpdates, msg.Stats.TotalRequests)
		}
		return m, nil

	case http.LoadTestCompleteMsg:
		// final update
		m.loadTestUpdates = nil
		if msg.Stats != nil {
			m.responsePane.SetLoadTestStats(msg.Stats)
		}
		m.requestPane.ExitLoadTestMode()
		m.focusPanel(utils.ResponsePanel) // Switch focus to results
		return m, nil

	case http.LoadTestErrorMsg:
		m.loadTestUpdates = nil
		m.requestPane.ExitLoadTestMode()
		// TODO: Display error message
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		// Update shortcut pane dimensions (modal sizing)
		modalWidth := 60
		modalHeight := 25
		if m.width < 80 {
			modalWidth = m.width - 10
		}
		if m.height < 30 {
			modalHeight = m.height - 5
		}
		m.shortcutPane.SetWidth(modalWidth)
		m.shortcutPane.SetHeight(modalHeight)
		m.groupsPane.SetWidth(modalWidth)
		m.groupsPane.SetHeight(modalHeight)
		m.importPane.SetWidth(modalWidth)
		m.importPane.SetHeight(modalHeight)

		m.resizeContentPanels()
	}

	// Existing panel update routing (only when no modal is open)
	if !m.showHelpModal && !m.showGroupsModal && !m.showImportModal {
		if m.focusedPanel == utils.SidebarPanel {
			var sidebarPaneModel tea.Model
			sidebarPaneModel, cmd = m.sidebarPane.Update(msg)
			m.sidebarPane = sidebarPaneModel.(*ui.SidebarPane)
			return m, cmd
		} else if m.focusedPanel == utils.RequestPanel {
			m.requestPane.SetFocused(true)
			var requestPaneModel tea.Model
			requestPaneModel, cmd = m.requestPane.Update(msg)
			m.requestPane = requestPaneModel.(requestpane.RequestPane)
			return m, cmd
		} else if m.focusedPanel == utils.ResponsePanel {
			var responsePaneModel tea.Model
			responsePaneModel, cmd = m.responsePane.Update(msg)
			m.responsePane = responsePaneModel.(*responsepane.ResponsePane)
			return m, cmd
		}
	}

	return m, cmd
}
