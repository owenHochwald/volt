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
	if result.viewport.YOffset() != 0 {
		t.Errorf("expected \"gg\" to scroll to the top (offset 0), got %d", result.viewport.YOffset())
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
// real feature request: "g", then a line number, then "g" again (e.g.
// "g12g") should jump to that line, vim/less-style. The leading "g"
// matters - without it, digits are ActionDirectTab's own tab-jump
// instead (see ActionGotoLine's own key hint, "g{n}g", not "{n}g").
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
	if result.viewport.YOffset() == 0 {
		t.Error("expected \"j\" to still scroll down after cancelling the sequence")
	}
}

// TestResponsePane_GotoSequence_ExpiresAfterInactivity is a regression
// test for a real feature request: a "g" pressed and then forgotten about
// (no follow-up key at all, not even an unrelated one - see
// TestResponsePane_GotoSequence_CancelledByUnrelatedKey for that case)
// must eventually let go of digit keys on its own, or they'd stay
// captured as a goto-line buffer indefinitely instead of switching tabs
// (see ActionDirectTab) the next time the user reaches for one.
func TestResponsePane_GotoSequence_ExpiresAfterInactivity(t *testing.T) {
	pane := setupPaneWithLines(t, 200)

	model, cmd := pane.Update(keyRunes("g"))
	armed := mustResponsePane(t, model)
	if !armed.gPending {
		t.Fatal("expected \"g\" to arm gPending")
	}
	if cmd == nil {
		t.Fatal("expected \"g\" to schedule an expiry timeout")
	}

	// Deliver the expiry directly with the sequence armGotoTimeout
	// actually armed, rather than calling cmd() - a real tea.Tick command
	// blocks for the full gotoSequenceTimeout before returning its
	// message, which would make this test take 3 real seconds for no
	// reason.
	model, _ = armed.Update(gotoSequenceExpiredMsg{sequence: armed.gotoSequence})
	result := mustResponsePane(t, model)
	if result.gPending {
		t.Error("expected the sequence to expire after inactivity")
	}
}

// TestResponsePane_GotoSequence_StaleExpiryIgnoredAfterResolving verifies
// a *stale* expiry (scheduled by an earlier keystroke in a sequence
// that's since resolved, e.g. "gg", or been re-armed by a later digit)
// doesn't cancel a sequence it no longer applies to - otherwise typing
// "g" then quickly "1" then waiting could have the *first* keystroke's
// timer fire and cancel what the "1" already re-armed.
func TestResponsePane_GotoSequence_StaleExpiryIgnoredAfterResolving(t *testing.T) {
	pane := setupPaneWithLines(t, 200)

	model, cmd := pane.Update(keyRunes("g"))
	armed := mustResponsePane(t, model)
	if cmd == nil {
		t.Fatal("expected \"g\" to schedule an expiry timeout")
	}
	// The stale message this first "g" would eventually deliver, built
	// directly rather than by calling cmd() - see
	// TestResponsePane_GotoSequence_ExpiresAfterInactivity for why.
	staleMsg := gotoSequenceExpiredMsg{sequence: armed.gotoSequence}

	model, _ = armed.Update(keyRunes("g")) // resolves "gg" - gPending clears
	resolved := mustResponsePane(t, model)
	if resolved.gPending {
		t.Fatal("test setup error: expected \"gg\" to resolve the sequence")
	}

	// A new "g" press re-arms gPending (a new, independent sequence) -
	// the stale expiry from the *first* "g" must not cancel it.
	model, _ = resolved.Update(keyRunes("g"))
	rearmed := mustResponsePane(t, model)
	if !rearmed.gPending {
		t.Fatal("test setup error: expected the second \"g\" to re-arm gPending")
	}

	model, _ = rearmed.Update(staleMsg)
	result := mustResponsePane(t, model)
	if !result.gPending {
		t.Error("expected a stale expiry to leave the current sequence armed")
	}
}
