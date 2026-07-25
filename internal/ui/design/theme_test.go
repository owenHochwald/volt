package design

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/assert/v2"
)

func TestDefaultThemeUsesControlledVoltagePalette(t *testing.T) {
	theme := DefaultTheme()

	assert.Equal(t, "default", theme.Name)
	assert.Equal(t, MotionSystem, theme.Motion)

	expected := map[string]struct {
		got  any
		want any
	}{
		"canvas":         {theme.Colors.Canvas, lipgloss.Color("#090B10")},
		"surface":        {theme.Colors.Surface, lipgloss.Color("#11151D")},
		"surface raised": {theme.Colors.SurfaceRaised, lipgloss.Color("#181E29")},
		"border":         {theme.Colors.Border, lipgloss.Color("#30394A")},
		"text":           {theme.Colors.Text, lipgloss.Color("#EDF2FF")},
		"text muted":     {theme.Colors.TextMuted, lipgloss.Color("#7F8A9D")},
		"brand":          {theme.Colors.Brand, lipgloss.Color("#9B6CFF")},
		"brand strong":   {theme.Colors.BrandStrong, lipgloss.Color("#7038E8")},
		"charge":         {theme.Colors.Charge, lipgloss.Color("#D8FF3E")},
		"signal":         {theme.Colors.Signal, lipgloss.Color("#3DE4E8")},
		"info":           {theme.Colors.Info, lipgloss.Color("#68B7FF")},
		"success":        {theme.Colors.Success, lipgloss.Color("#5EE08A")},
		"warning":        {theme.Colors.Warning, lipgloss.Color("#FFC857")},
		"error":          {theme.Colors.Error, lipgloss.Color("#FF647C")},
		"GET":            {theme.Colors.MethodGET, lipgloss.Color("#5EE08A")},
		"POST":           {theme.Colors.MethodPOST, lipgloss.Color("#FFC857")},
		"PUT":            {theme.Colors.MethodPUT, lipgloss.Color("#68B7FF")},
		"PATCH":          {theme.Colors.MethodPATCH, lipgloss.Color("#B78CFF")},
		"DELETE":         {theme.Colors.MethodDELETE, lipgloss.Color("#FF647C")},
		"chart primary":  {theme.Colors.ChartPrimary, lipgloss.Color("#9B6CFF")},
		"chart secondary": {
			theme.Colors.ChartSecondary,
			lipgloss.Color("#3DE4E8"),
		},
	}

	for name, value := range expected {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, value.want, value.got)
		})
	}
}

func TestStylesUseSemanticThemeRoles(t *testing.T) {
	theme := DefaultTheme()
	styles := NewStyles(theme)

	assert.Equal(t, theme.Colors.BrandStrong, styles.Tabs.Active.GetBackground())
	assert.Equal(t, theme.Colors.Text, styles.Tabs.Active.GetForeground())
	assert.Equal(t, theme.Colors.TextMuted, styles.Tabs.Inactive.GetForeground())

	assert.Equal(t, theme.Colors.Border, styles.Panel.Base.GetBorderLeftForeground())
	assert.Equal(t, theme.Colors.Brand, styles.Panel.Focused.GetBorderLeftForeground())
	assert.Equal(t, theme.Colors.Charge, styles.Panel.Running.GetBorderLeftForeground())

	assert.Equal(t, theme.Colors.Charge, styles.Action.Primary.GetForeground())
	assert.Equal(t, theme.Colors.Charge, styles.Action.Focused.GetBackground())
	assert.Equal(t, theme.Colors.Canvas, styles.Action.Focused.GetForeground())

	assert.Equal(t, theme.Colors.TextMuted, styles.Text.Label.GetForeground())
	assert.Equal(t, theme.Colors.Brand, styles.Text.Logo.GetForeground())
	assert.Equal(t, theme.Colors.Error, styles.Badge.Error.GetBackground())
	assert.Equal(t, theme.Colors.MethodGET, styles.Method.GET.GetForeground())
	assert.Equal(t, theme.Colors.MethodDELETE, styles.Method.DELETE.GetForeground())
}

func TestStylesDoNotChangePanelGeometryAcrossStates(t *testing.T) {
	styles := NewStyles(DefaultTheme())

	baseWidth := styles.Panel.Base.GetHorizontalFrameSize()
	baseHeight := styles.Panel.Base.GetVerticalFrameSize()

	assert.Equal(t, baseWidth, styles.Panel.Focused.GetHorizontalFrameSize())
	assert.Equal(t, baseHeight, styles.Panel.Focused.GetVerticalFrameSize())
	assert.Equal(t, baseWidth, styles.Panel.Running.GetHorizontalFrameSize())
	assert.Equal(t, baseHeight, styles.Panel.Running.GetVerticalFrameSize())
}
