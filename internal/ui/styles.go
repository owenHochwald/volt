package ui

import (
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

var (
	defaultStyles = design.NewStyles(design.DefaultTheme())

	InactiveTab = defaultStyles.Tabs.Inactive
	ActiveTab   = defaultStyles.Tabs.Active

	FocusedButton   = defaultStyles.Action.Focused
	UnfocusedButton = defaultStyles.Action.Primary

	HelpStyle = lipgloss.NewStyle().
			Margin(1, 2)

	HeaderStyle   = defaultStyles.Panel.Header
	SidebarStyle  = defaultStyles.Panel.Sidebar
	RequestStyle  = defaultStyles.Panel.Base
	ResponseStyle = defaultStyles.Panel.Base

	LabelStyle = defaultStyles.Text.Label

	LoadTestBorderStyle = defaultStyles.Panel.Running
)

func ApplyFocus(style lipgloss.Style, focus bool) lipgloss.Style {
	if focus {
		return style.
			BorderForeground(defaultStyles.Panel.Focused.GetBorderLeftForeground()).
			Bold(true)
	}
	return style.BorderForeground(defaultStyles.Panel.Base.GetBorderLeftForeground())
}
