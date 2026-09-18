package design

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/assert/v2"
)

func TestThemeConfigIgnoresUnsupportedFields(t *testing.T) {
	config, err := ParseThemeConfig([]byte(`
version: 1
name: Purple Machine
extends: default
colors:
  brand: "#a66cff"
  future_color_role: "#ffffff"
components:
  panel:
    focused_rail: "#c08bff"
future_setting: ignored
`))
	assert.NoError(t, err)

	theme, err := ResolveThemeConfig(config)
	assert.NoError(t, err)
	assert.Equal(t, lipgloss.Color("#A66CFF"), theme.Colors.Brand)
	assert.Equal(t, DefaultTheme().Colors.Charge, theme.Colors.Charge)
}

func TestThemeConfigRejectsInvalidRecognizedColor(t *testing.T) {
	_, err := ParseThemeConfig([]byte(`
version: 1
name: Broken Theme
colors:
  brand: purple
`))

	assert.Error(t, err)
}
