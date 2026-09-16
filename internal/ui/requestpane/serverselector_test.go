package requestpane

import (
	"strings"
	"testing"

	"github.com/owenHochwald/Volt/internal/http"
)

// TestRebuildServerSelector_UngroupedRequestHasNoServerField verifies a
// "clean" request with no group (GroupID == nil) has no server field at
// all - not a picker listing every server in the database (the earlier
// behavior), and not one scoped to some group's allowed set either, since
// there's no group to scope by and an ungrouped request typically
// addresses its server directly via its own absolute URL. See
// ShowServerSelector.
func TestRebuildServerSelector_UngroupedRequestHasNoServerField(t *testing.T) {
	pane, db := setupTestPane(t)

	if _, err := db.UpsertServer("dev", "http://localhost:3000"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if _, err := db.UpsertServer("prod", "https://api.example.com"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req := &http.Request{Method: http.GET, URL: "http://example.com/pets", Headers: map[string]string{}}
	pane.reinitRequestPane(req)

	if pane.ShowServerSelector() {
		t.Error("expected an ungrouped request to not show a server field")
	}
	if len(pane.Servers) != 0 {
		t.Errorf("expected an ungrouped request to list no servers, got %+v", pane.Servers)
	}

	nm := &NormalMode{}
	for _, c := range nm.components(&pane) {
		if c == pane.ServerSelector {
			t.Error("expected ServerSelector to not be in the focus cycle for an ungrouped request")
		}
	}

	content, _ := pane.fieldsContent()
	if strings.Contains(content, "Server") {
		t.Errorf("expected no rendered server field, got:\n%s", content)
	}
}

// TestRebuildServerSelector_GroupedRequestOnlyListsItsOwnAllowedServers is
// a regression test for a real feature request: a request belonging to a
// group must only ever offer that group's own allowed servers (see
// storage.AddGroupServer/ListGroupServers) - never a server that belongs
// to a different group/API entirely.
func TestRebuildServerSelector_GroupedRequestOnlyListsItsOwnAllowedServers(t *testing.T) {
	pane, db := setupTestPane(t)

	petstore, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.GetOrCreateGroup("other"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.UpsertServer("petstore-dev", "http://localhost:3000"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if _, err := db.UpsertServer("other-prod", "https://other.example.com"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.AddGroupServer("petstore", "petstore-dev"); err != nil {
		t.Fatalf("failed to add group server: %v", err)
	}
	if err := db.AddGroupServer("other", "other-prod"); err != nil {
		t.Fatalf("failed to add group server: %v", err)
	}

	req := &http.Request{Method: http.GET, URL: "/pets", Headers: map[string]string{}, GroupID: &petstore.ID}
	pane.reinitRequestPane(req)

	if len(pane.Servers) != 1 || pane.Servers[0].Name != "petstore-dev" {
		t.Fatalf("expected only petstore's own allowed server, got %+v", pane.Servers)
	}

	// The picker's options must reflect the same restriction - "other-prod"
	// (belonging to a different group) must not be selectable at all.
	found := false
	pane.ServerSelector.Next() // cycle from "(none)" to whatever comes next
	for i := 0; i < 5; i++ {
		if pane.ServerSelector.Value() == "other-prod" {
			found = true
			break
		}
		pane.ServerSelector.Next()
	}
	if found {
		t.Error("expected a different group's server to never be a selectable option")
	}
}
