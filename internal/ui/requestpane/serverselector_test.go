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

// TestRebuildServerSelector_GroupWithDefaultServer_SkipsNoneOption is a
// regression test for a real bug report: a group with a configured
// default server (see SetGroupServer) still had its ServerSelector start
// on "(none)" - but the request went out to that default server
// regardless, since an unset request.ServerID falls through to the
// group's own default (see ResolveEffectiveServer). "(none)" looked like
// "no server will be used", which wasn't true whenever the group actually
// had one - the picker should start on that real server instead.
func TestRebuildServerSelector_GroupWithDefaultServer_SkipsNoneOption(t *testing.T) {
	pane, db := setupTestPane(t)

	if _, err := db.GetOrCreateGroup("petstore"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.UpsertServer("prod", "https://api.example.com"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.SetGroupServer("petstore", "prod"); err != nil {
		t.Fatalf("failed to set group server: %v", err)
	}
	group, err := db.GetGroupByName("petstore")
	if err != nil || group == nil {
		t.Fatalf("failed to load group: %v", err)
	}

	req := &http.Request{Method: http.GET, URL: "/pets", Headers: map[string]string{}, GroupID: &group.ID}
	pane.reinitRequestPane(req)

	if got := pane.ServerSelector.Value(); got != "prod" {
		t.Errorf("expected the selector to start on the group's default server %q, got %q", "prod", got)
	}

	// "(none)" must not be reachable at all - every server the group
	// allows resolves to the exact same effective server picking it
	// "none" would silently fall back to anyway, so there's nothing left
	// for that option to mean.
	start := pane.ServerSelector.Value()
	for i := 0; i < 5; i++ {
		pane.ServerSelector.Next()
		if pane.ServerSelector.Value() == noServerOption {
			t.Fatal("expected \"(none)\" to not be an option when the group has a default server")
		}
		if pane.ServerSelector.Value() == start {
			break // cycled all the way around
		}
	}
}

// TestRebuildServerSelector_GroupWithNoDefaultServer_StillOffersNone
// verifies the fix above didn't remove "(none)" unconditionally - a group
// with allowed servers but no configured *default* genuinely resolves to
// no server when "(none)" is selected (see ResolveEffectiveServer), so it
// stays accurate and available there.
func TestRebuildServerSelector_GroupWithNoDefaultServer_StillOffersNone(t *testing.T) {
	pane, db := setupTestPane(t)

	group, err := db.GetOrCreateGroup("petstore")
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.UpsertServer("dev", "http://localhost:3000"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	// Allowed, but never set as the group's default.
	if err := db.AddGroupServer("petstore", "dev"); err != nil {
		t.Fatalf("failed to add group server: %v", err)
	}

	req := &http.Request{Method: http.GET, URL: "/pets", Headers: map[string]string{}, GroupID: &group.ID}
	pane.reinitRequestPane(req)

	if got := pane.ServerSelector.Value(); got != noServerOption {
		t.Errorf("expected the selector to start on %q when the group has no default, got %q", noServerOption, got)
	}
}
