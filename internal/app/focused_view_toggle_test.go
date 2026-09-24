package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/owenHochwald/Volt/internal/ui/design"
)

func f2KeyMsg() tea.KeyMsg {
	return tea.KeyPressMsg(tea.Key{Code: tea.KeyF2})
}

// TestToggleFocusedView_ForcesSinglePanelRegardlessOfWidth is a regression
// test for a real feature request: "F2" should force the single-panel
// layout even on a terminal wide enough for layoutWide - the same layout
// a small terminal already gets automatically, just chosen deliberately.
func TestToggleFocusedView_ForcesSinglePanelRegardlessOfWidth(t *testing.T) {
	m := newTestModel(t)
	m.width, m.height = 160, 50
	if got := m.currentLayout().mode; got != layoutWide {
		t.Fatalf("test setup error: expected layoutWide on a spacious terminal, got %v", got)
	}

	model, _ := m.Update(f2KeyMsg())
	result, ok := model.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model)
	}

	if !result.forceFocusedView {
		t.Fatal("expected \"F2\" to set forceFocusedView")
	}
	if got := result.currentLayout().mode; got != layoutFocused {
		t.Errorf("expected layoutFocused after F2 on a spacious terminal, got %v", got)
	}

	model2, _ := result.Update(f2KeyMsg())
	result2, ok := model2.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model2)
	}
	if result2.forceFocusedView {
		t.Error("expected a second \"F2\" to turn it back off")
	}
	if got := result2.currentLayout().mode; got != layoutWide {
		t.Errorf("expected layoutWide restored after the second F2, got %v", got)
	}
}

// TestToggleFocusedView_Persists verifies the toggle is saved to
// config.yaml (see design.SaveForceFocusedView) so it survives a restart,
// not just the current session.
func TestToggleFocusedView_Persists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newTestModel(t)

	model, cmd := m.Update(f2KeyMsg())
	if _, ok := model.(Model); !ok {
		t.Fatalf("expected Model, got %T", model)
	}
	if cmd == nil {
		t.Fatal("expected a persist command from toggling F2")
	}
	if _, ok := cmd().(preferenceSavedMsg); !ok {
		t.Fatalf("expected preferenceSavedMsg, got %T", cmd())
	}

	got, err := design.LoadForceFocusedView()
	if err != nil {
		t.Fatalf("LoadForceFocusedView: %v", err)
	}
	if !got {
		t.Error("expected the toggle to have been persisted as true")
	}
}
