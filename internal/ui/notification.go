package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/apperror"
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

func (n Notification) View(width int) string {
	if strings.TrimSpace(n.Text) == "" {
		return ""
	}

	label := strings.ToUpper(string(n.Level))
	color := lipgloss.Color("39")
	switch n.Level {
	case NotificationSuccess:
		color = lipgloss.Color("42")
	case NotificationWarning:
		color = lipgloss.Color("214")
	case NotificationError:
		color = lipgloss.Color("196")
	}

	prefix := lipgloss.NewStyle().Bold(true).Foreground(color).Render(" " + label + " ")
	content := lipgloss.JoinHorizontal(lipgloss.Left, prefix, " ", n.Text)
	if strings.TrimSpace(n.Hint) != "" {
		hint := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" · " + n.Hint)
		content = lipgloss.JoinHorizontal(lipgloss.Left, content, hint)
	}
	return lipgloss.NewStyle().
		Width(max(width, 1)).
		MaxWidth(max(width, 1)).
		MaxHeight(1).
		Render(content)
}
