package app

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/groupspane"
	"github.com/owenHochwald/Volt/internal/ui/importpane"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ui/responsepane"
	"github.com/owenHochwald/Volt/internal/ui/shortcutpane"
	"github.com/owenHochwald/Volt/internal/utils"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if keybindings.Matches(msg, m.keys.ForceQuit) {
			if m.loadTestCancel != nil {
				m.loadTestCancel()
			}
			return m, tea.Quit
		}

		noModalOpen := !m.showHelpModal && !m.showGroupsModal && !m.showImportModal

		if m.showHelpModal {
			if keybindings.Matches(msg, m.keys.GlobalHelp) {
				m.closeHelp()
				return m, nil
			}
			m.shortcutPane, cmd = m.shortcutPane.Update(msg)
			return m, cmd
		}

		if keybindings.Matches(msg, m.keys.GlobalHelp) {
			m.openHelp(keybindings.ContextGlobal)
			return m, nil
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
			m.groupsPane, cmd = m.groupsPane.Update(msg)
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
			m.importPane, cmd = m.importPane.Update(msg)
			return m, cmd
		}

		isEditing := m.focusedPanel == utils.RequestPanel && m.requestPane.IsEditing()
		if keybindings.Matches(msg, m.keys.ContextHelp) && !isEditing {
			m.openHelp(m.focusedContext())
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.PreviousPanel) {
			m.movePanel(-1)
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.NextPanel) {
			m.movePanel(1)
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.CancelLoadTest) && m.loadTestCancel != nil {
			m.loadTestCanceled = true
			m.loadTestCancel()
			m.notification = ui.Notification{Level: ui.NotificationWarning, Text: "Canceling load test…"}
			return m, nil
		}
		if keybindings.Matches(msg, m.keys.Quit) &&
			(m.focusedPanel == utils.SidebarPanel || m.focusedPanel == utils.ResponsePanel) {
			if m.loadTestCancel != nil {
				m.loadTestCancel()
			}
			return m, tea.Quit
		}
		// "z" expands the response panel to the whole right column,
		// hiding the request panel, while focused on it - the same key
		// un-zooms it again, and so does leaving the response panel any
		// other way (see setFocusedPanel). Meaningless (and left
		// unhandled) anywhere else.
		if keybindings.Matches(msg, m.keys.ToggleZoomResponse) && noModalOpen && m.focusedPanel == utils.ResponsePanel {
			m.zoomedResponse = !m.zoomedResponse
			m.resizeContentPanels()
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
			// resizeContentPanels' own math starting this line - the next
			// WindowSizeMsg (a real terminal resize) would eventually
			// recompute this anyway, but that might be a long time from
			// now, and the picker needs it sized correctly the instant it
			// opens.
			m.resizeContentPanels()
		}
		if keybindings.Matches(msg, m.keys.EscapePanel) {
			// Esc's own save-to-file/search/style-picker overlays (see
			// SaveResponseRaw/SearchResponse/PickBodyStyle) take priority
			// over the panel switch below - otherwise esc could never
			// actually cancel any of them, since this would always
			// intercept it first and switch panels instead (it still
			// reaches responsePane.Update below, which is what actually
			// closes whichever overlay is open).
			overlayOpen := m.focusedPanel == utils.ResponsePanel &&
				(m.responsePane.IsSavingToFile() || m.responsePane.IsSearching() || m.responsePane.IsPickingStyle())
			if !overlayOpen && m.focusedPanel != utils.SidebarPanel {
				m.setFocusedPanel(utils.SidebarPanel)
				return m, nil
			}
		}
		if keybindings.Matches(msg, m.keys.LoadRequest) && !m.sidebarPane.IsFiltering() {
			// Guarded on IsFiltering(): LoadRequest is bound to enter/space,
			// which while filtering must type a space or confirm the query
			// instead of immediately loading whatever's selected.
			if m.focusedPanel == utils.SidebarPanel {
				if item, ok := m.sidebarPane.SelectedItem(); ok {
					m.setFocusedPanel(utils.RequestPanel)
					return m, ui.SetRequestPaneRequestCmd(item.Request)
				}
			}
		}
	case shortcutpane.CloseHelpModalMsg:
		m.closeHelp()
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
		m.importPane, cmd = m.importPane.Update(msg)
		return m, cmd

	case http.ResultMsg:
		m.requestPane.ResultMsgCleanup()
		m.responsePane.SetResponse(msg.Response, msg.Request)
		m.setFocusedPanel(utils.ResponsePanel)
		switch {
		case msg.Response == nil:
			m.notification = ui.Notification{Level: ui.NotificationError, Text: "Request failed: empty response"}
		case msg.Response.Error != "":
			m.notification = ui.Notification{Level: ui.NotificationError, Text: "Request failed: " + msg.Response.Error}
		case msg.Response.StatusCode >= 400:
			m.notification = ui.Notification{Level: ui.NotificationError, Text: "Request completed with " + msg.Response.Status}
		default:
			m.notification = ui.Notification{Level: ui.NotificationSuccess, Text: "Request completed with " + msg.Response.Status}
		}
		return m, nil

	case ui.RequestSavedMsg:
		if msg.Err != nil {
			m.notification = ui.Notification{Level: ui.NotificationError, Text: "Save failed: " + msg.Err.Error()}
			return m, nil
		}
		m.requestPane.MarkSaved(msg.Request)
		m.notification = ui.Notification{Level: ui.NotificationSuccess, Text: "Request saved"}
		return m, ui.LoadRequestsCmd(m.db)

	case ui.RequestDeletedMsg:
		if msg.Err != nil {
			m.notification = ui.Notification{Level: ui.NotificationError, Text: "Delete failed: " + msg.Err.Error()}
			return m, nil
		}
		m.notification = ui.Notification{Level: ui.NotificationSuccess, Text: "Request deleted"}
		return m, ui.LoadRequestsCmd(m.db)

	case ui.RequestsLoadingMsg:
		if msg.Err != nil {
			m.notification = ui.Notification{Level: ui.NotificationError, Text: "Loading requests failed: " + msg.Err.Error()}
			return m, nil
		}
		m.sidebarPane, cmd = m.sidebarPane.Update(msg)
		return m, cmd

	case ui.NotificationMsg:
		m.notification = msg.Notification
		return m, nil

	case responsepane.ResponseCopiedMsg:
		if msg.Err != nil {
			m.notification = ui.Notification{Level: ui.NotificationError, Text: "Copy failed: " + msg.Err.Error()}
		} else {
			m.notification = ui.Notification{Level: ui.NotificationSuccess, Text: "Response copied"}
		}
		return m, nil

	case http.LoadTestStartMsg:
		updates := make(chan *http.LoadTestStats, 100)
		m.loadTestUpdates = updates
		runContext, cancel := context.WithCancel(context.Background())
		m.loadTestCancel = cancel
		m.loadTestCanceled = false

		go func() {
			msg.Config.Run(runContext, updates)
		}()

		m.responsePane.SetLoadTestPending(msg.Config.TotalRequests)
		m.notification = ui.Notification{Level: ui.NotificationInfo, Text: "Load test started"}
		m.setFocusedPanel(utils.ResponsePanel)
		return m, ui.WaitForLoadTestUpdatesCmd(updates, msg.Config.TotalRequests)

	case http.LoadTestStatsMsg:
		m.responsePane.SetLoadTestStats(msg.Stats)

		if m.loadTestUpdates != nil {
			return m, ui.WaitForLoadTestUpdatesCmd(m.loadTestUpdates, msg.Stats.TotalRequests)
		}
		return m, nil

	case http.LoadTestCompleteMsg:
		m.loadTestUpdates = nil
		m.loadTestCancel = nil
		if msg.Stats != nil {
			m.responsePane.SetLoadTestStats(msg.Stats)
		}
		m.requestPane.ExitLoadTestMode()
		m.setFocusedPanel(utils.ResponsePanel)
		switch {
		case m.loadTestCanceled:
			completed := 0
			if msg.Stats != nil {
				completed = msg.Stats.CompletedRequests
			}
			m.notification = ui.Notification{
				Level: ui.NotificationWarning,
				Text:  fmt.Sprintf("Load test canceled after %d requests", completed),
			}
		case msg.Stats == nil:
			m.notification = ui.Notification{Level: ui.NotificationError, Text: "Load test ended without final statistics"}
		case msg.Stats.FailedRequests > 0:
			m.notification = ui.Notification{
				Level: ui.NotificationError,
				Text:  fmt.Sprintf("Load test complete: %d failed requests", msg.Stats.FailedRequests),
			}
		default:
			m.notification = ui.Notification{
				Level: ui.NotificationSuccess,
				Text:  fmt.Sprintf("Load test complete: %d requests", msg.Stats.CompletedRequests),
			}
		}
		m.loadTestCanceled = false
		return m, nil

	case http.LoadTestErrorMsg:
		m.loadTestUpdates = nil
		if m.loadTestCancel != nil {
			m.loadTestCancel()
			m.loadTestCancel = nil
		}
		m.requestPane.ExitLoadTestMode()
		m.setFocusedPanel(utils.ResponsePanel)
		errorText := "unknown error"
		if msg.Error != nil {
			errorText = msg.Error.Error()
		}
		m.notification = ui.Notification{Level: ui.NotificationError, Text: "Load test failed: " + errorText}
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizeContentPanels()
	}

	// Existing panel update routing (only when no modal is open)
	if !m.showHelpModal && !m.showGroupsModal && !m.showImportModal {
		if m.focusedPanel == utils.SidebarPanel {
			m.sidebarPane, cmd = m.sidebarPane.Update(msg)
			return m, cmd
		} else if m.focusedPanel == utils.RequestPanel {
			m.requestPane, cmd = m.requestPane.Update(msg)
			return m, cmd
		} else if m.focusedPanel == utils.ResponsePanel {
			m.responsePane, cmd = m.responsePane.Update(msg)
			return m, cmd
		}
	}

	return m, cmd
}
