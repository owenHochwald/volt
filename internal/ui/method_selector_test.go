package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/assert/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

func TestMethodSelectorUsesInjectedSemanticStyles(t *testing.T) {
	theme := design.DefaultTheme()
	theme.Colors.MethodGET = lipgloss.Color("#010203")
	theme.Colors.Brand = lipgloss.Color("#040506")
	styles := design.NewStyles(theme)
	selector := NewMethodSelector(styles)

	assert.Equal(t, theme.Colors.MethodGET, selector.GetStyle().GetForeground())
	assert.Equal(t, theme.Colors.Border, selector.GetStyle().GetBorderLeftForeground())

	selector.Focus()
	assert.Equal(t, theme.Colors.Brand, selector.GetStyle().GetBorderLeftForeground())
}
