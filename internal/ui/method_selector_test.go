package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/assert/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

// TestMethodSelectorUsesInjectedSemanticStyles verifies GetStyle() picks
// up the injected theme's per-method color. Deliberately no border color
// assertions here - unlike design.NewStyles' own methodBase, this
// selector renders with no border at all (see GetStyle's own comment:
// that saved 2 vertical lines in a pane that's often tight on height),
// showing focus via Reverse instead of a border color change.
func TestMethodSelectorUsesInjectedSemanticStyles(t *testing.T) {
	theme := design.DefaultTheme()
	theme.Colors.MethodGET = lipgloss.Color("#010203")
	styles := design.NewStyles(theme)
	selector := NewMethodSelector(styles)

	assert.Equal(t, theme.Colors.MethodGET, selector.GetStyle().GetForeground())
	assert.Equal(t, false, selector.GetStyle().GetReverse())

	selector.Focus()
	assert.Equal(t, true, selector.GetStyle().GetReverse())
}
