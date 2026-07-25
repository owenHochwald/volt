package design

import "charm.land/lipgloss/v2"

type Styles struct {
	Panel  PanelStyles
	Tabs   TabStyles
	Action ActionStyles
	Text   TextStyles
	Header HeaderStyles
	Badge  BadgeStyles
	Notice NoticeStyles
	Method MethodStyles
	Status StatusStyles
}

type PanelStyles struct {
	Base, Focused, Running lipgloss.Style
	Header, Sidebar        lipgloss.Style
}

type TabStyles struct {
	Active, Inactive, Disabled lipgloss.Style
}

type ActionStyles struct {
	Primary, Focused, Busy, Disabled, Destructive lipgloss.Style
}

type TextStyles struct {
	Label, Value, Muted, Logo  lipgloss.Style
	ResponseKey, ResponseLabel lipgloss.Style
	Faint                      lipgloss.Style
}

type HeaderStyles struct {
	Logo, Metadata lipgloss.Style
}

type BadgeStyles struct {
	Info, Success, Warning, Error, Live lipgloss.Style
}

type NoticeStyles struct {
	Info, Success, Warning, Error lipgloss.Style
}

type MethodStyles struct {
	GET, POST, PUT, PATCH, DELETE lipgloss.Style
}

type StatusStyles struct {
	Success, Redirect, ClientError, ServerError, Unknown lipgloss.Style
}

func NewStyles(theme Theme) Styles {
	panelBase := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Colors.Border)

	methodBase := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true).
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.Colors.Border)

	badgeBase := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)

	return Styles{
		Panel: PanelStyles{
			Base:    panelBase,
			Focused: panelBase.BorderForeground(theme.Colors.Brand).Bold(true),
			Running: panelBase.BorderForeground(theme.Colors.Charge).Bold(true),
			Header:  panelBase.Height(2),
			Sidebar: panelBase.Width(20),
		},
		Tabs: TabStyles{
			Active: lipgloss.NewStyle().
				Padding(0, 2).
				Background(theme.Colors.BrandStrong).
				Foreground(theme.Colors.Text).
				Bold(true),
			Inactive: lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(theme.Colors.TextMuted),
			Disabled: lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(theme.Colors.TextMuted).
				Faint(true),
		},
		Action: ActionStyles{
			Primary: lipgloss.NewStyle().
				Foreground(theme.Colors.Charge).
				Bold(true),
			Focused: lipgloss.NewStyle().
				Background(theme.Colors.Charge).
				Foreground(theme.Colors.Canvas).
				Padding(0, 1).
				Bold(true),
			Busy: lipgloss.NewStyle().
				Foreground(theme.Colors.Charge).
				Bold(true),
			Disabled: lipgloss.NewStyle().
				Foreground(theme.Colors.TextMuted).
				Faint(true),
			Destructive: lipgloss.NewStyle().
				Foreground(theme.Colors.Error).
				Bold(true),
		},
		Text: TextStyles{
			Label:         lipgloss.NewStyle().Foreground(theme.Colors.TextMuted),
			Value:         lipgloss.NewStyle().Foreground(theme.Colors.Text),
			Muted:         lipgloss.NewStyle().Foreground(theme.Colors.TextMuted),
			Logo:          lipgloss.NewStyle().Foreground(theme.Colors.Brand).Bold(true),
			ResponseKey:   lipgloss.NewStyle().Foreground(theme.Colors.Brand).Bold(true),
			ResponseLabel: lipgloss.NewStyle().Foreground(theme.Colors.TextMuted).Bold(true),
			Faint:         lipgloss.NewStyle().Foreground(theme.Colors.TextMuted).Faint(true),
		},
		Header: HeaderStyles{
			Logo:     lipgloss.NewStyle().Foreground(theme.Colors.Brand).Bold(true),
			Metadata: lipgloss.NewStyle().Foreground(theme.Colors.TextMuted),
		},
		Badge: BadgeStyles{
			Info:    badgeBase.Background(theme.Colors.Info).Foreground(theme.Colors.Canvas),
			Success: badgeBase.Background(theme.Colors.Success).Foreground(theme.Colors.Canvas),
			Warning: badgeBase.Background(theme.Colors.Warning).Foreground(theme.Colors.Canvas),
			Error:   badgeBase.Background(theme.Colors.Error).Foreground(theme.Colors.Canvas),
			Live:    badgeBase.Background(theme.Colors.Charge).Foreground(theme.Colors.Canvas),
		},
		Notice: NoticeStyles{
			Info:    lipgloss.NewStyle().Foreground(theme.Colors.Info).Bold(true),
			Success: lipgloss.NewStyle().Foreground(theme.Colors.Success).Bold(true),
			Warning: lipgloss.NewStyle().Foreground(theme.Colors.Warning).Bold(true),
			Error:   lipgloss.NewStyle().Foreground(theme.Colors.Error).Bold(true),
		},
		Method: MethodStyles{
			GET:    methodBase.Foreground(theme.Colors.MethodGET),
			POST:   methodBase.Foreground(theme.Colors.MethodPOST),
			PUT:    methodBase.Foreground(theme.Colors.MethodPUT),
			PATCH:  methodBase.Foreground(theme.Colors.MethodPATCH),
			DELETE: methodBase.Foreground(theme.Colors.MethodDELETE),
		},
		Status: StatusStyles{
			Success:     lipgloss.NewStyle().Foreground(theme.Colors.Success),
			Redirect:    lipgloss.NewStyle().Foreground(theme.Colors.Signal),
			ClientError: lipgloss.NewStyle().Foreground(theme.Colors.Warning),
			ServerError: lipgloss.NewStyle().Foreground(theme.Colors.Error),
			Unknown:     lipgloss.NewStyle().Foreground(theme.Colors.Brand),
		},
	}
}

func (s PanelStyles) ForState(focused, running bool) lipgloss.Style {
	if running {
		return s.Running
	}
	if focused {
		return s.Focused
	}
	return s.Base
}
