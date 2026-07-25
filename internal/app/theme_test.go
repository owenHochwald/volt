package app

import (
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/assert/v2"
	"github.com/owenHochwald/Volt/internal/ui"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

func TestSetupModelAppliesLoadedThemeAndWarning(t *testing.T) {
	base := newTestModel(t)
	theme := design.DefaultTheme()
	theme.Name = "Purple Machine"
	theme.Colors.Brand = lipgloss.Color("#A66CFF")

	model := SetupModel(base.db, design.ThemeLoadResult{
		Theme:   theme,
		Source:  "/tmp/purple-machine.yaml",
		Warning: "Theme warning",
	})

	assert.Equal(t, theme, model.theme)
	assert.Equal(t, "/tmp/purple-machine.yaml", model.themeSource)
	assert.Equal(t, theme.Colors.Brand, model.styles.Panel.Focused.GetBorderLeftForeground())
	assert.Equal(t, ui.NotificationWarning, model.notification.Level)
	assert.Equal(t, "Theme warning", model.notification.Text)
}

func TestApplyThemePreservesApplicationState(t *testing.T) {
	model := newTestModel(t)
	model.requestPane.NameInput.SetValue("Keep this request")

	model.applyTheme(design.MonoTheme(), "mono")

	assert.Equal(t, "mono", model.theme.Name)
	assert.Equal(t, "mono", model.themeSource)
	assert.Equal(t, "Keep this request", model.requestPane.NameInput.Value())
	noColor := color.Color(lipgloss.NoColor{})
	assert.Equal(t, noColor, model.styles.Text.Value.GetForeground())
	assert.Equal(t, noColor, model.requestPane.MethodSelector.GetStyle().GetForeground())
}

func TestAdaptiveThemeRespondsToTerminalBackground(t *testing.T) {
	base := newTestModel(t)
	model := SetupModel(base.db, design.ThemeLoadResult{
		Theme:  design.AdaptiveTheme(true),
		Source: "adaptive",
	})

	updated, _ := model.Update(tea.BackgroundColorMsg{Color: color.White})
	model = updated.(Model)

	assert.Equal(t, design.AdaptiveTheme(false), model.theme)
	assert.Equal(t, "adaptive", model.themeSource)
}

func TestCustomThemeIgnoresTerminalBackground(t *testing.T) {
	base := newTestModel(t)
	custom := design.DefaultTheme()
	custom.Name = "adaptive"
	model := SetupModel(base.db, design.ThemeLoadResult{
		Theme:  custom,
		Source: "/tmp/adaptive.yaml",
	})

	updated, _ := model.Update(tea.BackgroundColorMsg{Color: color.White})
	model = updated.(Model)

	assert.Equal(t, custom, model.theme)
	assert.Equal(t, "/tmp/adaptive.yaml", model.themeSource)
}
