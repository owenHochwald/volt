package shortcutpane

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/design"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func TestHelpTabNavigationKeys(t *testing.T) {
	tests := []struct {
		name  string
		start TabIndex
		key   tea.KeyPressMsg
		want  TabIndex
	}{
		{name: "j advances", start: Global, key: helpKeyPress('j', "j"), want: Sidebar},
		{name: "k moves backward", start: Response, key: helpKeyPress('k', "k"), want: Request},
		{name: "j wraps", start: Response, key: helpKeyPress('j', "j"), want: Global},
		{name: "k wraps", start: Global, key: helpKeyPress('k', "k"), want: Response},
		{name: "one selects global", start: Response, key: helpKeyPress('1', "1"), want: Global},
		{name: "two selects sidebar", start: Global, key: helpKeyPress('2', "2"), want: Sidebar},
		{name: "three selects request", start: Global, key: helpKeyPress('3', "3"), want: Request},
		{name: "four selects response", start: Global, key: helpKeyPress('4', "4"), want: Response},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := SetupShortcutPane(keybindings.DefaultKeyMap())
			pane.activeTab = int(tt.start)

			updated, _ := pane.Update(tt.key)

			if got := TabIndex(updated.activeTab); got != tt.want {
				t.Fatalf("active tab = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHelpAndSettingsUseTopLevelHorizontalNavigation(t *testing.T) {
	pane := SetupShortcutPane(keybindings.DefaultKeyMap())

	pane, _ = pane.Update(helpKeyPress('l', "l"))
	if pane.ActiveSurface() != SurfaceSettings {
		t.Fatalf("l selected surface %d, want settings", pane.ActiveSurface())
	}

	pane, _ = pane.Update(helpKeyPress('h', "h"))
	if pane.ActiveSurface() != SurfaceHelp {
		t.Fatalf("h selected surface %d, want help", pane.ActiveSurface())
	}
}

func TestSettingsPreviewSaveAndCancel(t *testing.T) {
	pane := SetupShortcutPane(keybindings.DefaultKeyMap())
	pane.SetThemeOptions([]ThemeOption{
		{Name: "Default", Source: "default", Theme: design.DefaultTheme()},
		{Name: "Adaptive", Source: "adaptive", Theme: design.AdaptiveTheme(true)},
	}, "default")
	pane, _ = pane.Update(helpKeyPress('l', "l"))

	pane, previewCmd := pane.Update(helpKeyPress('j', "j"))
	preview, ok := previewCmd().(PreviewThemeMsg)
	if !ok || preview.Source != "adaptive" {
		t.Fatalf("preview message = %#v, want adaptive", previewCmd())
	}

	pane, saveCmd := pane.Update(helpKeyPress(tea.KeyEnter, ""))
	saved, ok := saveCmd().(SaveThemeMsg)
	if !ok || saved.Source != "adaptive" {
		t.Fatalf("save message = %#v, want adaptive", saveCmd())
	}

	pane, _ = pane.Update(helpKeyPress('k', "k"))
	_, cancelCmd := pane.Update(helpKeyPress('q', "q"))
	if _, ok := cancelCmd().(CancelThemePreviewMsg); !ok {
		t.Fatalf("cancel message = %#v", cancelCmd())
	}
}

func helpKeyPress(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Text: text})
}
