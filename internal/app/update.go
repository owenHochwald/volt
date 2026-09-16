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

		// Global key handling (only when modal is closed)
		if keybindings.Matches(msg, m.keys.CyclePanel) {
			m.focusedPanel = (m.focusedPanel + 1) % 3
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
		if keybindings.Matches(msg, m.keys.EscapePanel) {
			if m.focusedPanel == utils.RequestPanel {
				m.focusedPanel = utils.SidebarPanel
				return m, nil
			}
		}
		if keybindings.Matches(msg, m.keys.LoadRequest) && !m.sidebarPane.IsFiltering() {
			// Guarded on IsFiltering(): LoadRequest is bound to enter/space,
			// which while filtering must type a space or confirm the query
			// instead of immediately loading whatever's selected.
			if m.focusedPanel == utils.SidebarPanel {
				if item, ok := m.sidebarPane.SelectedItem(); ok {
					m.focusedPanel = utils.RequestPanel
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
		m.focusedPanel = utils.ResponsePanel
		return m, nil

	case ui.RequestSavedMsg:
		if msg.Err != nil {
			return m, nil
		}
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
		m.focusedPanel = utils.ResponsePanel // Switch focus to results
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

		// Existing size handling for other panels. sidebarWidth/
		// sidebarContentHeight are shared with view.go's rendering (see
		// layout.go) - they must match the box the sidebar is actually
		// rendered into, otherwise the list wraps its content to fit a
		// differently-sized box than it thinks it has, so the wrapped line
		// count (and therefore the sidebar's rendered height) drifts as
		// different items are selected, visibly shifting the right panel.
		m.sidebarPane.SetSize(m.sidebarWidth(), m.sidebarContentHeight())

		// requestPane.Height/Width/FieldsViewport are plain value fields,
		// unlike its *textinput.Model etc. fields - view.go's calls to
		// SetHeight/SetWidth only ever mutate View()'s own local, discarded
		// copy (value receiver), so without this, requestPane.Update() (a
		// pgup/pgdown scroll, say) would always see FieldsViewport at its
		// zero value and be unable to scroll at all.
		m.requestPane.SetHeight(m.requestHeight())
		m.requestPane.SetWidth(m.requestWidth())
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
