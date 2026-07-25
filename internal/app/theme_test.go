package app

import (
	"testing"

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
	assert.Equal(t, lipgloss.NoColor{}, model.styles.Text.Value.GetForeground())
	assert.Equal(t, lipgloss.NoColor{}, model.requestPane.MethodSelector.GetStyle().GetForeground())
}
