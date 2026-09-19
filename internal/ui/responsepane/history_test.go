package responsepane

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/history"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func setupPaneWithHistory(t *testing.T) (*ResponsePane, *history.Store) {
	t.Helper()
	h := history.NewStore()
	pane := SetupResponsePane(keybindings.DefaultKeyMap())
	pane.SetWidth(120)
	pane.SetHeight(30)
	pane.SetHistoryViewerSize(60, 20)
	pane.SetHistory(h)
	return &pane, h
}

// TestResponsePane_SetResponse_RecordsIntoHistory is a regression test
// for Phase 3 of "history and diff": every non-load-test response must
// be recorded automatically, with no setting to opt in (unlike the
// cookie jar's auto-remember toggles - there's no sensitivity concern
// here the user asked to gate).
func TestResponsePane_SetResponse_RecordsIntoHistory(t *testing.T) {
	pane, h := setupPaneWithHistory(t)

	pane.SetResponse(
		&http.Response{StatusCode: 200, Status: "200 OK"},
		&http.Request{Method: http.GET, URL: "https://api.example.com/x"},
	)

	entries := h.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 recorded entry, got %d", len(entries))
	}
	if entries[0].SentRequest.URL != "https://api.example.com/x" {
		t.Errorf("expected the sent request recorded, got %+v", entries[0].SentRequest)
	}
}

func TestResponsePane_SetResponse_MultipleSendsAccumulate(t *testing.T) {
	pane, h := setupPaneWithHistory(t)

	pane.SetResponse(&http.Response{StatusCode: 200}, &http.Request{Method: http.GET, URL: "https://a"})
	pane.SetResponse(&http.Response{StatusCode: 404}, &http.Request{Method: http.GET, URL: "https://b"})

	entries := h.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 recorded entries, got %d", len(entries))
	}
	if entries[0].SentRequest.URL != "https://b" {
		t.Errorf("expected the most recent send first, got %+v", entries[0].SentRequest)
	}
}

func TestResponsePane_ViewHistory_OpensAndShowsPastSends(t *testing.T) {
	pane, _ := setupPaneWithHistory(t)
	pane.SetResponse(&http.Response{StatusCode: 200}, &http.Request{Method: http.GET, URL: "https://api.example.com/widgets"})

	model, _ := pane.Update(keyRunes("H"))
	result := model
	if !result.IsViewingHistory() {
		t.Fatal("expected \"H\" to open the history viewer")
	}

	view := result.HistoryViewerListView()
	if !strings.Contains(view, "widgets") {
		t.Errorf("expected the past send to be listed, got:\n%s", view)
	}
}

// TestResponsePane_ViewHistory_EnterViewsSelectedEntry verifies selecting
// a past entry makes it the currently displayed response, without
// re-recording it as a new entry.
func TestResponsePane_ViewHistory_EnterViewsSelectedEntry(t *testing.T) {
	pane, h := setupPaneWithHistory(t)
	pane.SetResponse(&http.Response{StatusCode: 200, Body: "first"}, &http.Request{Method: http.GET, URL: "https://a"})
	pane.SetResponse(&http.Response{StatusCode: 200, Body: "second"}, &http.Request{Method: http.GET, URL: "https://b"})

	// Currently displaying the most recent send ("b") - move the cursor
	// down to select the older one ("a", now the second/newest-first item).
	model, _ := pane.Update(keyRunes("H"))
	opened := model
	opened, _ = opened.Update(keyRunes("j"))

	model, _ = opened.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	result := model

	if result.IsViewingHistory() {
		t.Error("expected enter to close the viewer")
	}
	if result.Response.Body != "first" {
		t.Errorf("expected the older entry (\"a\") now displayed, got body %q", result.Response.Body)
	}
	if len(h.Entries()) != 2 {
		t.Errorf("expected viewing an entry not to add a new one, got %d entries", len(h.Entries()))
	}
}

func TestResponsePane_ViewHistory_DeleteRemovesSelectedEntry(t *testing.T) {
	pane, h := setupPaneWithHistory(t)
	pane.SetResponse(&http.Response{StatusCode: 200}, &http.Request{Method: http.GET, URL: "https://api.example.com/widgets"})

	model, _ := pane.Update(keyRunes("H"))
	opened := model

	_, _ = opened.Update(keyRunes("d"))

	if len(h.Entries()) != 0 {
		t.Errorf("expected the entry to be deleted, got %+v", h.Entries())
	}
}

func TestResponsePane_ViewHistory_ClearAllRemovesEverything(t *testing.T) {
	pane, h := setupPaneWithHistory(t)
	pane.SetResponse(&http.Response{StatusCode: 200}, &http.Request{Method: http.GET, URL: "https://a"})
	pane.SetResponse(&http.Response{StatusCode: 200}, &http.Request{Method: http.GET, URL: "https://b"})

	model, _ := pane.Update(keyRunes("H"))
	opened := model

	_, _ = opened.Update(tea.KeyPressMsg(tea.Key{Text: "D"}))

	if len(h.Entries()) != 0 {
		t.Errorf("expected all entries cleared, got %+v", h.Entries())
	}
}

func TestResponsePane_ViewHistory_EscClosesViewer(t *testing.T) {
	pane, _ := setupPaneWithHistory(t)
	pane.SetResponse(&http.Response{StatusCode: 200}, &http.Request{Method: http.GET, URL: "https://a"})

	model, _ := pane.Update(keyRunes("H"))
	opened := model

	model, _ = opened.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	result := model
	if result.IsViewingHistory() {
		t.Error("expected esc to close the history viewer")
	}
}
