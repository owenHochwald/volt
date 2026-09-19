package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

// preferenceSavedMsg reports the outcome of persisting an auto-remember
// setting (see saveAutoRememberCookiesCmd/saveAutoRememberAuthHeadersCmd) -
// mirrors themeSelectionSavedMsg, but a failure here only affects what
// the *next* restart starts with (the setting is already applied to this
// session's responsePane regardless - see the Toggle*Msg handling in
// update.go), so it's reported as a quieter warning notification rather
// than blocking anything.
type preferenceSavedMsg struct {
	label string
	err   error
}

func saveAutoRememberCookiesCmd(enabled bool) tea.Cmd {
	return func() tea.Msg {
		return preferenceSavedMsg{
			label: "Auto-remember cookies",
			err:   design.SaveAutoRememberCookies(enabled),
		}
	}
}

func saveAutoRememberAuthHeadersCmd(enabled bool) tea.Cmd {
	return func() tea.Msg {
		return preferenceSavedMsg{
			label: "Auto-remember auth headers",
			err:   design.SaveAutoRememberAuthHeaders(enabled),
		}
	}
}

func (m *Model) handlePreferenceSaved(msg preferenceSavedMsg) {
	if msg.err == nil {
		return
	}
	m.notification = ui.Notification{
		Level: ui.NotificationWarning,
		Text:  msg.label + " setting couldn't be saved",
		Hint:  msg.err.Error(),
	}
}
