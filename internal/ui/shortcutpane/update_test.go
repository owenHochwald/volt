package shortcutpane

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
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
		// Wraps past index 5 (the Import help group, the last of our 6
		// contextual groups - see keybindings.KeyMap.GetKeyGroups), not
		// Response - main only has the first 4 groups (Global/Sidebar/
		// Request/Response); ours adds Groups and Import.
		{name: "j wraps", start: TabIndex(5), key: helpKeyPress('j', "j"), want: Global},
		{name: "k wraps", start: Global, key: helpKeyPress('k', "k"), want: TabIndex(5)},
		{name: "one selects global", start: Response, key: helpKeyPress('1', "1"), want: Global},
		{name: "two selects sidebar", start: Global, key: helpKeyPress('2', "2"), want: Sidebar},
		{name: "three selects request", start: Global, key: helpKeyPress('3', "3"), want: Request},
		{name: "four selects response", start: Global, key: helpKeyPress('4', "4"), want: Response},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := SetupShortcutPane(keybindings.DefaultKeyMap(), nil)
			pane.activeTab = int(tt.start)

			updated, _ := pane.Update(tt.key)

			if got := TabIndex(updated.activeTab); got != tt.want {
				t.Fatalf("active tab = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestHelpAndSettingsUseTopLevelHorizontalNavigation(t *testing.T) {
	pane := SetupShortcutPane(keybindings.DefaultKeyMap(), nil)

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
	pane := SetupShortcutPane(keybindings.DefaultKeyMap(), nil)
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

// settingsPaneWithTwoThemes is the Settings surface, cursor at row 0
// (the first theme) - shared setup for the preference-row tests below,
// which all need to move the cursor past a known number of theme rows.
func settingsPaneWithTwoThemes(t *testing.T) ShortcutPane {
	t.Helper()
	pane := SetupShortcutPane(keybindings.DefaultKeyMap(), nil)
	pane.SetThemeOptions([]ThemeOption{
		{Name: "Default", Source: "default", Theme: design.DefaultTheme()},
		{Name: "Adaptive", Source: "adaptive", Theme: design.AdaptiveTheme(true)},
	}, "default")
	pane.surface = SurfaceSettings
	return pane
}

// TestSettingsCursor_MovesPastThemesOntoPreferenceRows verifies NavDown
// carries the cursor from the theme list onto the two preference toggles
// below it, rather than only ever cycling the themes - a regression test
// for moveSettingsCursor unifying what used to be moveTheme's own,
// theme-only wrap bound.
func TestSettingsCursor_MovesPastThemesOntoPreferenceRows(t *testing.T) {
	pane := settingsPaneWithTwoThemes(t)

	// 2 themes at indices 0-1, so index 2 is the first preference row.
	for range 2 {
		pane, _ = pane.Update(helpKeyPress('j', "j"))
	}
	if pane.settingsCursor != 2 {
		t.Fatalf("settingsCursor = %d, want 2 (first preference row)", pane.settingsCursor)
	}

	pane, _ = pane.Update(helpKeyPress('j', "j"))
	if pane.settingsCursor != 3 {
		t.Fatalf("settingsCursor = %d, want 3 (second preference row)", pane.settingsCursor)
	}

	// Wraps back to the first theme row.
	pane, cmd := pane.Update(helpKeyPress('j', "j"))
	if pane.settingsCursor != 0 {
		t.Fatalf("settingsCursor = %d, want 0 (wrapped back to the first theme)", pane.settingsCursor)
	}
	if cmd == nil {
		t.Fatal("expected wrapping back onto a theme row to preview it")
	}
}

// TestSettingsCursor_OnPreferenceRow_DoesNotPreviewATheme verifies moving
// the cursor onto a preference row returns no PreviewThemeMsg - there's
// nothing to preview there (see moveSettingsCursor).
func TestSettingsCursor_OnPreferenceRow_DoesNotPreviewATheme(t *testing.T) {
	pane := settingsPaneWithTwoThemes(t)

	var cmd tea.Cmd
	for range 2 {
		pane, cmd = pane.Update(helpKeyPress('j', "j"))
	}
	if cmd != nil {
		t.Errorf("expected no preview command when landing on a preference row, got %#v", cmd())
	}
}

// TestActivateSettingsRow_TogglesAutoRememberCookies is a regression test
// for the "combine automatic and manual" cookie-capture setting: enter on
// that row must flip the in-pane display state and emit
// ToggleAutoRememberCookiesMsg for app/update.go to persist.
func TestActivateSettingsRow_TogglesAutoRememberCookies(t *testing.T) {
	pane := settingsPaneWithTwoThemes(t)
	for range 2 {
		pane, _ = pane.Update(helpKeyPress('j', "j"))
	}

	pane, cmd := pane.Update(helpKeyPress(tea.KeyEnter, ""))
	msg, ok := cmd().(ToggleAutoRememberCookiesMsg)
	if !ok || !msg.Enabled {
		t.Fatalf("expected ToggleAutoRememberCookiesMsg{Enabled: true}, got %#v", cmd())
	}
	if !pane.autoRememberCookies {
		t.Error("expected autoRememberCookies to be true after toggling on")
	}

	_, cmd = pane.Update(helpKeyPress(tea.KeyEnter, ""))
	msg, ok = cmd().(ToggleAutoRememberCookiesMsg)
	if !ok || msg.Enabled {
		t.Fatalf("expected ToggleAutoRememberCookiesMsg{Enabled: false} on a second toggle, got %#v", cmd())
	}
}

// TestActivateSettingsRow_TogglesAutoRememberAuthHeaders_Independently
// verifies the two preference rows toggle independently of each other.
func TestActivateSettingsRow_TogglesAutoRememberAuthHeaders_Independently(t *testing.T) {
	pane := settingsPaneWithTwoThemes(t)
	for range 3 {
		pane, _ = pane.Update(helpKeyPress('j', "j"))
	}

	pane, cmd := pane.Update(helpKeyPress(tea.KeyEnter, ""))
	msg, ok := cmd().(ToggleAutoRememberAuthHeadersMsg)
	if !ok || !msg.Enabled {
		t.Fatalf("expected ToggleAutoRememberAuthHeadersMsg{Enabled: true}, got %#v", cmd())
	}
	if pane.autoRememberCookies {
		t.Error("expected autoRememberCookies to stay untouched by toggling auth headers")
	}
}

// TestSetPreferences_ReflectsInitialState verifies SetPreferences (called
// once at startup with the persisted config, and again every time the
// Command Center opens) actually changes what the toggle rows show.
func TestSetPreferences_ReflectsInitialState(t *testing.T) {
	pane := settingsPaneWithTwoThemes(t)
	pane.SetPreferences(true, false)

	rendered := ansi.Strip(pane.renderSettings())
	if !strings.Contains(rendered, "[x] Auto-remember cookies for a server") {
		t.Errorf("expected cookies row to show as enabled, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "[ ] Auto-remember auth headers for a server") {
		t.Errorf("expected auth headers row to show as disabled, got:\n%s", rendered)
	}
}
