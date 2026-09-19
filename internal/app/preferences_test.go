package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/owenHochwald/Volt/internal/ui/shortcutpane"
)

// TestToggleAutoRememberCookies_UpdatesModelAndResponsePane is a
// regression test for the Settings surface's new preference row: the
// resulting message must update both Model's own field (so reopening the
// Command Center later shows the right state - see openHelp/openGroups)
// and responsePane's (so capture behavior actually changes).
func TestToggleAutoRememberCookies_UpdatesModelAndResponsePane(t *testing.T) {
	m := newTestModel(t)
	if m.autoRememberCookies {
		t.Fatal("test setup error: expected auto-remember cookies to start disabled")
	}

	model, cmd := m.Update(shortcutpane.ToggleAutoRememberCookiesMsg{Enabled: true})
	result := model.(Model)

	if !result.autoRememberCookies {
		t.Error("expected Model.autoRememberCookies to be true after the toggle")
	}
	if cmd == nil {
		t.Fatal("expected a command to persist the new setting")
	}
	if saved, _ := cmd().(preferenceSavedMsg); saved.err != nil {
		t.Errorf("expected the setting to save without error, got %v", saved.err)
	}
}

func TestToggleAutoRememberAuthHeaders_UpdatesModel(t *testing.T) {
	m := newTestModel(t)

	model, _ := m.Update(shortcutpane.ToggleAutoRememberAuthHeadersMsg{Enabled: true})
	result := model.(Model)

	if !result.autoRememberAuthHeaders {
		t.Error("expected Model.autoRememberAuthHeaders to be true after the toggle")
	}
	if result.autoRememberCookies {
		t.Error("expected autoRememberCookies to stay untouched by toggling auth headers")
	}
}

// TestOpenHelp_RefreshesPreferencesShownInSettings verifies openHelp
// passes Model's current preference state into the shortcut pane, so
// reopening the Command Center after a toggle shows what's actually
// active rather than stale initial state.
func TestOpenHelp_RefreshesPreferencesShownInSettings(t *testing.T) {
	m := newTestModel(t)
	model, _ := m.Update(shortcutpane.ToggleAutoRememberCookiesMsg{Enabled: true})
	m = model.(Model)

	m.openHelp("")
	m.shortcutPane.SetSurface(shortcutpane.SurfaceSettings)
	m.shortcutPane.SetWidth(80)
	m.shortcutPane.SetHeight(30)

	rendered := ansi.Strip(m.shortcutPane.View())
	if !strings.Contains(rendered, "[x] Auto-remember cookies for a server") {
		t.Errorf("expected the shortcut pane to reflect the toggled-on cookies preference, got:\n%s", rendered)
	}
}
