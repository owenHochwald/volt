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

// ToggleAutoRememberCookiesMsg/ToggleAutoRememberAuthHeadersMsg report a
// preference row's new state after ActivateControl (enter) flips it (see
// activateSettingsRow) - handled by app/update.go, which updates its own
// Model field, propagates it to responsepane.ResponsePane (see
// SetAutoRememberCookies/SetAutoRememberAuthHeaders), and persists it
// (see design.SaveAutoRememberCookies/SaveAutoRememberAuthHeaders).
type ToggleAutoRememberCookiesMsg struct{ Enabled bool }
type ToggleAutoRememberAuthHeadersMsg struct{ Enabled bool }

// Settings surface rows after the theme list: preference toggles, each a
// simple immediate-effect on/off (no staged "preview" the way a theme
// row has - see moveSettingsCursor/activateSettingsRow).
const (
	prefRowAutoCookies = iota
	prefRowAutoAuthHeaders
	prefRowCount
)

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
	m.settingsCursor = m.themeIndex
	m.previewing = false
}

// SetPreferences sets the Settings surface's two toggle rows' displayed
// state - called once at startup (see design.LoadAutoRememberCookies/
// LoadAutoRememberAuthHeaders) and refreshed every time the Command
// Center opens (see app.Model.openHelp/openGroups), mirroring how
// SetThemeOptions refreshes the theme list's active entry each time too.
func (m *ShortcutPane) SetPreferences(autoRememberCookies, autoRememberAuthHeaders bool) {
	m.autoRememberCookies = autoRememberCookies
	m.autoRememberAuthHeaders = autoRememberAuthHeaders
}

func (m *ShortcutPane) BeginSession() {
	m.surface = SurfaceHelp
	m.savedTheme = m.themeIndex
	m.settingsCursor = m.themeIndex
	m.previewing = false
}

// totalSettingsRows is every row NavUp/NavDown can land on in the
// Settings surface: the theme list, then the two preference toggles.
func (m ShortcutPane) totalSettingsRows() int {
	return len(m.themeOptions) + prefRowCount
}

// moveSettingsCursor moves settingsCursor by delta, wrapping across every
// row (theme options, then the two preference toggles). Landing on a
// theme row previews it live, exactly as moveTheme always did; landing on
// a preference row previews nothing - there's nothing to stage, enter
// just flips it in place (see activateSettingsRow) - so any theme preview
// already in progress is left exactly as it was, resolved later the same
// way it always would be (enter to save, esc/close to cancel).
func (m *ShortcutPane) moveSettingsCursor(delta int) tea.Cmd {
	total := m.totalSettingsRows()
	if total == 0 {
		return nil
	}
	m.settingsCursor = ((m.settingsCursor+delta)%total + total) % total
	if m.settingsCursor >= len(m.themeOptions) {
		return nil
	}
	m.themeIndex = m.settingsCursor
	m.previewing = m.themeIndex != m.savedTheme
	option := m.themeOptions[m.themeIndex]
	return func() tea.Msg {
		return PreviewThemeMsg{Theme: option.Theme, Source: option.Source}
	}
}

// activateSettingsRow is ActivateControl's (enter) handler for the
// Settings surface - saves the previewed theme if settingsCursor sits on
// a theme row, otherwise flips whichever preference toggle it sits on.
func (m *ShortcutPane) activateSettingsRow() tea.Cmd {
	if m.settingsCursor < len(m.themeOptions) {
		return m.saveTheme()
	}
	switch m.settingsCursor - len(m.themeOptions) {
	case prefRowAutoCookies:
		return m.toggleAutoRememberCookies()
	case prefRowAutoAuthHeaders:
		return m.toggleAutoRememberAuthHeaders()
	}
	return nil
}

func (m *ShortcutPane) toggleAutoRememberCookies() tea.Cmd {
	m.autoRememberCookies = !m.autoRememberCookies
	enabled := m.autoRememberCookies
	return func() tea.Msg {
		return ToggleAutoRememberCookiesMsg{Enabled: enabled}
	}
}

func (m *ShortcutPane) toggleAutoRememberAuthHeaders() tea.Cmd {
	m.autoRememberAuthHeaders = !m.autoRememberAuthHeaders
	enabled := m.autoRememberAuthHeaders
	return func() tea.Msg {
		return ToggleAutoRememberAuthHeadersMsg{Enabled: enabled}
	}
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
		if index == m.settingsCursor {
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
		if index == m.settingsCursor && option.Description != "" {
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

	lines = append(lines, "", m.styles.Text.Logo.Render("COOKIES & AUTH"))
	prefRows := [prefRowCount]struct {
		label   string
		enabled bool
	}{
		prefRowAutoCookies:     {"Auto-remember cookies for a server", m.autoRememberCookies},
		prefRowAutoAuthHeaders: {"Auto-remember auth headers for a server", m.autoRememberAuthHeaders},
	}
	for i, row := range prefRows {
		rowIndex := len(m.themeOptions) + i
		marker := m.styles.Text.Muted.Render("│")
		labelStyle := m.styles.Text.Value
		if rowIndex == m.settingsCursor {
			marker = m.styles.Text.Logo.Render("┃")
			labelStyle = m.styles.Text.Logo
		}
		box := "[ ]"
		if row.enabled {
			box = m.styles.Notice.Success.Render("[x]")
		}
		lines = append(lines, lipgloss.JoinHorizontal(
			lipgloss.Left,
			marker,
			" ",
			box,
			" ",
			labelStyle.Render(row.label),
		))
	}
	lines = append(
		lines,
		m.styles.Text.Muted.Render("  off by default - when off, remember on demand from the response pane (\"c\"/\"v\")"),
	)

	return strings.Join(lines, "\n")
}
