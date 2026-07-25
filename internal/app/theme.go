package app

import (
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/shortcutpane"
)

func (m *Model) applyTheme(theme design.Theme, source string) {
	styles := design.NewStyles(theme)

	m.theme = theme
	m.styles = styles
	if source == "" {
		source = theme.Name
	}
	m.themeSource = source

	m.sidebarPane.SetStyles(styles)
	m.requestPane.SetStyles(styles)
	m.responsePane.SetStyles(styles)
	m.headerPane.SetStyles(styles)
	m.shortcutPane.SetStyles(styles)
}

func (m Model) settingsThemeOptions() []shortcutpane.ThemeOption {
	adaptive := design.AdaptiveTheme(true)
	if m.themeSource == "adaptive" {
		adaptive = m.theme
	}
	options := []shortcutpane.ThemeOption{
		{
			Name:        "Controlled Voltage",
			Description: "Violet focus, charge-yellow action, and a dark instrument canvas.",
			Source:      "default",
			Theme:       design.DefaultTheme(),
		},
		{
			Name:        "Adaptive",
			Description: "Tracks the terminal background while preserving Volt semantics.",
			Source:      "adaptive",
			Theme:       adaptive,
		},
		{
			Name:        "Mono",
			Description: "No-color compatibility with structural focus and explicit states.",
			Source:      "mono",
			Theme:       design.MonoTheme(),
		},
	}
	if !isBuiltInTheme(m.themeSource) {
		options = append(options, shortcutpane.ThemeOption{
			Name:        m.theme.Name,
			Description: "The custom YAML theme loaded when Volt started.",
			Source:      m.themeSource,
			Theme:       m.theme,
		})
	}
	return options
}

func isBuiltInTheme(source string) bool {
	switch source {
	case "default", "adaptive", "mono":
		return true
	default:
		return false
	}
}
