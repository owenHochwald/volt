package app

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
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
