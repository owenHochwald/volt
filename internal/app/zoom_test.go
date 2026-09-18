package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/owenHochwald/Volt/internal/utils"
)

func zKeyMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("z")}
}

// TestToggleZoomResponse_ExpandsAndCollapses is a regression test for a
// real feature request: "z" while focused on the response panel should
// expand it to the right column's full height (hiding the request
// panel), and the same key should collapse it back.
func TestToggleZoomResponse_ExpandsAndCollapses(t *testing.T) {
	m := setupTestModel(t)
	m.focusedPanel = utils.ResponsePanel

	model, _ := m.Update(zKeyMsg())
	result, ok := model.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model)
	}
	if !result.zoomedResponse {
		t.Fatal("expected \"z\" to zoom the response panel")
	}
	if result.requestHeight() != 0 {
		t.Errorf("expected the request panel to get 0 height while zoomed, got %d", result.requestHeight())
	}
	if result.responseHeight() != result.rightColumnHeight()-boxBorderLines {
		t.Errorf("expected the response panel to take the whole right column while zoomed, got %d (want %d)", result.responseHeight(), result.rightColumnHeight()-boxBorderLines)
	}

	model2, _ := result.Update(zKeyMsg())
	result2, ok := model2.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model2)
	}
	if result2.zoomedResponse {
		t.Error("expected a second \"z\" to un-zoom the response panel")
	}
}

// TestToggleZoomResponse_OnlyAppliesWhileResponseFocused verifies "z" is
// a no-op anywhere other than the response panel - it shouldn't zoom a
// panel that isn't even focused.
func TestToggleZoomResponse_OnlyAppliesWhileResponseFocused(t *testing.T) {
	for _, panel := range []utils.Panel{utils.SidebarPanel, utils.RequestPanel} {
		m := setupTestModel(t)
		m.focusedPanel = panel

		model, _ := m.Update(zKeyMsg())
		result, ok := model.(Model)
		if !ok {
			t.Fatalf("expected Model, got %T", model)
		}
		if result.zoomedResponse {
			t.Errorf("expected \"z\" to do nothing while focused on panel %v, but it zoomed", panel)
		}
	}
}

// TestZoom_ResetsWhenFocusLeavesResponsePanel is a regression test for
// the other half of the feature request: leaving the response panel by
// any means (esc, f1, ...) must restore its size, even if it was left
// zoomed.
func TestZoom_ResetsWhenFocusLeavesResponsePanel(t *testing.T) {
	m := setupTestModel(t)
	m.focusedPanel = utils.ResponsePanel
	model, _ := m.Update(zKeyMsg())
	zoomed := model.(Model)
	if !zoomed.zoomedResponse {
		t.Fatal("test setup error: expected \"z\" to zoom the response panel")
	}

	model2, _ := zoomed.Update(escMsg())
	result, ok := model2.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model2)
	}
	if result.focusedPanel != utils.RequestPanel {
		t.Fatalf("test setup error: expected esc to move focus to the request panel, got %v", result.focusedPanel)
	}
	if result.zoomedResponse {
		t.Error("expected leaving the response panel to un-zoom it")
	}
}

// TestZoom_ResetsWhenFocusJumpsAwayViaFKey mirrors the esc test above for
// f1 (FocusSidebar), a different code path that also reassigns
// focusedPanel (see app.Model.focusPanel).
func TestZoom_ResetsWhenFocusJumpsAwayViaFKey(t *testing.T) {
	m := setupTestModel(t)
	m.focusedPanel = utils.ResponsePanel
	model, _ := m.Update(zKeyMsg())
	zoomed := model.(Model)
	if !zoomed.zoomedResponse {
		t.Fatal("test setup error: expected \"z\" to zoom the response panel")
	}

	model2, _ := zoomed.Update(tea.KeyMsg{Type: tea.KeyF1})
	result, ok := model2.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model2)
	}
	if result.focusedPanel != utils.SidebarPanel {
		t.Fatalf("test setup error: expected f1 to move focus to the sidebar, got %v", result.focusedPanel)
	}
	if result.zoomedResponse {
		t.Error("expected jumping away via f1 to un-zoom the response panel")
	}
}
