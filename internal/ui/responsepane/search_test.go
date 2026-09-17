package responsepane

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func typeAndConfirmSearch(t *testing.T, pane *ResponsePane, query string) *ResponsePane {
	t.Helper()
	model, _ := pane.Update(keyRunes("/"))
	result := mustResponsePane(t, model)
	if !result.searching {
		t.Fatal("expected \"/\" to open the search prompt")
	}

	model, _ = result.Update(keyRunes(query))
	result = mustResponsePane(t, model)

	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return mustResponsePane(t, model)
}

// TestResponsePane_Search_FindsMatchInRawBody is a regression test for a
// real feature request: "/" should search the response body, jumping to
// the first match.
func TestResponsePane_Search_FindsMatchInRawBody(t *testing.T) {
	pane := setupPaneWithLines(t, 200)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line 150")

	if result.searching {
		t.Error("expected enter to close the search prompt")
	}
	if len(result.searchMatches) != 1 {
		t.Fatalf("expected exactly 1 match for \"line 150\", got %d: %v", len(result.searchMatches), result.searchMatches)
	}
	if !strings.Contains(result.viewport.View(), "line 150") {
		t.Errorf("expected the match to be scrolled into view, got:\n%s", result.viewport.View())
	}
}

// TestResponsePane_Search_FindsMatchInPrettyBody verifies search also
// works against the syntax-highlighted (ANSI-styled) pretty view, not
// just raw - a naive substring search over the ANSI-wrapped text would
// otherwise miss matches whose characters fall under different color
// spans (see searchMatchLines's use of ansi.Strip).
func TestResponsePane_Search_FindsMatchInPrettyBody(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetWidth(100)
	pane.SetHeight(20)
	pane.SetResponse(&http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       `{"albumId": 1, "title": "a distinctive marker phrase"}`,
	}, &http.Request{})

	if pane.rawBody {
		t.Fatal("test setup error: expected the pretty (non-raw) view by default")
	}

	result := typeAndConfirmSearch(t, &pane, "distinctive marker phrase")

	if len(result.searchMatches) != 1 {
		t.Fatalf("expected the pretty/highlighted view to still be searchable, got %d matches", len(result.searchMatches))
	}
}

// TestResponsePane_Search_NoMatches_ReportsStatus verifies a query with no
// matches leaves the pane in a coherent state (no crash, an explanatory
// status) instead of matches from some previous, unrelated search.
func TestResponsePane_Search_NoMatches_ReportsStatus(t *testing.T) {
	pane := setupPaneWithLines(t, 20)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "nonexistent-xyz")

	if len(result.searchMatches) != 0 {
		t.Errorf("expected no matches, got %v", result.searchMatches)
	}
	if result.searchStatusMsg == "" {
		t.Error("expected a \"no matches\" status message")
	}
}

// TestResponsePane_SearchNextPrev_CycleThroughMatches is a regression test
// for the vim/less-style "n"/"b" match navigation this was built for.
func TestResponsePane_SearchNextPrev_CycleThroughMatches(t *testing.T) {
	pane := setupPaneWithLines(t, 30)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	// Every line contains "line ", so all 30 lines match - a simple,
	// predictable sequence to cycle through.
	result := typeAndConfirmSearch(t, pane, "line ")
	if len(result.searchMatches) != 30 {
		t.Fatalf("expected 30 matches, got %d", len(result.searchMatches))
	}
	startIdx := result.searchMatchIdx

	model, _ := result.Update(keyRunes("n"))
	afterNext := mustResponsePane(t, model)
	wantNext := (startIdx + 1) % len(afterNext.searchMatches)
	if afterNext.searchMatchIdx != wantNext {
		t.Errorf("expected \"n\" to advance to match %d, got %d", wantNext, afterNext.searchMatchIdx)
	}

	model, _ = afterNext.Update(keyRunes("b"))
	afterPrev := mustResponsePane(t, model)
	if afterPrev.searchMatchIdx != startIdx {
		t.Errorf("expected \"b\" to move back to match %d, got %d", startIdx, afterPrev.searchMatchIdx)
	}

	// "b" from the very first match must wrap around to the last one.
	model, _ = afterPrev.Update(keyRunes("b"))
	wrapped := mustResponsePane(t, model)
	if wrapped.searchMatchIdx != len(wrapped.searchMatches)-1 {
		t.Errorf("expected \"b\" to wrap to the last match (%d), got %d", len(wrapped.searchMatches)-1, wrapped.searchMatchIdx)
	}
}

// TestResponsePane_Search_ResetsWhenContentChanges verifies switching
// tabs (or toggling raw/pretty) invalidates a previous search - its
// matches are line indices into the content that was displayed when it
// ran, so they'd otherwise silently point at the wrong lines once the
// content underneath changes.
func TestResponsePane_Search_ResetsWhenContentChanges(t *testing.T) {
	pane := setupPaneWithLines(t, 30)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line 5")
	if len(result.searchMatches) == 0 {
		t.Fatal("test setup error: expected at least one match")
	}

	model, _ := result.Update(keyRunes("r")) // toggle back to pretty
	afterToggle := mustResponsePane(t, model)
	if len(afterToggle.searchMatches) != 0 || afterToggle.searchQuery != "" {
		t.Errorf("expected toggling raw/pretty to reset the search, got query=%q matches=%v", afterToggle.searchQuery, afterToggle.searchMatches)
	}
}

// TestResponsePane_Search_PersistsAcrossIdenticalContent verifies
// re-rendering the *same* content (e.g. switching to another tab and
// back) does NOT throw away an active search, only an actual content
// change does (see TestResponsePane_Search_ResetsWhenContentChanges).
func TestResponsePane_Search_PersistsAcrossIdenticalContent(t *testing.T) {
	pane := setupPaneWithLines(t, 30)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line 5")
	if len(result.searchMatches) == 0 {
		t.Fatal("test setup error: expected at least one match")
	}

	// Re-populating the viewport with the exact same content it already
	// has (as switching tabs and back would, once returned to this tab)
	// must not disturb the search.
	result.updateViewportForActiveTab()

	if len(result.searchMatches) == 0 || result.searchQuery != "line 5" {
		t.Errorf("expected the search to survive a re-render of identical content, got query=%q matches=%v", result.searchQuery, result.searchMatches)
	}
}

// TestResponsePane_SearchEsc_ClosesPromptWithoutDiscardingPreviousSearch
// verifies esc cancels only the *in-progress edit* of a new query, the
// same way it cancels the save-to-file prompt without discarding the
// response itself.
func TestResponsePane_SearchEsc_ClosesPromptWithoutDiscardingPreviousSearch(t *testing.T) {
	pane := setupPaneWithLines(t, 30)
	pane.rawBody = true
	pane.updateViewportForActiveTab()

	result := typeAndConfirmSearch(t, pane, "line 5")

	model, _ := result.Update(keyRunes("/"))
	reopened := mustResponsePane(t, model)
	model, _ = reopened.Update(keyRunes("something else"))
	typed := mustResponsePane(t, model)

	model, _ = typed.Update(tea.KeyMsg{Type: tea.KeyEsc})
	afterEsc := mustResponsePane(t, model)

	if afterEsc.searching {
		t.Error("expected esc to close the search prompt")
	}
	if afterEsc.searchQuery != "line 5" {
		t.Errorf("expected esc to leave the previous search intact, got query=%q", afterEsc.searchQuery)
	}
}
