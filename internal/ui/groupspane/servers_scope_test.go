package groupspane

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// typeString feeds each rune of s through Update, as if typed one key at a
// time - mirroring how a real keystroke sequence arrives.
func typeString(t *testing.T, pane GroupsPane, s string) GroupsPane {
	t.Helper()
	for _, r := range s {
		pane = pressKey(t, pane, string(r))
	}
	return pane
}

// pressSpecial sends a non-character key (tab/enter/...) by its tea.KeyCode
// rather than pressKey's rune-based construction, which only works for
// plain single-character keys.
func pressSpecial(t *testing.T, pane GroupsPane, code rune) GroupsPane {
	t.Helper()
	updated, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: code}))
	return updated
}

// TestGroupsPane_Servers_ScopedToSelectedGroup is a regression test for a
// real feature request: the browse view's Servers list must only ever
// show the currently selected group's own servers (see
// db.ListGroupServers), not every server in the database.
func TestGroupsPane_Servers_ScopedToSelectedGroup(t *testing.T) {
	pane, db := setupTestPane(t)
	if _, err := db.GetOrCreateGroup("alpha"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.GetOrCreateGroup("beta"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.UpsertServer("alpha-server", "http://alpha.example.com"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.AddGroupServer("alpha", "alpha-server"); err != nil {
		t.Fatalf("failed to add server to group: %v", err)
	}
	pane.Open()
	pane.target = focusGroups

	// Groups are listed alphabetically (see ListGroups) - "alpha" first.
	if item, ok := pane.selectedGroup(); !ok || item.group.Name != "alpha" {
		t.Fatalf("test setup error: expected \"alpha\" selected first, got %+v (ok=%v)", item, ok)
	}
	if len(pane.servers.Items()) != 1 {
		t.Fatalf("expected 1 server shown for \"alpha\", got %d: %+v", len(pane.servers.Items()), pane.servers.Items())
	}

	pane = pressKey(t, pane, "j")
	if item, ok := pane.selectedGroup(); !ok || item.group.Name != "beta" {
		t.Fatalf("test setup error: expected \"beta\" selected after moving down, got %+v (ok=%v)", item, ok)
	}
	if len(pane.servers.Items()) != 0 {
		t.Errorf("expected no servers shown for \"beta\" (none assigned), got %d: %+v", len(pane.servers.Items()), pane.servers.Items())
	}
}

// TestGroupsPane_CreateServer_ConnectsToSelectedGroup is a regression test
// for a real feature request: a server created from the TUI ("n" while
// browsing Servers) must be connected to whichever group is currently
// selected right away, since there's no "assign server" step anymore -
// creating one for a group is the only way to give it a server at all.
func TestGroupsPane_CreateServer_ConnectsToSelectedGroup(t *testing.T) {
	pane, db := setupTestPane(t)
	if _, err := db.GetOrCreateGroup("alpha"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	pane.Open()
	pane.target = focusServers

	pane = pressKey(t, pane, "n")
	if pane.mode != modeCreateServer {
		t.Fatalf("expected \"n\" to open the new-server form, got mode=%v", pane.mode)
	}
	pane = typeString(t, pane, "newserver")
	pane = pressSpecial(t, pane, tea.KeyTab)
	pane = typeString(t, pane, "http://new.example.com")
	pane = pressSpecial(t, pane, tea.KeyEnter)

	if pane.statusErr {
		t.Fatalf("expected a success status, got error: %s", pane.statusMsg)
	}

	group, err := db.GetGroupByName("alpha")
	if err != nil || group == nil {
		t.Fatalf("expected \"alpha\" to exist, err=%v", err)
	}
	servers, err := db.ListGroupServers(group.ID)
	if err != nil {
		t.Fatalf("ListGroupServers: %v", err)
	}
	if len(servers) != 1 || servers[0].Name != "newserver" {
		t.Errorf("expected \"newserver\" connected to \"alpha\", got %+v", servers)
	}
}

// TestGroupsPane_CreateServer_RequiresAGroupFirst verifies creating a
// server is blocked (with an explanatory status, not a silent no-op or a
// crash) when there's no group to connect it to yet.
func TestGroupsPane_CreateServer_RequiresAGroupFirst(t *testing.T) {
	pane, _ := setupTestPane(t)
	pane.Open()
	pane.target = focusServers

	pane = pressKey(t, pane, "n")
	if pane.mode != modeBrowse {
		t.Fatalf("expected \"n\" to stay in browse mode with no group to attach to, got mode=%v", pane.mode)
	}
	if !pane.statusErr || pane.statusMsg == "" {
		t.Error("expected an explanatory error status")
	}
}
