package importpane

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/ui/keybindings"
)

func setupTestPane(t *testing.T) ImportPane {
	t.Helper()
	pane := SetupImportPane(nil, keybindings.DefaultKeyMap())
	pane.Open()
	return pane
}

// TestImportPane_TabCyclesThroughForceField verifies the force toggle (the
// TUI equivalent of `volt import -force`) is a real stop in the tab cycle
// alongside path/group/base URL - added as a 4th field, so this guards
// against the cycle length silently staying at 3 and skipping it.
func TestImportPane_TabCyclesThroughForceField(t *testing.T) {
	pane := setupTestPane(t)
	if pane.focusIndex != 0 {
		t.Fatalf("expected to start on field 0, got %d", pane.focusIndex)
	}

	for i, want := range []int{1, 2, 3, 0} {
		pane, _ = pane.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
		if pane.focusIndex != want {
			t.Errorf("tab %d: expected focusIndex=%d, got %d", i+1, want, pane.focusIndex)
		}
	}
}

// TestImportPane_SpaceTogglesForce_OnlyWhenFocused verifies space toggles
// m.force only while the force field is focused - it must not accidentally
// fire while typing a spec path/group name containing a space.
func TestImportPane_SpaceTogglesForce_OnlyWhenFocused(t *testing.T) {
	pane := setupTestPane(t)
	spaceMsg := tea.KeyPressMsg(tea.Key{Code: ' ', Text: " "})

	// Not focused on the toggle yet (still on the path field) - space
	// should type into pathInput, not toggle force.
	pane, _ = pane.Update(spaceMsg)
	if pane.force {
		t.Fatal("expected force to stay false while the path field is focused")
	}

	// Move focus to the force field (index 3).
	for i := 0; i < 3; i++ {
		pane, _ = pane.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	}
	if pane.focusIndex != 3 {
		t.Fatalf("expected focusIndex=3, got %d", pane.focusIndex)
	}

	pane, _ = pane.Update(spaceMsg)
	if !pane.force {
		t.Error("expected space on the focused force field to toggle it on")
	}

	pane, _ = pane.Update(spaceMsg)
	if pane.force {
		t.Error("expected a second space to toggle it back off")
	}
}
