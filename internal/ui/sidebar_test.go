package ui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
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

// TestSidebar_View_ShowsSelectedRequestDescription verifies the currently
// selected request's OpenAPI description (see http.Request.Description)
// renders somewhere in the sidebar's view - a hand-typed request (or one
// imported from a spec with no description) shows none at all, rather
// than a stray blank line.
func TestSidebar_View_ShowsSelectedRequestDescription(t *testing.T) {
	sidebar, _, req := setupSidebarWithRequest(t)
	req.Description = "Fetches the example resource."
	sidebar.SetRequests([]list.Item{RequestItem{title: req.Name, desc: req.URL, Request: req}})

	if got := sidebar.renderSelectedDescriptionLine(); !strings.Contains(got, "Fetches the example resource.") {
		t.Errorf("expected the selected request's description to render, got %q", got)
	}
	if !strings.Contains(sidebar.View(), "Fetches the example resource.") {
		t.Errorf("expected View() to include the description, got:\n%s", sidebar.View())
	}
}

// TestSidebar_View_NoDescriptionLineWhenRequestHasNone is the counterpart
// to the above - nothing to show means nothing rendered, not an empty
// line silently eating into listHeight's budget.
func TestSidebar_View_NoDescriptionLineWhenRequestHasNone(t *testing.T) {
	sidebar, _, _ := setupSidebarWithRequest(t)

	if got := sidebar.renderSelectedDescriptionLine(); got != "" {
		t.Errorf("expected no description line, got %q", got)
	}
}

func sendKey(t *testing.T, sidebar *SidebarPane, key string) tea.Cmd {
	t.Helper()
	_, cmd := sidebar.Update(tea.KeyPressMsg(tea.Key{Code: []rune(key)[0], Text: key}))
	return cmd
}

// sendCtrlKey sends ctrl+<letter> - configureFilterKeys' Filter/ClearFilter
// bindings (ctrl+f/ctrl+x) aren't single printable runes, so they need
// their own construction rather than sendKey's.
func sendCtrlKey(t *testing.T, sidebar *SidebarPane, letter rune) tea.Cmd {
	t.Helper()
	_, cmd := sidebar.Update(tea.KeyPressMsg(tea.Key{Code: letter, Mod: tea.ModCtrl}))
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

// TestSidebar_Esc_DoesNotQuitTheApp is a regression test for a real bug:
// bubbles' list.Model ships with esc bound to its own internal Quit
// keybinding (alongside "q"), returning tea.Quit the instant esc is
// pressed with the sidebar focused and no filter active - silently
// exiting the whole program instead of leaving esc free for its other
// uses elsewhere (closing a filter query, switching panels). "q" is left
// alone; only esc must stop quitting.
func TestSidebar_Esc_DoesNotQuitTheApp(t *testing.T) {
	sidebar, _, _ := setupSidebarWithRequest(t)

	_, cmd := sidebar.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if cmd == nil {
		return
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); ok {
		t.Fatal("expected esc to NOT quit the app while the sidebar is focused")
	}
}

// TestSidebar_WantsEscForItself covers every state that must claim esc for
// itself before the app's global quit-arming ever sees it (see
// app/update.go's overlayOpen) - actively typing a filter, a pending
// delete confirmation, and the ordinary case with none of that active. An
// already-*applied* filter (not being typed) is deliberately NOT one of
// these states - see TestSidebar_ClearFilter_MovedOffEsc.
func TestSidebar_WantsEscForItself(t *testing.T) {
	t.Run("false by default", func(t *testing.T) {
		sidebar, _, _ := setupSidebarWithRequest(t)
		if sidebar.WantsEscForItself() {
			t.Error("expected no active filter or pending delete")
		}
	})

	t.Run("true while typing a filter", func(t *testing.T) {
		sidebar, _, _ := setupSidebarWithRequest(t)
		sendCtrlKey(t, sidebar, 'f')
		if !sidebar.IsFiltering() {
			t.Fatal("expected ctrl+f to start filtering")
		}
		if !sidebar.WantsEscForItself() {
			t.Error("expected WantsEscForItself while actively filtering")
		}
	})

	t.Run("false with an applied (not being edited) filter", func(t *testing.T) {
		sidebar, _, _ := setupSidebarWithRequest(t)
		sendCtrlKey(t, sidebar, 'f')
		sendKey(t, sidebar, "x")
		sidebar.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
		if sidebar.IsFiltering() {
			t.Fatal("expected enter to apply, not still be typing, the filter")
		}
		if sidebar.WantsEscForItself() {
			t.Error("expected an applied filter not to claim esc - clearing it moved to ctrl+x")
		}
	})

	t.Run("true with a pending delete", func(t *testing.T) {
		sidebar, _, _ := setupSidebarWithRequest(t)
		sendKey(t, sidebar, "d")
		if sidebar.pendingDelete == nil {
			t.Fatal("expected \"d\" to arm a pending delete")
		}
		if !sidebar.WantsEscForItself() {
			t.Error("expected WantsEscForItself while a delete confirmation is pending")
		}
	})
}

// TestSidebar_ClearFilter_MovedOffEsc is a regression test for the
// deliberate follow-up to WantsEscForItself's original fix: esc clearing
// an *applied* filter turned out to be too easy to trigger by accident
// (just browsing the filtered list) - reported live. ctrl+x now does that
// instead (see configureFilterKeys); esc on an applied filter must do
// nothing to it at all (see TestFirstEscapeDoesNotClearAnAppliedSidebarFilter
// for the app-level half of this - it falls through to the ordinary
// quit-arming/panel-switch behavior instead).
func TestSidebar_ClearFilter_MovedOffEsc(t *testing.T) {
	sidebar, _, _ := setupSidebarWithRequest(t)
	sendCtrlKey(t, sidebar, 'f')
	sendKey(t, sidebar, "x")
	sidebar.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if sidebar.requestsList.FilterState() != list.FilterApplied {
		t.Fatalf("expected the filter to be applied, got state %v", sidebar.requestsList.FilterState())
	}

	sidebar.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if sidebar.requestsList.FilterState() != list.FilterApplied {
		t.Fatalf("expected esc to leave the applied filter alone, got state %v", sidebar.requestsList.FilterState())
	}

	sendCtrlKey(t, sidebar, 'x')
	if sidebar.requestsList.FilterState() != list.Unfiltered {
		t.Fatalf("expected ctrl+x to clear the applied filter, got state %v", sidebar.requestsList.FilterState())
	}
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

// TestRequestItem_FilterValue_FuzzyMatchesURL locks in that "/" search
// already fuzzy-matches against a request's full URL (not just its name):
// RequestItem.FilterValue includes the URL, and the sidebar's list never
// overrides list.Model's own Filter field, so list.DefaultFilter (backed
// by sahilm/fuzzy - genuine subsequence matching, not a plain substring
// check) is what actually runs. "vnd" matches "/vm/v3/node" as the
// subsequence v...n...d, even though it's not a literal substring of it.
func TestRequestItem_FilterValue_FuzzyMatchesURL(t *testing.T) {
	items := []RequestItem{
		{title: "getNodes", Request: &http.Request{Method: http.GET, URL: "http://127.0.0.1:1500/vm/v3/node"}},
		{title: "getHosts", Request: &http.Request{Method: http.GET, URL: "http://127.0.0.1:1500/vm/v3/host"}},
		{title: "getInvoices", Request: &http.Request{Method: http.GET, URL: "http://127.0.0.1:1500/billing/v2/invoice"}},
	}
	values := make([]string, len(items))
	for i, item := range items {
		values[i] = item.FilterValue()
	}

	ranks := list.DefaultFilter("vnd", values)
	if len(ranks) != 1 || ranks[0].Index != 0 {
		t.Fatalf("expected \"vnd\" to fuzzy-match only getNodes's URL (index 0), got %+v", ranks)
	}
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

// TestSidebar_GroupFilter_CyclingPersistsChoice is a regression test for a
// real feature request: changing the group filter ("[" / "]") should
// persist the choice (see storage.SaveLastGroupFilter) so the next app
// start reopens on it - not always reset to "All groups".
func TestSidebar_GroupFilter_CyclingPersistsChoice(t *testing.T) {
	sidebar, db := setupSidebarWithGroups(t)

	cmd := sendKey(t, sidebar, "]")
	if cmd == nil {
		t.Fatal("expected a non-nil command to persist the new filter")
	}
	cmd() // execute the fire-and-forget save

	want := sidebar.currentGroupFilter()
	if want == "" {
		t.Fatal("test setup error: expected cycling to actually move off \"All groups\"")
	}
	got, err := db.LoadLastGroupFilter()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("expected the persisted filter to match the newly selected group %q, got %q", want, got)
	}
}

// TestSidebar_GroupFilter_CyclingBackToAllGroupsPersistsThatToo verifies
// "All groups" is itself saved as a deliberate choice (an empty string),
// not skipped - otherwise cycling back to it would leave a stale non-empty
// value on disk that the next start would incorrectly restore.
func TestSidebar_GroupFilter_CyclingBackToAllGroupsPersistsThatToo(t *testing.T) {
	sidebar, db := setupSidebarWithGroups(t)

	sendKey(t, sidebar, "]")() // move off "All groups"
	if sidebar.currentGroupFilter() == "" {
		t.Fatal("test setup error: expected to have moved off \"All groups\"")
	}

	n := len(sidebar.groupOptions)
	for i := 0; i < n-1; i++ {
		cmd := sendKey(t, sidebar, "]")
		if cmd != nil {
			cmd()
		}
	}
	if sidebar.currentGroupFilter() != "" {
		t.Fatalf("test setup error: expected to have cycled all the way back to \"All groups\", got %q", sidebar.currentGroupFilter())
	}

	got, err := db.LoadLastGroupFilter()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected \"All groups\" to be persisted as an empty string, got %q", got)
	}
}

// TestSidebar_GroupFilter_RestoresFromPreviousSession verifies a group
// filter saved by a previous session (see storage.SaveLastGroupFilter) is
// applied on the sidebar's first load - not left sitting unused in the
// database.
func TestSidebar_GroupFilter_RestoresFromPreviousSession(t *testing.T) {
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	alpha, err := db.GetOrCreateGroup("alpha")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.GetOrCreateGroup("beta"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if err := db.SaveLastGroupFilter("alpha"); err != nil {
		t.Fatalf("failed to save group filter: %v", err)
	}

	req := &http.Request{Name: "in alpha", Method: http.GET, URL: "http://a", Headers: map[string]string{}, GroupID: &alpha.ID}
	if err := db.Save(req); err != nil {
		t.Fatalf("failed to save request: %v", err)
	}
	requests, err := db.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// NewSidebar (not setupSidebarWithGroups, which predates this restore
	// feature) is what actually reads the persisted filter - see its
	// pendingGroupFilterName.
	sidebar := NewSidebar(db, keybindings.DefaultKeyMap())
	sidebar.SetSize(40, 20)
	loadSidebarRequests(t, sidebar, requests)

	if got := sidebar.currentGroupFilter(); got != "alpha" {
		t.Errorf("expected the restored filter to be %q, got %q", "alpha", got)
	}
}

// TestSidebar_GroupFilter_RestoreOnlyAppliesOnFirstLoad verifies a second
// RequestsLoadingMsg this session (e.g. after saving/deleting/importing a
// request) does not re-apply the on-disk filter over whatever the user
// has since selected in this session - only the very first load gets that
// one chance (see restoredGroupFilter).
func TestSidebar_GroupFilter_RestoreOnlyAppliesOnFirstLoad(t *testing.T) {
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.GetOrCreateGroup("alpha"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.GetOrCreateGroup("beta"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if err := db.SaveLastGroupFilter("alpha"); err != nil {
		t.Fatalf("failed to save group filter: %v", err)
	}

	sidebar := NewSidebar(db, keybindings.DefaultKeyMap())
	sidebar.SetSize(40, 20)
	loadSidebarRequests(t, sidebar, nil) // first load: restores "alpha"
	if got := sidebar.currentGroupFilter(); got != "alpha" {
		t.Fatalf("test setup error: expected the first load to restore %q, got %q", "alpha", got)
	}

	// The user switches to "beta" mid-session, without saving it back to
	// disk (simulating a lost/skipped write, or just checking this
	// in-memory case directly).
	sidebar.groupFilterIndex = 0
	for i, name := range sidebar.groupOptions {
		if name == "beta" {
			sidebar.groupFilterIndex = i
		}
	}
	if sidebar.currentGroupFilter() != "beta" {
		t.Fatalf("test setup error: expected to have selected %q, got %q", "beta", sidebar.currentGroupFilter())
	}

	// A second load (e.g. after some other change) must preserve "beta",
	// not silently jump back to the on-disk "alpha".
	loadSidebarRequests(t, sidebar, nil)
	if got := sidebar.currentGroupFilter(); got != "beta" {
		t.Errorf("expected the second load to preserve the in-session selection %q, got %q", "beta", got)
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
func TestSidebarVimCountNavigation(t *testing.T) {
	tests := []struct {
		name  string
		items int
		start int
		keys  string
		want  int
	}{
		{name: "single j", items: 12, keys: "j", want: 1},
		{name: "ten down", items: 12, keys: "10j", want: 10},
		{name: "ten up wraps", items: 12, keys: "10k", want: 2},
		{name: "large count wraps", items: 12, start: 11, keys: "14j", want: 1},
		{name: "count resets after motion", items: 12, keys: "2jj", want: 3},
		{name: "unrelated key clears count", items: 12, keys: "2xj", want: 1},
		{name: "empty list remains safe", keys: "10j", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sidebar := NewSidebar(nil, keybindings.DefaultKeyMap())
			sidebar.SetRequests(sidebarTestItems(tt.items))
			if tt.items > 0 {
				sidebar.requestsList.Select(tt.start)
			}

			for _, input := range tt.keys {
				_, _ = sidebar.Update(sidebarKeyPress(input))
			}

			if got := sidebar.requestsList.Index(); got != tt.want {
				t.Fatalf("selected index = %d, want %d after %q", got, tt.want, tt.keys)
			}
		})
	}
}

func sidebarTestItems(count int) []list.Item {
	items := make([]list.Item, count)
	for i := range items {
		items[i] = RequestItem{title: fmt.Sprintf("request-%d", i)}
	}
	return items
}

func sidebarKeyPress(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Text: string(code)})
}
