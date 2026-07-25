package shortcutpane

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

type PreviewThemeMsg struct {
	Theme  design.Theme
	Source string
}

type SaveThemeMsg struct {
	Theme  design.Theme
	Source string
}

type CancelThemePreviewMsg struct{}

func (m *ShortcutPane) SetThemeOptions(options []ThemeOption, activeSource string) {
	m.themeOptions = append(m.themeOptions[:0], options...)
	m.themeIndex = 0
	for index, option := range m.themeOptions {
		if option.Source == activeSource {
			m.themeIndex = index
			break
		}
	}
	m.savedTheme = m.themeIndex
	m.previewing = false
}

func (m *ShortcutPane) BeginSession() {
	m.surface = SurfaceHelp
	m.savedTheme = m.themeIndex
	m.previewing = false
}

func (m ShortcutPane) ActiveSurface() Surface {
	return m.surface
}

func (m *ShortcutPane) moveTheme(delta int) tea.Cmd {
	if len(m.themeOptions) == 0 {
		return nil
	}
	m.themeIndex = (m.themeIndex + delta) % len(m.themeOptions)
	if m.themeIndex < 0 {
		m.themeIndex += len(m.themeOptions)
	}
	m.previewing = m.themeIndex != m.savedTheme
	option := m.themeOptions[m.themeIndex]
	return func() tea.Msg {
		return PreviewThemeMsg{Theme: option.Theme, Source: option.Source}
	}
}

func (m *ShortcutPane) saveTheme() tea.Cmd {
	if len(m.themeOptions) == 0 {
		return nil
	}
	m.savedTheme = m.themeIndex
	m.previewing = false
	option := m.themeOptions[m.themeIndex]
	return func() tea.Msg {
		return SaveThemeMsg{Theme: option.Theme, Source: option.Source}
	}
}

func (m *ShortcutPane) cancelThemePreview() tea.Cmd {
	if !m.previewing {
		return nil
	}
	m.themeIndex = m.savedTheme
	m.previewing = false
	return func() tea.Msg {
		return CancelThemePreviewMsg{}
	}
}

func (m ShortcutPane) activeThemeOption() (ThemeOption, bool) {
	if m.themeIndex < 0 || m.themeIndex >= len(m.themeOptions) {
		return ThemeOption{}, false
	}
	return m.themeOptions[m.themeIndex], true
}

func isBuiltInThemeSource(source string) bool {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "default", "adaptive", "mono":
		return true
	default:
		return false
	}
}

func (m ShortcutPane) renderSettings() string {
	lines := []string{m.styles.Text.Logo.Render("APPEARANCE"), ""}
	for index, option := range m.themeOptions {
		marker := m.styles.Text.Muted.Render("│")
		nameStyle := m.styles.Text.Value
		if index == m.themeIndex {
			marker = m.styles.Text.Logo.Render("┃")
			nameStyle = m.styles.Text.Logo
		}

		kind := "custom YAML"
		if isBuiltInThemeSource(option.Source) {
			kind = "built-in"
		}
		state := ""
		if index == m.savedTheme {
			state = m.styles.Notice.Success.Render("  ACTIVE")
		}
		line := lipgloss.JoinHorizontal(
			lipgloss.Left,
			marker,
			" ",
			nameStyle.Render(option.Name),
			m.styles.Text.Muted.Render("  "+kind),
			state,
		)
		lines = append(lines, line)
		if index == m.themeIndex && option.Description != "" {
			lines = append(lines, m.styles.Text.Muted.Render("  "+option.Description))
		}
	}

	if option, ok := m.activeThemeOption(); ok {
		lines = append(
			lines,
			"",
			m.styles.Text.Label.Render("LIVE PREVIEW"),
			lipgloss.JoinHorizontal(
				lipgloss.Left,
				m.styles.Text.Logo.Render("┃ FOCUS"),
				"  ",
				m.styles.Action.Focused.Render("SEND"),
				"  ",
				m.styles.Notice.Success.Render("✓ 200"),
				"  ",
				m.styles.Notice.Warning.Render("! WARN"),
				"  ",
				m.styles.Notice.Error.Render("× ERROR"),
				"  ",
				m.styles.Method.GET.Render("GET"),
			),
			m.styles.Text.Muted.Render("source: "+option.Source),
		)
	}

	return strings.Join(lines, "\n")
}
