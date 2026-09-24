package requestpane

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/history"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/ui"
)

// recordSend is a small helper mirroring resolvedRequest's own shape - a
// fully-resolved SentRequest (no {name}/{{var}} placeholders left), the
// way sync.go actually builds one right before sending.
func recordSend(store *history.Store, method, url, body string) {
	store.Record(&http.Request{
		ID:      99, // a template's saved ID - must not carry over (see requestFromHistoryEntry)
		Name:    "widget lookup",
		Method:  method,
		URL:     url,
		Headers: map[string]string{"Authorization": "Bearer abc123"},
		Body:    body,
		PathParamSchema: map[string]http.BodyFieldSchema{
			"id": {Type: "integer"},
		},
	}, &http.Response{StatusCode: 200}, time.Now())
}

func TestRequestFromHistoryEntry_StripsSchemaAndCarriesLiteralValues(t *testing.T) {
	sent := &http.Request{
		ID:      42,
		Name:    "widget lookup",
		Method:  http.GET,
		URL:     "https://api.example.com/widgets/7", // resolved, no {id} left
		Headers: map[string]string{"Authorization": "Bearer abc123"},
		Body:    `{"count":7}`,
		PathParamSchema: map[string]http.BodyFieldSchema{
			"id": {Type: "integer"},
		},
		ParamValues: http.ParamValues{PathParamValues: map[string]string{"id": "{{widget_id}}"}},
	}

	got := requestFromHistoryEntry(sent)

	if got.ID != 0 {
		t.Errorf("expected ID cleared (a fresh snapshot, not a reopen), got %d", got.ID)
	}
	if got.URL != sent.URL {
		t.Errorf("expected the literal resolved URL %q, got %q", sent.URL, got.URL)
	}
	if got.Body != sent.Body {
		t.Errorf("expected the literal resolved body %q, got %q", sent.Body, got.Body)
	}
	if got.Headers["Authorization"] != "Bearer abc123" {
		t.Errorf("expected headers to carry over, got %+v", got.Headers)
	}
	if got.PathParamSchema != nil {
		t.Errorf("expected PathParamSchema cleared (URL has no {name} left to derive it from), got %+v", got.PathParamSchema)
	}
	if got.ParamValues.PathParamValues != nil {
		t.Errorf("expected ParamValues cleared (would reintroduce {{widget_id}} instead of the sent literal), got %+v", got.ParamValues)
	}
}

func TestRequestFromHistoryEntry_HeadersAreACopyNotAnAlias(t *testing.T) {
	sent := &http.Request{Method: http.GET, URL: "https://x", Headers: map[string]string{"X-A": "1"}}
	got := requestFromHistoryEntry(sent)
	got.Headers["X-A"] = "mutated"
	if sent.Headers["X-A"] != "1" {
		t.Errorf("expected the original history entry's headers to stay untouched, got %+v", sent.Headers)
	}
}

func TestRequestPane_StepRequestHistory_OlderLoadsNewestFirst(t *testing.T) {
	pane, _ := setupTestPane(t)
	store := history.NewStore()
	pane.SetHistory(store)
	recordSend(store, http.GET, "https://api.example.com/a", "")
	recordSend(store, http.GET, "https://api.example.com/b", "")

	pane.stepRequestHistory(1) // "["

	if !pane.browsingHistory {
		t.Fatal("expected browsingHistory true after stepping older")
	}
	if pane.historyPosition != 0 {
		t.Errorf("expected position 0 (newest), got %d", pane.historyPosition)
	}
	if pane.URLInput.Value() != "https://api.example.com/b" {
		t.Errorf("expected the most recently sent URL loaded, got %q", pane.URLInput.Value())
	}
}

func TestRequestPane_StepRequestHistory_NewerRestoresPreHistoryEdits(t *testing.T) {
	pane, _ := setupTestPane(t)
	store := history.NewStore()
	pane.SetHistory(store)
	recordSend(store, http.GET, "https://api.example.com/a", "")

	pane.URLInput.SetValue("https://unsent.example.com/still-editing")
	pane.NameInput.SetValue("unsent draft")

	pane.stepRequestHistory(1) // "[" - into history
	if pane.URLInput.Value() != "https://api.example.com/a" {
		t.Fatalf("expected history entry loaded, got %q", pane.URLInput.Value())
	}

	pane.stepRequestHistory(-1) // "]" - back past the newest entry

	if pane.browsingHistory {
		t.Error("expected browsingHistory false after stepping newer past the newest entry")
	}
	if pane.URLInput.Value() != "https://unsent.example.com/still-editing" {
		t.Errorf("expected the pre-browse draft URL restored, got %q", pane.URLInput.Value())
	}
	if pane.NameInput.Value() != "unsent draft" {
		t.Errorf("expected the pre-browse draft name restored, got %q", pane.NameInput.Value())
	}
}

func TestRequestPane_StepRequestHistory_OlderStopsAtOldestEntry(t *testing.T) {
	pane, _ := setupTestPane(t)
	store := history.NewStore()
	pane.SetHistory(store)
	recordSend(store, http.GET, "https://api.example.com/a", "")
	recordSend(store, http.GET, "https://api.example.com/b", "")

	pane.stepRequestHistory(1) // position 0 (b)
	pane.stepRequestHistory(1) // position 1 (a, the oldest)
	pane.stepRequestHistory(1) // no-op, already oldest

	if pane.historyPosition != 1 {
		t.Errorf("expected to stay at the oldest entry (position 1), got %d", pane.historyPosition)
	}
	if pane.URLInput.Value() != "https://api.example.com/a" {
		t.Errorf("expected the oldest entry still loaded, got %q", pane.URLInput.Value())
	}
}

func TestRequestPane_StepRequestHistory_NewerBeforeBrowsingIsANoop(t *testing.T) {
	pane, _ := setupTestPane(t)
	store := history.NewStore()
	pane.SetHistory(store)
	recordSend(store, http.GET, "https://api.example.com/a", "")
	pane.URLInput.SetValue("https://unsent.example.com/draft")

	pane.stepRequestHistory(-1) // "]" with nothing browsed yet

	if pane.browsingHistory {
		t.Error("expected browsingHistory to stay false")
	}
	if pane.URLInput.Value() != "https://unsent.example.com/draft" {
		t.Errorf("expected the draft URL untouched, got %q", pane.URLInput.Value())
	}
}

func TestRequestPane_StepRequestHistory_EmptyHistoryIsANoop(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.SetHistory(history.NewStore())
	pane.URLInput.SetValue("https://unsent.example.com/draft")

	pane.stepRequestHistory(1)

	if pane.browsingHistory {
		t.Error("expected browsingHistory to stay false with no history recorded")
	}
	if pane.URLInput.Value() != "https://unsent.example.com/draft" {
		t.Errorf("expected the draft URL untouched, got %q", pane.URLInput.Value())
	}
}

func TestRequestPane_HistoryViewer_EnterLoadsSelectedEntry(t *testing.T) {
	pane, _ := setupTestPane(t)
	store := history.NewStore()
	pane.SetHistory(store)
	recordSend(store, http.GET, "https://api.example.com/a", "")
	recordSend(store, http.POST, "https://api.example.com/b", `{"x":1}`)

	pane.openHistoryViewer()
	if !pane.IsViewingHistory() {
		t.Fatal("expected the viewer to be open")
	}
	pane.historyList.Select(1) // the older entry ("a")

	pane, cmd := pane.handleRequestHistoryViewerKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd != nil {
		t.Errorf("expected no command from loading a history entry, got %v", cmd)
	}

	if pane.IsViewingHistory() {
		t.Error("expected the viewer to close after enter")
	}
	if pane.URLInput.Value() != "https://api.example.com/a" {
		t.Errorf("expected the selected (older) entry loaded, got %q", pane.URLInput.Value())
	}
	if pane.MethodSelector.Current() != http.GET {
		t.Errorf("expected method GET, got %q", pane.MethodSelector.Current())
	}
}

func TestRequestPane_HistoryViewer_EscClosesWithoutLoading(t *testing.T) {
	pane, _ := setupTestPane(t)
	store := history.NewStore()
	pane.SetHistory(store)
	recordSend(store, http.GET, "https://api.example.com/a", "")
	pane.URLInput.SetValue("https://unsent.example.com/draft")

	pane.openHistoryViewer()
	pane, _ = pane.handleRequestHistoryViewerKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))

	if pane.IsViewingHistory() {
		t.Error("expected esc to close the viewer")
	}
	if pane.URLInput.Value() != "https://unsent.example.com/draft" {
		t.Errorf("expected esc to leave the draft untouched, got %q", pane.URLInput.Value())
	}
}

// TestRequestPane_HistoryKeys_DoNotFireWhileEditing verifies "[" types
// literally into the Body textarea (JSON arrays use it constantly)
// instead of being swallowed as RequestHistoryOlder - the IsEditing()
// guard in Update's dispatch.
func TestRequestPane_HistoryKeys_DoNotFireWhileEditing(t *testing.T) {
	pane, _ := setupTestPane(t)
	store := history.NewStore()
	pane.SetHistory(store)
	recordSend(store, http.GET, "https://api.example.com/a", "")
	pane.PanelFocused = true

	// A POST with no path params and no known body schema focuses the
	// freeform Body textarea directly (see
	// TestSetRequestPaneRequestMsg_FocusesBodyWhenNoPathParams).
	req := &http.Request{Method: http.POST, URL: "https://api.example.com/hosts", Headers: map[string]string{}}
	pane, _ = pane.Update(ui.SetRequestPaneRequestMsg{Request: req})
	if pane.FocusManager.Current() != pane.Body {
		t.Fatalf("test setup error: expected focus on the Body textarea, got %T", pane.FocusManager.Current())
	}

	updated, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: '[', Text: "["}))

	if updated.browsingHistory {
		t.Error("expected \"[\" typed into the Body textarea not to trigger history browsing")
	}
	if updated.Body.Value() != "[" {
		t.Errorf("expected \"[\" to be typed literally into the Body textarea, got %q", updated.Body.Value())
	}
}
