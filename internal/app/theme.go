package app

import "github.com/owenHochwald/Volt/internal/ui/design"

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
