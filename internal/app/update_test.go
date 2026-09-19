package app

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
	"github.com/owenHochwald/Volt/internal/utils"
)

func TestQuestionMarkOpensFocusedPanelHelp(t *testing.T) {
	model := newTestModel(t)

	updated, _ := model.Update(appKeyPress('?', "?", 0))
	got := updated.(Model)

	if !got.showHelpModal {
		t.Fatal("question mark did not open help")
	}
	if context := got.shortcutPane.ActiveContext(); context != keybindings.ContextSidebar {
		t.Fatalf("help context = %s, want %s", context, keybindings.ContextSidebar)
	}
}

func TestQuestionMarkAndQRemainEditableInRequestFields(t *testing.T) {
	for _, text := range []string{"?", "q"} {
		t.Run(text, func(t *testing.T) {
			model := newTestModel(t)
			model.setFocusedPanel(utils.RequestPanel)
			model.requestPane.FocusManager.Next()

			updated, _ := model.Update(appKeyPress(rune(text[0]), text, 0))
			got := updated.(Model)

			if got.showHelpModal {
				t.Fatalf("%q opened help while editing", text)
			}
			if value := got.requestPane.URLInput.Value(); value != text {
				t.Fatalf("URL value = %q, want %q", value, text)
			}
		})
	}
}

func TestQNeverQuits(t *testing.T) {
	tests := []struct {
		name  string
		panel utils.Panel
	}{
		{name: "sidebar", panel: utils.SidebarPanel},
		{name: "request controls", panel: utils.RequestPanel},
		{name: "response", panel: utils.ResponsePanel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newTestModel(t)
			model.setFocusedPanel(tt.panel)

			_, cmd := model.Update(appKeyPress('q', "q", 0))

			if commandQuits(cmd) {
				t.Fatal("q quit Volt")
			}
		})
	}
}

func TestDoubleEscapeQuits(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.ResponsePanel)

	updated, firstCmd := model.Update(appKeyPress(tea.KeyEscape, "", 0))
	afterFirst := updated.(Model)

	if commandQuits(firstCmd) {
		t.Fatal("first escape quit Volt")
	}
	if afterFirst.focusedPanel != utils.SidebarPanel {
		t.Fatalf("first escape left panel = %d, want sidebar", afterFirst.focusedPanel)
	}

	_, secondCmd := afterFirst.Update(appKeyPress(tea.KeyEscape, "", 0))
	if !commandQuits(secondCmd) {
		t.Fatal("second consecutive escape did not quit Volt")
	}
}

func TestNonEscapeBreaksQuitSequence(t *testing.T) {
	model := newTestModel(t)

	updated, _ := model.Update(appKeyPress(tea.KeyEscape, "", 0))
	updated, _ = updated.(Model).Update(appKeyPress('q', "q", 0))
	_, cmd := updated.(Model).Update(appKeyPress(tea.KeyEscape, "", 0))

	if commandQuits(cmd) {
		t.Fatal("escape quit after an intervening key")
	}
}

func TestEscapeQuitSequenceExpires(t *testing.T) {
	model := newTestModel(t)

	updated, _ := model.Update(appKeyPress(tea.KeyEscape, "", 0))
	armed := updated.(Model)
	updated, _ = armed.Update(quitSequenceExpiredMsg{sequence: armed.quitSequence})
	_, cmd := updated.(Model).Update(appKeyPress(tea.KeyEscape, "", 0))

	if commandQuits(cmd) {
		t.Fatal("escape quit after the quit sequence expired")
	}
}

func TestFirstEscapeClosesHelpWithoutQuitting(t *testing.T) {
	model := newTestModel(t)
	model.openHelp(keybindings.ContextGlobal)

	updated, cmd := model.Update(appKeyPress(tea.KeyEscape, "", 0))
	got := updated.(Model)

	if commandQuits(cmd) {
		t.Fatal("first escape quit from help")
	}
	if got.showHelpModal {
		t.Fatal("first escape did not close help")
	}
}

func TestControlCAlwaysQuits(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.RequestPanel)
	model.requestPane.FocusManager.Next()

	_, cmd := model.Update(appKeyPress('c', "", tea.ModCtrl))

	if !commandQuits(cmd) {
		t.Fatal("ctrl+c did not quit while editing")
	}
}

func TestF1AlwaysOpensGlobalHelp(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.RequestPanel)
	model.requestPane.FocusManager.Next()

	updated, _ := model.Update(appKeyPress(tea.KeyF1, "", 0))
	got := updated.(Model)

	if !got.showHelpModal {
		t.Fatal("F1 did not open help while editing")
	}
	if context := got.shortcutPane.ActiveContext(); context != keybindings.ContextGlobal {
		t.Fatalf("help context = %s, want %s", context, keybindings.ContextGlobal)
	}
}

func TestPanelNavigationBlursRequestControl(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.RequestPanel)
	model.requestPane.FocusManager.Next()
	if !model.requestPane.URLInput.Focused() {
		t.Fatal("URL input is not focused before panel navigation")
	}

	updated, _ := model.Update(appKeyPress('l', "", tea.ModAlt))
	got := updated.(Model)

	if got.focusedPanel != utils.ResponsePanel {
		t.Fatalf("focused panel = %d, want response", got.focusedPanel)
	}
	if got.requestPane.URLInput.Focused() {
		t.Fatal("URL input remained focused after leaving request panel")
	}
}

func escKeyPress() tea.KeyPressMsg {
	return appKeyPress(tea.KeyEsc, "", 0)
}

// TestEsc_InResponsePanel_ClosesSaveToFilePromptFirst verifies esc's more
// specific existing meaning - canceling the response pane's own
// save-to-file filename prompt (see SaveResponseRaw) - takes priority over
// the panel-switch binding; otherwise that prompt could never actually be
// cancelled with esc, since the panel switch would always intercept it
// first.
func TestEsc_InResponsePanel_ClosesSaveToFilePromptFirst(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.ResponsePanel)
	model.responsePane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK", Body: "hello"}, &http.Request{Method: http.GET, URL: "http://x"})
	model.responsePane.Update(appKeyPress('s', "s", 0))
	if !model.responsePane.IsSavingToFile() {
		t.Fatal("test setup error: expected 's' to open the save-to-file prompt")
	}

	updated, _ := model.Update(escKeyPress())
	got := updated.(Model)
	if got.focusedPanel != utils.ResponsePanel {
		t.Errorf("expected esc to stay in the response panel while its save prompt is open, got %v", got.focusedPanel)
	}
}

// TestEsc_InResponsePanel_ClosesSearchPromptFirst is the same guard as
// TestEsc_InResponsePanel_ClosesSaveToFilePromptFirst above, for the
// response pane's other filename-prompt-shaped overlay: its "/" search
// prompt (see SearchResponse).
func TestEsc_InResponsePanel_ClosesSearchPromptFirst(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.ResponsePanel)
	model.responsePane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK", Body: "hello"}, &http.Request{Method: http.GET, URL: "http://x"})
	model.responsePane.Update(appKeyPress('/', "/", 0))
	if !model.responsePane.IsSearching() {
		t.Fatal("test setup error: expected '/' to open the search prompt")
	}

	updated, _ := model.Update(escKeyPress())
	got := updated.(Model)
	if got.focusedPanel != utils.ResponsePanel {
		t.Errorf("expected esc to stay in the response panel while its search prompt is open, got %v", got.focusedPanel)
	}
}

// TestEsc_InResponsePanel_CancelsPendingGotoSequenceFirst is the same
// guard as TestEsc_InResponsePanel_ClosesSaveToFilePromptFirst above, for
// a pending "g{n}g" sequence (see ResponsePane.IsGotoPending) - esc must
// reach the response pane's own Update (where continueGoto's default case
// actually clears gPending) instead of the panel-switch/quit-arm
// swallowing it first and leaving the sequence stuck armed.
func TestEsc_InResponsePanel_CancelsPendingGotoSequenceFirst(t *testing.T) {
	model := newTestModel(t)
	model.setFocusedPanel(utils.ResponsePanel)
	model.responsePane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK", Body: "hello"}, &http.Request{Method: http.GET, URL: "http://x"})
	model.responsePane.Update(appKeyPress('g', "g", 0))
	if !model.responsePane.IsGotoPending() {
		t.Fatal("test setup error: expected 'g' to arm the goto sequence")
	}

	updated, _ := model.Update(escKeyPress())
	got := updated.(Model)
	if got.focusedPanel != utils.ResponsePanel {
		t.Errorf("expected esc to stay in the response panel while a goto sequence is pending, got %v", got.focusedPanel)
	}
	if got.responsePane.IsGotoPending() {
		t.Error("expected esc to cancel the pending goto sequence")
	}
}

func TestPanelNavigationKeySequences(t *testing.T) {
	tests := []struct {
		name  string
		start utils.Panel
		keys  []tea.KeyPressMsg
		want  utils.Panel
	}{
		{
			name:  "control w moves to next panel",
			start: utils.SidebarPanel,
			keys:  []tea.KeyPressMsg{appKeyPress('w', "", tea.ModCtrl)},
			want:  utils.RequestPanel,
		},
		{
			name:  "control e moves to previous panel and wraps",
			start: utils.SidebarPanel,
			keys:  []tea.KeyPressMsg{appKeyPress('e', "", tea.ModCtrl)},
			want:  utils.ResponsePanel,
		},
		{
			name:  "mac option next panel with associated text",
			start: utils.SidebarPanel,
			keys: []tea.KeyPressMsg{tea.KeyPressMsg(tea.Key{
				Code:     'l',
				BaseCode: 'l',
				Text:     "¬",
				Mod:      tea.ModAlt,
			})},
			want: utils.RequestPanel,
		},
		{
			name:  "mac option previous panel with associated text",
			start: utils.SidebarPanel,
			keys: []tea.KeyPressMsg{tea.KeyPressMsg(tea.Key{
				Code:     'h',
				BaseCode: 'h',
				Text:     "˙",
				Mod:      tea.ModAlt,
			})},
			want: utils.ResponsePanel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := newTestModel(t)
			model.setFocusedPanel(tt.start)

			for _, key := range tt.keys {
				updated, _ := model.Update(key)
				model = updated.(Model)
			}

			if model.focusedPanel != tt.want {
				t.Fatalf("focused panel = %d, want %d", model.focusedPanel, tt.want)
			}
		})
	}
}

func newTestModel(t *testing.T) Model {
	t.Helper()

	// TestMain sandboxes XDG_CONFIG_HOME/HOME for the whole test binary
	// (see main_test.go), but that one shared directory is only safe for
	// reads - a test that actually persists a preference (see
	// preferences_test.go's saveAutoRememberCookiesCmd) would otherwise
	// leak it into every other test in this package that also goes
	// through SetupModel. A fresh, per-test directory keeps that write
	// contained to the test that made it.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	db, err := storage.NewSQLiteStorage(filepath.Join(t.TempDir(), "volt.db"))
	if err != nil {
		t.Fatalf("create test storage: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close test storage: %v", err)
		}
	})
	return SetupModel(db)
}

func appKeyPress(code rune, text string, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{
		Code: code,
		Text: text,
		Mod:  mod,
	})
}

func commandQuits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case tea.QuitMsg:
		return true
	case tea.BatchMsg:
		for _, batched := range msg {
			if commandQuits(batched) {
				return true
			}
		}
	}
	return false
}
