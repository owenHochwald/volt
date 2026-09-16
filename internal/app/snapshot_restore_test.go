package app

import (
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/utils"
)

// TestSetupModel_RestoresLastSentRequest is a regression test for the
// "restore the last sent request after a restart" feature: SetupModel
// (what runs on every app start) must pick up a previously saved snapshot
// (see requestpane's handleSubmit, which is what saves one) and load it
// into the request pane, focused so it's immediately visible - not just
// leave it sitting unread in the database.
func TestSetupModel_RestoresLastSentRequest(t *testing.T) {
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	snapshot := &http.RequestSnapshot{
		Request: http.Request{
			Method:  http.POST,
			URL:     "https://api.example.com/hosts/{host_id}",
			Headers: map[string]string{},
			PathParamSchema: map[string]http.BodyFieldSchema{
				"host_id": {Type: "integer"},
			},
		},
		PathParamValues: map[string]string{"host_id": "42"},
	}
	if err := db.SaveLastSentRequest(snapshot); err != nil {
		t.Fatalf("failed to save snapshot: %v", err)
	}

	m := SetupModel(db)

	if m.focusedPanel != utils.RequestPanel {
		t.Errorf("expected the request panel to be focused after restoring a snapshot, got %v", m.focusedPanel)
	}
	if len(m.requestPane.PathParams) != 1 || m.requestPane.PathParams[0].Value() != "42" {
		t.Fatalf("expected host_id=42 restored into PathParams, got %+v", m.requestPane.PathParams)
	}
	if m.requestPane.Request.URL != snapshot.Request.URL {
		t.Errorf("expected the restored request's URL to match the snapshot, got %q", m.requestPane.Request.URL)
	}
}

// TestSetupModel_NoSnapshot_StartsOnSidebarAsUsual verifies a database
// with nothing ever sent starts exactly as before this feature existed -
// sidebar focused, no path params conjured from nowhere.
func TestSetupModel_NoSnapshot_StartsOnSidebarAsUsual(t *testing.T) {
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	m := SetupModel(db)

	if m.focusedPanel != utils.SidebarPanel {
		t.Errorf("expected the sidebar to stay focused with no snapshot to restore, got %v", m.focusedPanel)
	}
}
