package responsepane

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/history"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

// TestResponsePane_HistoryViewer_XStartsDiff verifies "x" in the history
// viewer diffs the selected (older) entry against whatever's currently
// displayed, and closes the viewer.
func TestResponsePane_HistoryViewer_XStartsDiff(t *testing.T) {
	pane, _ := setupPaneWithHistory(t)
	pane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK", Body: "one"}, &http.Request{Method: http.GET, URL: "https://a"})
	pane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK", Body: "two"}, &http.Request{Method: http.GET, URL: "https://b"})

	model, _ := pane.Update(keyRunes("H"))
	opened := model
	// Move down to select the older entry ("a", now the second/
	// newest-first item), same as TestResponsePane_ViewHistory_EnterViewsSelectedEntry.
	opened, _ = opened.Update(keyRunes("j"))

	model, _ = opened.Update(keyRunes("x"))
	result := model

	if result.IsViewingHistory() {
		t.Error("expected \"x\" to close the history viewer")
	}
	if !result.IsDiffing() {
		t.Fatal("expected \"x\" to start diff mode")
	}
	// The currently displayed response ("two") is unchanged by diffing.
	if result.Response.Body != "two" {
		t.Errorf("expected the currently displayed response to stay \"two\", got %q", result.Response.Body)
	}
}

// TestResponsePane_Diff_ContentShowsChangedLines verifies the diff content
// marks a changed body line as removed/added.
func TestResponsePane_Diff_ContentShowsChangedLines(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetWidth(120)
	pane.SetHeight(30)
	pane.SetResponse(&http.Response{StatusCode: 200, Status: "200 OK", Body: "after"}, &http.Request{Method: http.GET, URL: "https://a"})

	older := history.Entry{
		Response:    &http.Response{StatusCode: 200, Status: "200 OK", Body: "before"},
		SentRequest: &http.Request{Method: http.GET, URL: "https://a"},
		SentAt:      time.Now(),
	}
	pane.startDiff(older)

	if !pane.IsDiffing() {
		t.Fatal("expected startDiff to enter diff mode")
	}
	if !strings.Contains(pane.diffContent, "before") || !strings.Contains(pane.diffContent, "after") {
		t.Errorf("expected diff content to mention both bodies, got:\n%s", pane.diffContent)
	}
}

// TestResponsePane_Diff_NoResponse_DoesNothing verifies startDiff is a
// no-op when there's nothing currently displayed to diff against.
func TestResponsePane_Diff_NoResponse_DoesNothing(t *testing.T) {
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	older := history.Entry{
		Response:    &http.Response{StatusCode: 200},
		SentRequest: &http.Request{Method: http.GET, URL: "https://a"},
		SentAt:      time.Now(),
	}

	pane.startDiff(older)

	if pane.IsDiffing() {
		t.Error("expected startDiff to no-op when there's no current response")
	}
}

// TestResponsePane_Diff_EscExits verifies esc exits diff mode back to the
// normal tab content.
func TestResponsePane_Diff_EscExits(t *testing.T) {
	pane, _ := setupPaneWithHistory(t)
	pane.SetResponse(&http.Response{StatusCode: 200, Body: "one"}, &http.Request{Method: http.GET, URL: "https://a"})
	pane.SetResponse(&http.Response{StatusCode: 200, Body: "two"}, &http.Request{Method: http.GET, URL: "https://b"})

	model, _ := pane.Update(keyRunes("H"))
	opened, _ := model.Update(keyRunes("j"))
	diffing, _ := opened.Update(keyRunes("x"))
	if !diffing.IsDiffing() {
		t.Fatal("expected diff mode to have started")
	}

	model, _ = diffing.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	result := model

	if result.IsDiffing() {
		t.Error("expected esc to exit diff mode")
	}
}

// TestResponsePane_Diff_TabSwitchExits verifies switching tabs while
// diffing exits diff mode and lets the tab switch proceed normally.
func TestResponsePane_Diff_TabSwitchExits(t *testing.T) {
	pane, _ := setupPaneWithHistory(t)
	pane.SetResponse(&http.Response{StatusCode: 200, Body: "one"}, &http.Request{Method: http.GET, URL: "https://a"})
	pane.SetResponse(&http.Response{StatusCode: 200, Body: "two"}, &http.Request{Method: http.GET, URL: "https://b"})

	model, _ := pane.Update(keyRunes("H"))
	opened, _ := model.Update(keyRunes("j"))
	diffing, _ := opened.Update(keyRunes("x"))
	if !diffing.IsDiffing() {
		t.Fatal("expected diff mode to have started")
	}

	model, _ = diffing.Update(keyRunes("2"))
	result := model

	if result.IsDiffing() {
		t.Error("expected switching tabs to exit diff mode")
	}
	if TabIndex(result.activeTab) != TabHeaders {
		t.Errorf("expected the tab switch itself to still take effect, got active tab %d", result.activeTab)
	}
}
