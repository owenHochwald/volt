package groupspane

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/storage"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func setupTestPane(t *testing.T) (GroupsPane, *storage.SQLiteStorage) {
	t.Helper()
	db, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite storage: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	pane := SetupGroupsPane(db, keybindings.DefaultKeyMap())
	pane.Open()
	return pane, db
}

func pressKey(t *testing.T, pane GroupsPane, key string) GroupsPane {
	t.Helper()
	updated, _ := pane.Update(tea.KeyPressMsg(tea.Key{Code: []rune(key)[0], Text: key}))
	return updated
}

func TestGroupsPane_DeleteGroup_RequiresConfirmation(t *testing.T) {
	pane, db := setupTestPane(t)
	if _, err := db.GetOrCreateGroup("testgroup"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	pane.Open()
	pane.target = focusGroups

	pane = pressKey(t, pane, "d")
	if pane.pendingDelete == nil {
		t.Fatal("expected DeleteItem to arm a pending confirmation instead of deleting immediately")
	}

	groups, err := db.ListGroups()
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected the group to still exist before confirmation, got %d groups", len(groups))
	}
}

func TestGroupsPane_DeleteGroup_CancelKeepsGroup(t *testing.T) {
	pane, db := setupTestPane(t)
	if _, err := db.GetOrCreateGroup("testgroup"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	pane.Open()
	pane.target = focusGroups

	pane = pressKey(t, pane, "d")
	pane = pressKey(t, pane, "n")

	if pane.pendingDelete != nil {
		t.Error("expected the pending delete to be cleared after a non-confirming key")
	}
	groups, err := db.ListGroups()
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected the group to survive a cancelled delete, got %d groups", len(groups))
	}
}

func TestGroupsPane_DeleteGroup_ConfirmDeletes(t *testing.T) {
	pane, db := setupTestPane(t)
	if _, err := db.GetOrCreateGroup("testgroup"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	pane.Open()
	pane.target = focusGroups

	pane = pressKey(t, pane, "d")
	pane = pressKey(t, pane, "y")

	if pane.pendingDelete != nil {
		t.Error("expected the pending delete to be cleared after confirming")
	}
	groups, err := db.ListGroups()
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected the confirmed delete to remove the group, got %d groups", len(groups))
	}
	if pane.statusErr {
		t.Errorf("expected a success status, got error status: %s", pane.statusMsg)
	}
}

func TestGroupsPane_DeleteServer_ConfirmDeletes(t *testing.T) {
	pane, db := setupTestPane(t)
	if _, err := db.UpsertServer("testserver", "http://localhost:9999"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	pane.Open()
	pane.target = focusServers

	pane = pressKey(t, pane, "d")
	if pane.pendingDelete == nil {
		t.Fatal("expected DeleteItem to arm a pending confirmation for the server")
	}
	pane = pressKey(t, pane, "y")

	servers, err := db.ListServers()
	if err != nil {
		t.Fatalf("ListServers: %v", err)
	}
	if len(servers) != 0 {
		t.Errorf("expected the confirmed delete to remove the server, got %d servers", len(servers))
	}
}
