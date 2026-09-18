package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/apperror"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

type NotificationLevel string

const (
	NotificationInfo    NotificationLevel = "info"
	NotificationSuccess NotificationLevel = "success"
	NotificationWarning NotificationLevel = "warning"
	NotificationError   NotificationLevel = "error"
)

type Notification struct {
	Level NotificationLevel
	Text  string
	Hint  string
}

type NotificationMsg struct {
	Notification Notification
}

func NotifyCmd(level NotificationLevel, text string) tea.Cmd {
	return func() tea.Msg {
		return NotificationMsg{
			Notification: Notification{
				Level: level,
				Text:  text,
			},
		}
	}
}

func ErrorNotification(err *apperror.Error) Notification {
	if err == nil {
		err = apperror.ApplicationError()
	}
	return Notification{Level: NotificationError, Text: err.Message, Hint: err.Hint}
}

func NotifyErrorCmd(err *apperror.Error) tea.Cmd {
	return func() tea.Msg {
		return NotificationMsg{Notification: ErrorNotification(err)}
	}
}

func (n Notification) View(width int, optionalStyles ...design.Styles) string {
	if strings.TrimSpace(n.Text) == "" {
		return ""
	}

	styles := design.NewStyles(design.DefaultTheme())
	if len(optionalStyles) > 0 {
		styles = optionalStyles[0]
	}

	label := strings.ToUpper(string(n.Level))
	prefixStyle := styles.Notice.Info
	switch n.Level {
	case NotificationSuccess:
		prefixStyle = styles.Notice.Success
	case NotificationWarning:
		prefixStyle = styles.Notice.Warning
	case NotificationError:
		prefixStyle = styles.Notice.Error
	}

	prefix := prefixStyle.Render(" " + label + " ")
	content := lipgloss.JoinHorizontal(lipgloss.Left, prefix, " ", n.Text)
	if strings.TrimSpace(n.Hint) != "" {
		hint := styles.Text.Muted.Render(" · " + n.Hint)
		content = lipgloss.JoinHorizontal(lipgloss.Left, content, hint)
	}
	return lipgloss.NewStyle().
		Width(max(width, 1)).
		MaxWidth(max(width, 1)).
		MaxHeight(1).
		Render(content)
}
