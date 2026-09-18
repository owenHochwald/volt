package responsepane

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func setupPaneWithResponse(t *testing.T, db *storage.SQLiteStorage) *ResponsePane {
	t.Helper()
	pane := SetupResponsePane(db, keybindings.DefaultKeyMap())
	pane.SetWidth(120)
	pane.SetHeight(30)
	// The style picker's list is sized from the request pane's own box
	// dimensions (see SetStylePickerSize), not this pane's own width/
	// height - app/update.go sets this on every resize in the real app;
	// tests need to do it explicitly.
	pane.SetStylePickerSize(60, 20)
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       `{"id": 1, "name": "widget"}`,
	}, &http.Request{Method: http.GET, URL: "http://x"})
	return &pane
}

// TestResponsePane_DefaultBodyStyle_IsOnedark verifies a pane with no
// persisted style (a nil DB, or a DB nothing's ever been saved to)
// starts on defaultBodyStyle ("onedark") - the requested default.
func TestResponsePane_DefaultBodyStyle_IsOnedark(t *testing.T) {
	pane := SetupResponsePane(nil, keybindings.DefaultKeyMap())
	if pane.bodyStyleName != defaultBodyStyle {
		t.Errorf("expected the default body style to be %q, got %q", defaultBodyStyle, pane.bodyStyleName)
	}
}

// TestResponsePane_PickBodyStyle_OpensPicker verifies "t" opens the style
// picker (rendered separately, by app/view.go, in the request pane's own
// box - see StylePickerListView), and that this pane's own View() keeps
// showing the real response content behind it rather than being hidden
// the way the save-to-file prompt is - the whole point of moving the
// picker's list out of this pane's own box in the first place (see the
// package doc on ResponsePane.pickingStyle).
func TestResponsePane_PickBodyStyle_OpensPicker(t *testing.T) {
	pane := setupPaneWithResponse(t, nil)

	model, _ := pane.Update(keyRunes("t"))
	result := model
	if !result.pickingStyle {
		t.Fatal("expected \"t\" to open the style picker")
	}

	list := result.StylePickerListView()
	if !strings.Contains(list, "Body Style") {
		t.Errorf("expected the picker's list title to be visible, got:\n%s", list)
	}

	out := result.View()
	if !strings.Contains(out, "widget") {
		t.Errorf("expected the response content to stay visible while picking, got:\n%s", out)
	}
}

// TestResponsePane_StylePicker_LivePreviewsAsCursorMoves is a regression
// test for the redesigned picker: the response content itself (not a
// separate preview snippet) must re-highlight live as the list cursor
// moves, using whatever style it's currently on (see effectiveBodyStyle)
// - without ever touching bodyStyleName until confirmed.
func TestResponsePane_StylePicker_LivePreviewsAsCursorMoves(t *testing.T) {
	pane := setupPaneWithResponse(t, nil)
	original := pane.bodyStyleName

	model, _ := pane.Update(keyRunes("t"))
	opened := model

	// Pick a style guaranteed to differ from the current default.
	targetIdx := -1
	for i, name := range allChromaStyleNames() {
		if name != original {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		t.Fatal("test setup error: expected at least two chroma styles to exist")
	}
	opened.styleList.Select(targetIdx)
	opened.updateViewportForActiveTab()

	if opened.effectiveBodyStyle() == original {
		t.Fatal("test setup error: expected the selected candidate to differ from the original style")
	}
	if opened.bodyStyleName != original {
		t.Errorf("expected bodyStyleName to stay unchanged while merely browsing, got %q", opened.bodyStyleName)
	}
}

// TestResponsePane_ConfirmStylePicker_AppliesAndPersists is a regression
// test for a real feature request: picking a style from the app must
// both apply it immediately (the Body tab re-renders with it) and
// persist it (see storage.SaveBodyStyle) so the next app start restores
// it (see storage's own round-trip test for the load half of that).
func TestResponsePane_ConfirmStylePicker_AppliesAndPersists(t *testing.T) {
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pane := setupPaneWithResponse(t, db)

	model, _ := pane.Update(keyRunes("t"))
	opened := model

	// Move the cursor onto a specific, known style directly (list.Select
	// is synchronous) rather than through list.Model's own "/" filter -
	// typing into that kicks off the actual fuzzy match asynchronously (a
	// FilterMatchesMsg tea.Cmd, alongside the filter input's cursor-blink
	// tea.Tick, which - unlike everywhere else these tests run a
	// fire-and-forget command - would genuinely block for its real
	// interval if executed), so this sidesteps that entirely rather than
	// draining it.
	draculaIdx := -1
	for i, name := range allChromaStyleNames() {
		if name == "dracula" {
			draculaIdx = i
			break
		}
	}
	if draculaIdx == -1 {
		t.Fatal("test setup error: expected \"dracula\" to be a real chroma style")
	}
	opened.styleList.Select(draculaIdx)

	model, cmd := opened.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	confirmed := model
	if confirmed.pickingStyle {
		t.Error("expected enter to close the picker")
	}
	if confirmed.bodyStyleName != "dracula" {
		t.Errorf("expected bodyStyleName to be \"dracula\", got %q", confirmed.bodyStyleName)
	}
	if cmd == nil {
		t.Fatal("expected a non-nil command to persist the style")
	}
	cmd()

	saved, err := db.LoadBodyStyle()
	if err != nil {
		t.Fatalf("unexpected error loading style: %v", err)
	}
	if saved != "dracula" {
		t.Errorf("expected \"dracula\" persisted, got %q", saved)
	}
}

// TestResponsePane_CancelStylePicker_LeavesStyleUnchanged verifies esc
// closes the picker without applying whatever the list cursor happened
// to be sitting on - browsing is only ever a live preview (see
// effectiveBodyStyle), never a mutation, until enter confirms it.
func TestResponsePane_CancelStylePicker_LeavesStyleUnchanged(t *testing.T) {
	pane := setupPaneWithResponse(t, nil)
	original := pane.bodyStyleName

	model, _ := pane.Update(keyRunes("t"))
	opened := model

	model, _ = opened.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	moved := model

	model, _ = moved.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	result := model
	if result.pickingStyle {
		t.Error("expected esc to close the picker")
	}
	if result.bodyStyleName != original {
		t.Errorf("expected bodyStyleName to stay %q, got %q", original, result.bodyStyleName)
	}
}

// TestResponsePane_StylePicker_QDoesNotQuitTheApp is a regression test
// for the same class of bug fixed earlier in the sidebar: bubbles'
// list.Model ships with "q" (and esc) bound to its own internal Quit,
// which would otherwise exit the whole program the instant "q" is
// pressed while browsing styles (outside of the list's own "/" filter
// input, where "q" is a perfectly ordinary character to type).
func TestResponsePane_StylePicker_QDoesNotQuitTheApp(t *testing.T) {
	pane := setupPaneWithResponse(t, nil)

	model, _ := pane.Update(keyRunes("t"))
	opened := model

	_, cmd := opened.Update(keyRunes("q"))
	if cmd == nil {
		return
	}
	if _, ok := cmd().(tea.QuitMsg); ok {
		t.Fatal("expected \"q\" to NOT quit the app while picking a style")
	}
}
