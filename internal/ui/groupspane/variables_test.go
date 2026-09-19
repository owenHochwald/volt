package groupspane

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestGroupsPane_EditServerVariables_SavesToDB is a regression test for
// Phase 1 of "environment variables" (see http.SubstituteVariables): "V"
// on a selected server opens an editable list of its {{name}} placeholder
// values, and ctrl+s persists it - mirroring the existing "H" (headers)
// flow exactly.
func TestGroupsPane_EditServerVariables_SavesToDB(t *testing.T) {
	pane, db := setupTestPane(t)
	if _, err := db.GetOrCreateGroup("api"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.UpsertServer("prod", "https://api.example.com"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.AddGroupServer("api", "prod"); err != nil {
		t.Fatalf("failed to add server to group: %v", err)
	}
	pane.Open()
	pane.target = focusServers

	if item, ok := pane.selectedServer(); !ok || item.server.Name != "prod" {
		t.Fatalf("test setup error: expected \"prod\" selected, got %+v (ok=%v)", item, ok)
	}

	pane = pressKey(t, pane, "V")
	if pane.mode != modeEditServerVariables {
		t.Fatalf("expected \"V\" to open modeEditServerVariables, got %v", pane.mode)
	}
	if !pane.variablesInput.Focused() {
		t.Fatal("expected the variables textarea to be focused")
	}

	pane = typeString(t, pane, "token = abc123")
	pane, cmd := pane.Update(tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	if cmd != nil {
		cmd()
	}
	if pane.mode != modeBrowse {
		t.Fatalf("expected saving to return to browse mode, got %v", pane.mode)
	}

	server, err := db.GetServerByName("prod")
	if err != nil {
		t.Fatalf("failed to reload server: %v", err)
	}
	if server.Variables["token"] != "abc123" {
		t.Errorf("expected \"token\" saved as \"abc123\", got %+v", server.Variables)
	}
	if !strings.Contains(pane.statusMsg, "Updated") {
		t.Errorf("expected a success status message, got %q", pane.statusMsg)
	}
}

// TestGroupsPane_EditServerVariables_EscCancelsWithoutSaving verifies esc
// discards the edit, mirroring the headers form's own esc handling.
func TestGroupsPane_EditServerVariables_EscCancelsWithoutSaving(t *testing.T) {
	pane, db := setupTestPane(t)
	if _, err := db.GetOrCreateGroup("api"); err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	if _, err := db.UpsertServer("prod", "https://api.example.com"); err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	if err := db.AddGroupServer("api", "prod"); err != nil {
		t.Fatalf("failed to add server to group: %v", err)
	}
	pane.Open()
	pane.target = focusServers

	pane = pressKey(t, pane, "V")
	pane = typeString(t, pane, "token = abc123")
	pane, _ = pane.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))

	if pane.mode != modeBrowse {
		t.Fatalf("expected esc to return to browse mode, got %v", pane.mode)
	}
	server, err := db.GetServerByName("prod")
	if err != nil {
		t.Fatalf("failed to reload server: %v", err)
	}
	if len(server.Variables) != 0 {
		t.Errorf("expected nothing saved after esc, got %+v", server.Variables)
	}
}
