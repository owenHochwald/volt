package app

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/ui/responsepane"
	"github.com/owenHochwald/Volt/internal/ui/shortcutpane"
	"github.com/owenHochwald/Volt/internal/utils"
)

const (
	quitSequenceTimeout = 750 * time.Millisecond
	quitWarningText     = "Press Esc again to quit"
)

type quitSequenceExpiredMsg struct {
	sequence uint64
}

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

		if keybindings.Matches(msg, m.keys.Quit) {
			if m.quitArmed {
				if m.loadTestCancel != nil {
					m.loadTestCancel()
				}
				return m, tea.Quit
			}

			if m.showHelpModal {
				m.closeHelp()
			} else if m.focusedPanel != utils.SidebarPanel {
				m.setFocusedPanel(utils.SidebarPanel)
			}
			return m, m.armQuit()
		}
		m.disarmQuit()

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
		if keybindings.Matches(msg, m.keys.LoadRequest) {
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

	case quitSequenceExpiredMsg:
		if m.quitArmed && msg.sequence == m.quitSequence {
			m.disarmQuit()
		}
		return m, nil

	case http.ResultMsg:
		m.requestPane.ResultMsgCleanup()
		m.responsePane.SetResponse(msg.Response)
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
		m.applyLayout(calculateLayout(m.width, m.height))
	}

	// Existing panel update routing (only when help modal is closed)
	if !m.showHelpModal {
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

func (m *Model) armQuit() tea.Cmd {
	m.quitSequence++
	m.quitArmed = true
	m.notification = ui.Notification{Level: ui.NotificationWarning, Text: quitWarningText}
	sequence := m.quitSequence
	return tea.Tick(quitSequenceTimeout, func(time.Time) tea.Msg {
		return quitSequenceExpiredMsg{sequence: sequence}
	})
}

func (m *Model) disarmQuit() {
	if !m.quitArmed {
		return
	}
	m.quitArmed = false
	m.quitSequence++
	if m.notification.Text == quitWarningText {
		m.notification = ui.Notification{}
	}
}
