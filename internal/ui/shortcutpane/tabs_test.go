package shortcutpane

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/charmbracelet/x/ansi"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestHelpTabLabelsFollowConfiguredBinding(t *testing.T) {
	keys := keybindings.DefaultKeyMap()
	keys.DirectHelpTab = key.NewBinding(key.WithKeys("a", "b", "c", "d"))
	pane := SetupShortcutPane(keys)

	rendered := pane.renderTabs()
	for _, expected := range []string{"[a] Global", "[b] Sidebar", "[c] Request", "[d] Response"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("help tab bar does not contain %q: %s", expected, rendered)
		}
	}
}

func TestHelpDoesNotWrapLongRegistryKeys(t *testing.T) {
	pane := SetupShortcutPane(keybindings.DefaultKeyMap())
	pane.SetWidth(72)
	pane.SetHeight(30)
	pane.SetContext(keybindings.ContextRequest)

	rendered := ansi.Strip(pane.View())
	if !strings.Contains(rendered, "alt+enter") {
		t.Fatalf("long submit binding wrapped unexpectedly:\n%s", rendered)
	}
}

func TestReferencePanelShowsCurrentRegistryBindings(t *testing.T) {
	tests := []struct {
		name     string
		context  keybindings.Context
		expected []string
	}{
		{
			name:     "global panel and quit commands",
			context:  keybindings.ContextGlobal,
			expected: []string{"ctrl+e/alt+h", "ctrl+w/alt+l", "esc esc"},
		},
		{
			name:     "sidebar count navigation",
			context:  keybindings.ContextSidebar,
			expected: []string{"0-9", "prefix movement count", "j", "k"},
		},
		{
			name:     "request submission and activation",
			context:  keybindings.ContextRequest,
			expected: []string{"alt+enter", "enter", "activate focused control"},
		},
		{
			name:     "response load test cancellation",
			context:  keybindings.ContextResponse,
			expected: []string{"ctrl+x", "cancel load test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := SetupShortcutPane(keybindings.DefaultKeyMap())
			pane.SetWidth(100)
			pane.SetHeight(30)
			pane.SetContext(tt.context)
			rendered := ansi.Strip(pane.View())

			for _, expected := range tt.expected {
				if !strings.Contains(rendered, expected) {
					t.Errorf("reference panel does not contain %q:\n%s", expected, rendered)
				}
			}
		})
	}
}

func TestSettingsShowsThemesAndSemanticPreview(t *testing.T) {
	pane := SetupShortcutPane(keybindings.DefaultKeyMap())
	pane.SetWidth(72)
	pane.SetHeight(30)
	pane.SetThemeOptions([]ThemeOption{
		{
			Name:        "Controlled Voltage",
			Description: "Volt default",
			Source:      "default",
			Theme:       design.DefaultTheme(),
		},
		{
			Name:        "Mono",
			Description: "No color",
			Source:      "mono",
			Theme:       design.MonoTheme(),
		},
	}, "default")
	pane.surface = SurfaceSettings

	rendered := ansi.Strip(pane.View())
	for _, expected := range []string{
		"VOLT COMMAND CENTER",
		"HELP",
		"SETTINGS",
		"Controlled Voltage",
		"Mono",
		"ACTIVE",
		"LIVE PREVIEW",
		"SEND",
		"✓ 200",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("settings does not contain %q:\n%s", expected, rendered)
		}
	}
}
