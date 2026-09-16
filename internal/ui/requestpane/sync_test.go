package requestpane

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func setupTestPane(t *testing.T) (RequestPane, *storage.SQLiteStorage) {
	t.Helper()
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pane := SetupRequestPane(db, keybindings.DefaultKeyMap())
	return pane, db
}

func TestRequestPane_ResolvedRequest_NoGroup(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.Request = &http.Request{Method: http.GET, URL: "http://example.com/x"}

	resolved := pane.resolvedRequest()
	if resolved != pane.Request {
		t.Errorf("expected the same request when there's no group, got a different pointer/value: %+v", resolved)
	}
}

func TestRequestPane_ResolvedRequest_GroupWithServer(t *testing.T) {
	pane, db := setupTestPane(t)

	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.UpsertServer("petstore", "https://api.example.com/v1"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.SetGroupServer("petstore", "petstore"); err != nil {
		t.Fatalf("failed to assign server: %v", err)
	}

	pane.Request = &http.Request{Method: http.GET, URL: "/pets", GroupID: &group.ID}

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/v1/pets" {
		t.Errorf("expected resolved URL to use the group's server, got %q", resolved.URL)
	}
	// The stored request must stay untouched (relative), so editing/saving
	// keeps following the group's server rather than freezing an absolute URL.
	if pane.Request.URL != "/pets" {
		t.Errorf("expected the original request to remain relative, got %q", pane.Request.URL)
	}
}

func TestRequestPane_ResolvedRequest_GroupWithoutServer(t *testing.T) {
	pane, db := setupTestPane(t)

	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	pane.Request = &http.Request{Method: http.GET, URL: "/pets", GroupID: &group.ID}

	resolved := pane.resolvedRequest()
	if resolved.URL != "/pets" {
		t.Errorf("expected the URL to stay relative when the group has no server, got %q", resolved.URL)
	}
}

// TestRequestPane_ResolvedRequest_DirectServer covers a request that picks a
// server directly (via ServerID) rather than through a group - its relative
// URL must resolve against that server, and the server's default headers
// must be merged in underneath the request's own.
func TestRequestPane_ResolvedRequest_DirectServer(t *testing.T) {
	pane, db := setupTestPane(t)

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.UpdateServerHeaders("prod", map[string]string{"X-Env": "prod", "Authorization": "Bearer server-token"}); err != nil {
		t.Fatalf("failed to set server headers: %v", err)
	}

	pane.Request = &http.Request{
		Method:   http.GET,
		URL:      "/pets",
		Headers:  map[string]string{"Authorization": "Bearer request-token"},
		ServerID: &prod.ID,
	}

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/pets" {
		t.Errorf("expected the URL to resolve against the directly-selected server, got %q", resolved.URL)
	}
	if resolved.Headers["X-Env"] != "prod" {
		t.Errorf("expected the server's default header to be merged in, got %+v", resolved.Headers)
	}
	if resolved.Headers["Authorization"] != "Bearer request-token" {
		t.Errorf("expected the request's own header to win over the server's default, got %+v", resolved.Headers)
	}
	if pane.Request.URL != "/pets" {
		t.Errorf("expected the stored request to stay relative, got %q", pane.Request.URL)
	}
}

// TestRequestPane_ResolvedRequest_ServerIDOverridesGroup covers a request
// that has both a group and a direct ServerID - the direct selection must
// win, letting a single request point somewhere different from the rest of
// its group without leaving the group.
func TestRequestPane_ResolvedRequest_ServerIDOverridesGroup(t *testing.T) {
	pane, db := setupTestPane(t)

	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.UpsertServer("staging", "https://staging.example.com"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.SetGroupServer("petstore", "staging"); err != nil {
		t.Fatalf("failed to assign group server: %v", err)
	}
	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	pane.Request = &http.Request{Method: http.GET, URL: "/pets", GroupID: &group.ID, ServerID: &prod.ID}

	resolved := pane.resolvedRequest()
	if resolved.URL != "https://api.example.com/pets" {
		t.Errorf("expected the direct ServerID to win over the group's server, got %q", resolved.URL)
	}
}

// TestRequestPane_ServerSelector_SyncsRequestServerID covers the ServerSelector
// field itself: cycling to a server name and syncing must set the request's
// ServerID, and cycling back to "(none)" must clear it.
func TestRequestPane_ServerSelector_SyncsRequestServerID(t *testing.T) {
	pane, db := setupTestPane(t)

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req := &http.Request{Method: http.GET, URL: "http://example.com"}
	pane.reinitRequestPane(req)

	if pane.ServerSelector.Value() != noServerOption {
		t.Fatalf("expected a request with no ServerID to start on %q, got %q", noServerOption, pane.ServerSelector.Value())
	}

	pane.ServerSelector.Next()
	if pane.ServerSelector.Value() != "prod" {
		t.Fatalf("expected the only configured server to be next in the cycle, got %q", pane.ServerSelector.Value())
	}

	pane.syncRequest()
	if pane.Request.ServerID == nil || *pane.Request.ServerID != prod.ID {
		t.Errorf("expected syncRequest to set ServerID from the selector, got %+v", pane.Request.ServerID)
	}

	pane.ServerSelector.Next()
	if pane.ServerSelector.Value() != noServerOption {
		t.Fatalf("expected cycling past the only server to return to %q, got %q", noServerOption, pane.ServerSelector.Value())
	}
	pane.syncRequest()
	if pane.Request.ServerID != nil {
		t.Errorf("expected syncRequest to clear ServerID for %q, got %+v", noServerOption, pane.Request.ServerID)
	}
}

// TestRequestPane_ReinitRequestPane_ServerSelectorReflectsServerID covers
// loading a request that already has a ServerID: the selector must start on
// that server's name, not "(none)".
func TestRequestPane_ReinitRequestPane_ServerSelectorReflectsServerID(t *testing.T) {
	pane, db := setupTestPane(t)

	prod, err := db.UpsertServer("prod", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req := &http.Request{Method: http.GET, URL: "/pets", ServerID: &prod.ID}
	pane.reinitRequestPane(req)

	if pane.ServerSelector.Value() != "prod" {
		t.Errorf("expected the selector to start on the request's server, got %q", pane.ServerSelector.Value())
	}
}

// TestRequestPane_ServerSelector_CyclesViaFocusManager reproduces the actual
// key-handling path (FocusManager.Current() + HandleInput), not a direct
// call on the ServerSelector field - reinitRequestPane always rebuilds a
// fresh ServerSelector, but for a request with no path/body params (so
// syncPathParams/syncBodyParams don't themselves trigger a focus-cycle
// rebuild), FocusManager could be left pointing at the *previous*,
// now-orphaned ServerSelector instance. Cycling it via the real input path
// must still reach the field View() actually renders.
func TestRequestPane_ServerSelector_CyclesViaFocusManager(t *testing.T) {
	pane, db := setupTestPane(t)

	if _, err := db.UpsertServer("prod", "https://api.example.com"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// A request with no path params and no body schema - both
	// syncPathParams and syncBodyParams are no-ops for it.
	req := &http.Request{Method: http.GET, URL: "http://example.com", Headers: map[string]string{}}
	pane.reinitRequestPane(req)
	pane.PanelFocused = true

	if pane.FocusManager.CurrentIndex() != 0 {
		t.Fatalf("expected the server selector (first field) focused initially, got index %d", pane.FocusManager.CurrentIndex())
	}

	model, _ := pane.currentMode.HandleInput(&pane, tea.KeyMsg{Type: tea.KeyRight})
	result, ok := model.(*RequestPane)
	if !ok {
		t.Fatalf("expected *RequestPane, got %T", model)
	}

	if result.ServerSelector.Value() != "prod" {
		t.Errorf("expected cycling via the real input path to change the selector actually rendered, got %q", result.ServerSelector.Value())
	}
}
