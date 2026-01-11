package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/volt/internal/http"
	"github.com/owenHochwald/volt/internal/ui"
	"github.com/owenHochwald/volt/internal/ui/keybindings"
	"github.com/owenHochwald/volt/internal/ui/requestpane"
	"github.com/owenHochwald/volt/internal/ui/responsepane"
	"github.com/owenHochwald/volt/internal/ui/shortcutpane"
	"github.com/owenHochwald/volt/internal/utils"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle '?' to toggle help modal
		if keybindings.Matches(msg, m.keys.ShowHelp) && !m.showHelpModal && m.focusedPanel != utils.RequestPanel {
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
		if keybindings.Matches(msg, m.keys.LoadRequest) {
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

	case http.ResultMsg:
		m.requestPane.ResultMsgCleanup()
		m.responsePane.SetResponse(msg.Response)
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

		// Existing size handling for other panels
		m.sidebarPane.SetSize(m.width/2, (m.height-15)/2)
	}

	// Existing panel update routing (only when help modal is closed)
	if !m.showHelpModal {
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
