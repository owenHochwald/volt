package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func setupSidebarWithRequest(t *testing.T) (*SidebarPane, *storage.SQLiteStorage, *http.Request) {
	t.Helper()
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	req := &http.Request{Name: "test req", Method: http.GET, URL: "http://example.com"}
	if err := db.Save(req); err != nil {
		t.Fatalf("failed to save request: %v", err)
	}

	sidebar := NewSidebar(db, keybindings.DefaultKeyMap())
	sidebar.SetSize(40, 20)
	sidebar.SetRequests([]list.Item{RequestItem{title: req.Name, desc: req.URL, Request: req}})
	return sidebar, db, req
}

func sendKey(t *testing.T, sidebar *SidebarPane, key string) tea.Cmd {
	t.Helper()
	_, cmd := sidebar.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return cmd
}

// loadSidebarRequests feeds requests through the same RequestsLoadingMsg
// path Init()'s LoadRequestsCmd produces, so group filter state (groupNames/
// groupOptions) is populated exactly as it would be in the real app instead
// of being bypassed by calling SetRequests directly.
func loadSidebarRequests(t *testing.T, sidebar *SidebarPane, requests []http.Request) {
	t.Helper()
	sidebar.Update(RequestsLoadingMsg{Requests: requests})
}

func requestItemTitles(sidebar *SidebarPane) []string {
	titles := make([]string, 0)
	for _, item := range sidebar.requestsList.Items() {
		titles = append(titles, item.(RequestItem).title)
	}
	return titles
}

func TestSidebar_DeleteRequest_RequiresConfirmation(t *testing.T) {
	sidebar, db, _ := setupSidebarWithRequest(t)

	cmd := sendKey(t, sidebar, "d")
	if sidebar.pendingDelete == nil {
		t.Fatal("expected DeleteRequest to arm a pending confirmation instead of deleting immediately")
	}
	if cmd != nil {
		t.Error("expected no delete command to be issued before confirmation")
	}

	requests, err := db.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected the request to still exist before confirmation, got %d requests", len(requests))
	}
}

func TestSidebar_DeleteRequest_CancelKeepsRequest(t *testing.T) {
	sidebar, db, _ := setupSidebarWithRequest(t)

	sendKey(t, sidebar, "d")
	cmd := sendKey(t, sidebar, "n")

	if sidebar.pendingDelete != nil {
		t.Error("expected the pending delete to be cleared after a non-confirming key")
	}
	if cmd != nil {
		t.Error("expected cancelling to issue no command")
	}

	requests, err := db.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(requests) != 1 {
		t.Errorf("expected the request to survive a cancelled delete, got %d requests", len(requests))
	}
}

func TestSidebar_DeleteRequest_ConfirmDeletes(t *testing.T) {
	sidebar, db, req := setupSidebarWithRequest(t)

	sendKey(t, sidebar, "d")
	cmd := sendKey(t, sidebar, "y")

	if sidebar.pendingDelete != nil {
		t.Error("expected the pending delete to be cleared after confirming")
	}
	if cmd == nil {
		t.Fatal("expected confirming to issue the delete command")
	}

	msg, ok := cmd().(RequestDeletedMsg)
	if !ok {
		t.Fatalf("expected RequestDeletedMsg, got %T", msg)
	}
	if msg.Err != nil {
		t.Fatalf("delete command failed: %v", msg.Err)
	}
	if msg.ID != req.ID {
		t.Errorf("expected delete for request %d, got %d", req.ID, msg.ID)
	}

	requests, err := db.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(requests) != 0 {
		t.Errorf("expected the confirmed delete to remove the request, got %d requests", len(requests))
	}
}

func setupSidebarWithGroups(t *testing.T) (*SidebarPane, *storage.SQLiteStorage) {
	t.Helper()
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	alpha, err := db.GetOrCreateGroup("alpha")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	beta, err := db.GetOrCreateGroup("beta")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	for _, req := range []*http.Request{
		{Name: "in alpha", Method: http.GET, URL: "http://a", Headers: map[string]string{}, GroupID: &alpha.ID},
		{Name: "in beta", Method: http.GET, URL: "http://b", Headers: map[string]string{}, GroupID: &beta.ID},
		{Name: "ungrouped", Method: http.GET, URL: "http://c", Headers: map[string]string{}},
	} {
		if err := db.Save(req); err != nil {
			t.Fatalf("failed to save request: %v", err)
		}
	}

	requests, err := db.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	sidebar := NewSidebar(db, keybindings.DefaultKeyMap())
	sidebar.SetSize(40, 20)
	loadSidebarRequests(t, sidebar, requests)
	return sidebar, db
}

func TestSidebar_GroupFilter_StartsShowingEveryRequest(t *testing.T) {
	sidebar, _ := setupSidebarWithGroups(t)

	if got := sidebar.currentGroupFilter(); got != "" {
		t.Errorf("expected no filter (all groups) initially, got %q", got)
	}
	titles := requestItemTitles(sidebar)
	if len(titles) != 3 {
		t.Errorf("expected all 3 requests visible with no filter, got %v", titles)
	}
}

func TestSidebar_GroupFilter_CyclesAndNarrowsTheList(t *testing.T) {
	sidebar, _ := setupSidebarWithGroups(t)

	// groupOptions is ["All groups", "alpha", "beta"] (sorted) - cycling
	// forward once lands on "alpha".
	sendKey(t, sidebar, "]")
	if got := sidebar.currentGroupFilter(); got != "alpha" {
		t.Fatalf("expected the filter to land on %q, got %q", "alpha", got)
	}
	if titles := requestItemTitles(sidebar); len(titles) != 1 || titles[0] != "in alpha" {
		t.Errorf("expected only alpha's request visible, got %v", titles)
	}

	sendKey(t, sidebar, "]")
	if got := sidebar.currentGroupFilter(); got != "beta" {
		t.Fatalf("expected the filter to land on %q, got %q", "beta", got)
	}
	if titles := requestItemTitles(sidebar); len(titles) != 1 || titles[0] != "in beta" {
		t.Errorf("expected only beta's request visible, got %v", titles)
	}

	// Cycling forward past the last group wraps back to "All groups".
	sendKey(t, sidebar, "]")
	if got := sidebar.currentGroupFilter(); got != "" {
		t.Errorf("expected wrapping forward to return to no filter, got %q", got)
	}
	if titles := requestItemTitles(sidebar); len(titles) != 3 {
		t.Errorf("expected all 3 requests visible again, got %v", titles)
	}

	// Cycling backward from "All groups" wraps to the last group ("beta").
	sendKey(t, sidebar, "[")
	if got := sidebar.currentGroupFilter(); got != "beta" {
		t.Errorf("expected wrapping backward to land on %q, got %q", "beta", got)
	}
}

func TestSidebar_GroupFilter_FallsBackWhenSelectedGroupDisappears(t *testing.T) {
	sidebar, db := setupSidebarWithGroups(t)

	sendKey(t, sidebar, "]") // -> "alpha"
	if sidebar.currentGroupFilter() != "alpha" {
		t.Fatalf("setup: expected filter on alpha")
	}

	// Delete the "alpha" group itself (not just its requests) so the next
	// load's groupOptions genuinely no longer contains it - the stale index
	// must not point at the wrong group silently.
	if _, err := db.DeleteGroup("alpha"); err != nil {
		t.Fatalf("failed to delete group: %v", err)
	}
	requests, err := db.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	loadSidebarRequests(t, sidebar, requests)

	if got := sidebar.currentGroupFilter(); got != "" {
		t.Errorf("expected the filter to reset to all groups when the selected group vanished, got %q", got)
	}
	// DeleteGroup cascades to the requests that were in it, so only "in
	// beta" and "ungrouped" remain.
	if titles := requestItemTitles(sidebar); len(titles) != 2 {
		t.Errorf("expected the 2 surviving requests visible, got %v", titles)
	}
}
