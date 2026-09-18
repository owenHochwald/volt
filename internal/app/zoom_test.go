package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/owenHochwald/Volt/internal/utils"
)

func zKeyMsg() tea.KeyMsg {
	return tea.KeyPressMsg(tea.Key{Code: []rune("z")[0], Text: "z"})
}

// TestToggleZoomResponse_ExpandsAndCollapses is a regression test for a
// real feature request: "z" while focused on the response panel should
// expand it to the right column's full height (hiding the request
// panel), and the same key should collapse it back.
func TestToggleZoomResponse_ExpandsAndCollapses(t *testing.T) {
	m := newTestModel(t)
	m.setFocusedPanel(utils.ResponsePanel)

	model, _ := m.Update(zKeyMsg())
	result, ok := model.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model)
	}
	if !result.zoomedResponse {
		t.Fatal("expected \"z\" to zoom the response panel")
	}
	layout := result.currentLayout()
	if layout.requestHeight != 0 {
		t.Errorf("expected the request panel to get 0 height while zoomed, got %d", layout.requestHeight)
	}
	if layout.responseHeight != layout.contentHeight {
		t.Errorf("expected the response panel to take the whole content height while zoomed, got %d (want %d)", layout.responseHeight, layout.contentHeight)
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
		m := newTestModel(t)
		m.setFocusedPanel(panel)

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
// any means (esc, ...) must restore its size, even if it was left
// zoomed.
func TestZoom_ResetsWhenFocusLeavesResponsePanel(t *testing.T) {
	m := newTestModel(t)
	m.setFocusedPanel(utils.ResponsePanel)
	model, _ := m.Update(zKeyMsg())
	zoomed := model.(Model)
	if !zoomed.zoomedResponse {
		t.Fatal("test setup error: expected \"z\" to zoom the response panel")
	}

	model2, _ := zoomed.Update(escKeyPress())
	result, ok := model2.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model2)
	}
	if result.focusedPanel != utils.SidebarPanel {
		t.Fatalf("test setup error: expected esc to move focus to the sidebar, got %v", result.focusedPanel)
	}
	if result.zoomedResponse {
		t.Error("expected leaving the response panel to un-zoom it")
	}
}

// TestZoom_RestoresRequestPaneSizeWhenLeavingViaPanelSwitch is a
// regression test for a real bug: un-zooming (see
// TestZoom_ResetsWhenFocusLeavesResponsePanel above) cleared
// zoomedResponse but never re-applied the layout - the request pane's
// own Height stayed stuck at the near-zero value applyLayout gave it
// while zoomed (currentLayout sets requestHeight to 0 whenever
// zoomedResponse is true), so switching straight back to it (ctrl+e/
// alt+h - PreviousPanel, not just esc back to the sidebar) showed
// almost nothing of it - just its first field - until some *other*
// resize happened to come along.
func TestZoom_RestoresRequestPaneSizeWhenLeavingViaPanelSwitch(t *testing.T) {
	m := newTestModel(t)
	m.width, m.height = 160, 45
	m.applyLayout(m.currentLayout())
	m.setFocusedPanel(utils.ResponsePanel)

	model, _ := m.Update(zKeyMsg())
	zoomed := model.(Model)
	if !zoomed.zoomedResponse {
		t.Fatal("test setup error: expected \"z\" to zoom the response panel")
	}
	zoomedHeight := zoomed.requestPane.Height

	model2, _ := zoomed.Update(appKeyPress('e', "", tea.ModCtrl)) // PreviousPanel: response -> request
	result, ok := model2.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", model2)
	}
	if result.focusedPanel != utils.RequestPanel {
		t.Fatalf("test setup error: expected ctrl+e to move focus to the request panel, got %v", result.focusedPanel)
	}
	if result.requestPane.Height <= zoomedHeight {
		t.Errorf("expected the request pane's own Height restored above its zoomed value (%d), got %d", zoomedHeight, result.requestPane.Height)
	}
}
