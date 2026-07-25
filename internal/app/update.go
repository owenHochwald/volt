package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/apperror"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/design"
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

// Update prevents an unexpected UI panic from terminating Volt. The recovery
// message intentionally omits implementation details and stack traces.
func (m Model) Update(msg tea.Msg) (model tea.Model, cmd tea.Cmd) {
	return recoverUpdate(m, func() (tea.Model, tea.Cmd) {
		return m.update(msg)
	})
}

func recoverUpdate(m Model, update func() (tea.Model, tea.Cmd)) (model tea.Model, cmd tea.Cmd) {
	model = m
	defer func() {
		if recover() != nil {
			m.notification = ui.ErrorNotification(apperror.ApplicationError())
			model = m
			cmd = nil
		}
	}()
	return update()
}

func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		if m.themeSource == "adaptive" {
			m.applyTheme(design.AdaptiveTheme(msg.IsDark()), "adaptive")
		}
		return m, nil

	case tea.KeyPressMsg:
		if keybindings.Matches(msg, m.keys.ForceQuit) {
			if m.loadTestCancel != nil {
				m.loadTestCancel()
			}
			return m, tea.Quit
		}

		if keybindings.Matches(msg, m.keys.Quit) {
			if m.showHelpModal {
				m.closeHelp()
				return m, nil
			}
			if m.quitArmed {
				if m.loadTestCancel != nil {
					m.loadTestCancel()
				}
				return m, tea.Quit
			}

			if m.focusedPanel != utils.SidebarPanel {
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

	case shortcutpane.PreviewThemeMsg:
		m.applyTheme(msg.Theme, msg.Source)
		if msg.Source == "adaptive" {
			return m, tea.RequestBackgroundColor
		}
		return m, nil

	case shortcutpane.SaveThemeMsg:
		m.saveThemeSession(msg.Theme, msg.Source)
		m.notification = ui.Notification{
			Level: ui.NotificationSuccess,
			Text:  "Theme changed to " + msg.Theme.Name,
		}
		saveCmd := saveThemeSelectionCmd(msg.Source, msg.Theme.Name)
		if msg.Source == "adaptive" {
			return m, tea.Batch(saveCmd, tea.RequestBackgroundColor)
		}
		return m, saveCmd

	case shortcutpane.CancelThemePreviewMsg:
		m.closeHelp()
		m.notification = ui.Notification{
			Level: ui.NotificationInfo,
			Text:  "Theme preview canceled",
		}
		return m, nil

	case themeSelectionSavedMsg:
		if msg.err != nil {
			m.notification = ui.Notification{
				Level: ui.NotificationWarning,
				Text:  "Theme is active but couldn't be saved",
				Hint:  msg.err.Error(),
			}
			return m, nil
		}
		m.notification = ui.Notification{
			Level: ui.NotificationSuccess,
			Text:  "Theme saved: " + msg.themeName,
		}
		return m, nil

	case quitSequenceExpiredMsg:
		if m.quitArmed && msg.sequence == m.quitSequence {
			m.disarmQuit()
		}
		return m, nil

	case http.ResultMsg:
		m.requestPane.ResultMsgCleanup()
		m.setFocusedPanel(utils.ResponsePanel)
		if msg.Response == nil {
			failure := apperror.OperationError("Volt didn't receive a response.", "Try the request again.")
			msg.Response = &http.Response{Error: failure.Message, Failure: failure}
		}
		if msg.Response.Error != "" {
			failure := msg.Response.Failure
			if failure == nil {
				failure = apperror.FromNetwork(errors.New(msg.Response.Error))
			}
			msg.Response.Error = failure.Message
			msg.Response.Failure = failure
			m.responsePane.SetResponse(msg.Response)
			m.notification = ui.ErrorNotification(failure)
			return m, nil
		}

		m.responsePane.SetResponse(msg.Response)
		switch {
		case msg.Response.StatusCode >= 400:
			m.notification = ui.ErrorNotification(apperror.HTTPStatus(msg.Response.StatusCode))
		default:
			m.notification = ui.Notification{Level: ui.NotificationSuccess, Text: "Request completed with " + msg.Response.Status}
		}
		return m, nil

	case ui.RequestSavedMsg:
		if msg.Err != nil {
			m.notification = ui.ErrorNotification(apperror.FromStorage(msg.Err))
			return m, nil
		}
		m.notification = ui.Notification{Level: ui.NotificationSuccess, Text: "Request saved"}
		return m, ui.LoadRequestsCmd(m.db)

	case ui.RequestDeletedMsg:
		if msg.Err != nil {
			m.notification = ui.ErrorNotification(apperror.FromStorage(msg.Err))
			return m, nil
		}
		m.notification = ui.Notification{Level: ui.NotificationSuccess, Text: "Request deleted"}
		return m, ui.LoadRequestsCmd(m.db)

	case ui.RequestsLoadingMsg:
		if msg.Err != nil {
			m.notification = ui.ErrorNotification(apperror.FromStorage(msg.Err))
			return m, nil
		}
		m.sidebarPane, cmd = m.sidebarPane.Update(msg)
		return m, cmd

	case ui.NotificationMsg:
		m.notification = msg.Notification
		return m, nil

	case responsepane.ResponseCopiedMsg:
		if msg.Err != nil {
			m.notification = ui.ErrorNotification(apperror.OperationError("Volt couldn't copy the response.", "Try again after checking clipboard access."))
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
			m.notification = ui.ErrorNotification(apperror.OperationError("Load test ended without final results.", "Try the load test again."))
		case msg.Stats.FailedRequests > 0:
			m.notification = ui.ErrorNotification(apperror.LoadTestFailure(msg.Stats.FailedRequests, msg.Stats.Errors))
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
		m.notification = ui.ErrorNotification(apperror.OperationError("Volt couldn't start the load test.", "Check the load test settings and try again."))
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
