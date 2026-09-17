package responsepane

import (
	"fmt"
	"strings"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// setupPaneWithLines builds a ResponsePane showing a raw, plain-text body
// of n numbered lines ("line 1".."line n") - a small, predictable viewport
// content for exercising vim/less-style navigation and search against.
func setupPaneWithLines(t *testing.T, n int) *ResponsePane {
	t.Helper()
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetWidth(100)
	pane.SetHeight(20)

	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"text/plain"}},
		Body:       b.String(),
	}, &http.Request{Method: http.GET, URL: "http://x"})
	return &pane
}

func mustResponsePane(t *testing.T, model interface{}) *ResponsePane {
	t.Helper()
	result, ok := model.(*ResponsePane)
	if !ok {
		t.Fatalf("expected *ResponsePane, got %T", model)
	}
	return result
}

// TestResponsePane_GG_ScrollsToTop is a regression test for a real feature
// request: vim/less-style "gg" should scroll straight back to the top of
// whatever's currently displayed, regardless of how far down the user has
// scrolled.
func TestResponsePane_GG_ScrollsToTop(t *testing.T) {
	pane := setupPaneWithLines(t, 200)
	pane.viewport.SetYOffset(100)

	model, _ := pane.Update(keyRunes("g"))
	mid := mustResponsePane(t, model)
	if !mid.gPending {
		t.Fatal("expected the first \"g\" to arm gPending")
	}

	model2, _ := mid.Update(keyRunes("g"))
	result := mustResponsePane(t, model2)
	if result.gPending {
		t.Error("expected the second \"g\" to resolve the sequence, not stay armed")
	}
	if result.viewport.YOffset != 0 {
		t.Errorf("expected \"gg\" to scroll to the top (offset 0), got %d", result.viewport.YOffset)
	}
}

// TestResponsePane_G_ScrollsToBottom verifies "G" (shift+g) jumps straight
// to the bottom in a single keystroke, independent of "gg".
func TestResponsePane_G_ScrollsToBottom(t *testing.T) {
	pane := setupPaneWithLines(t, 200)

	model, _ := pane.Update(keyRunes("G"))
	result := mustResponsePane(t, model)
	if !result.viewport.AtBottom() {
		t.Error("expected \"G\" to scroll to the bottom")
	}
}

// TestResponsePane_GotoLine_JumpsToLineNumber is a regression test for a
// real feature request: typing a line number then "g" (e.g. "12g") should
// jump to that line, vim/less-style.
func TestResponsePane_GotoLine_JumpsToLineNumber(t *testing.T) {
	pane := setupPaneWithLines(t, 200)

	model, _ := pane.Update(keyRunes("g"))
	pane2 := mustResponsePane(t, model)
	model, _ = pane2.Update(keyRunes("1"))
	pane2 = mustResponsePane(t, model)
	model, _ = pane2.Update(keyRunes("2"))
	pane2 = mustResponsePane(t, model)
	model, _ = pane2.Update(keyRunes("0"))
	pane2 = mustResponsePane(t, model)
	model, _ = pane2.Update(keyRunes("g"))
	result := mustResponsePane(t, model)

	if result.gPending {
		t.Error("expected the sequence to be resolved after the closing \"g\"")
	}
	visible := result.viewport.View()
	if !strings.Contains(visible, "line 120") {
		t.Errorf("expected line 120 to be visible after \"120g\", got:\n%s", visible)
	}
}

// TestResponsePane_GotoSequence_CancelledByUnrelatedKey verifies a "g"
// press followed by a key that's neither a digit nor another "g" cancels
// the sequence and falls through to that key's normal handling (here,
// "j" still scrolls down) instead of silently swallowing it.
func TestResponsePane_GotoSequence_CancelledByUnrelatedKey(t *testing.T) {
	pane := setupPaneWithLines(t, 200)

	model, _ := pane.Update(keyRunes("g"))
	mid := mustResponsePane(t, model)

	model2, _ := mid.Update(keyRunes("j"))
	result := mustResponsePane(t, model2)

	if result.gPending {
		t.Error("expected \"j\" to cancel the pending \"g\" sequence")
	}
	if result.viewport.YOffset == 0 {
		t.Error("expected \"j\" to still scroll down after cancelling the sequence")
	}
}
