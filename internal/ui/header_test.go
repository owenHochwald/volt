package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/assert/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
)

func TestHeaderDisplaysInjectedVersion(t *testing.T) {
	tests := []struct {
		name    string
		compact bool
	}{
		{name: "wide"},
		{name: "compact", compact: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := SetupHeader("v9.8.7")
			header.SetSize(120, tt.compact)

			if view := header.View(); !strings.Contains(view, "v9.8.7") {
				t.Fatalf("header does not contain injected version: %q", view)
			}
		})
	}
}

func TestHeaderUsesInjectedSemanticStyles(t *testing.T) {
	theme := design.DefaultTheme()
	theme.Colors.Brand = lipgloss.Color("#010203")
	theme.Colors.TextMuted = lipgloss.Color("#040506")

	header := SetupHeader("dev", design.NewStyles(theme))

	assert.Equal(t, theme.Colors.Brand, header.styles.Header.Logo.GetForeground())
	assert.Equal(t, theme.Colors.TextMuted, header.styles.Header.Metadata.GetForeground())
}

func TestHeaderCompressesIntoTwoRowCommandCenter(t *testing.T) {
	header := SetupHeader("v9.8.7")
	header.SetSize(80, true)
	header.SetContext("request", "load test")

	view := header.View()
	for _, expected := range []string{"⚡ VOLT", "/  REQUEST", "LOAD TEST", "v9.8.7"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("compact header does not contain %q: %q", expected, view)
		}
	}
	assert.Equal(t, 2, lipgloss.Height(view))
	assert.True(t, lipgloss.Width(view) <= 80)
}

func TestFullHeaderKeepsStartupSignature(t *testing.T) {
	header := SetupHeader("dev")
	header.SetSize(120, false)

	assert.Equal(t, 7, lipgloss.Height(header.View()))
	assert.True(t, strings.Contains(header.View(), "██╗"))
}
