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
