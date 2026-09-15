package requestpane

import (
	"testing"

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
